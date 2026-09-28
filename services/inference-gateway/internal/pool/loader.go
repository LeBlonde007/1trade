package pool

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"sync"
	"syscall"
	"time"
)

// Loader starts and stops single-model runtime processes (F11 hot-swap). Load returns the URL of a
// process that is serving model on gpu (its /readyz answered 200); Unload stops the one at url.
type Loader interface {
	Load(ctx context.Context, model, gpu string) (url string, err error)
	Unload(ctx context.Context, model, gpu, url string) error
}

// ProcessLoader runs each replica as a local process — the runtime (vLLM, or the CPU stub in dev)
// started from Command. It is the loader for a single GPU host; on Kubernetes the same interface is
// backed by the compute control plane. Command is argv (no shell); "{model}", "{port}", "{gpu}" and
// "{device}" are substituted, and the process also gets MODEL_ID, PORT and CUDA_VISIBLE_DEVICES.
type ProcessLoader struct {
	Command      []string
	Devices      map[string]string // GPU id → CUDA device index
	Host         string            // where the processes listen (default 127.0.0.1)
	ReadyTimeout time.Duration     // how long a model may take to load (default 10 min)
	// OnExit is told when a replica's process dies without being unloaded, so it stops being routed.
	OnExit func(model, gpu string)

	mu    sync.Mutex
	procs map[string]*proc // url → process
}

// proc is one running replica process.
type proc struct {
	cmd      *exec.Cmd
	done     chan struct{} // closed when the process has exited
	stopping bool
}

// host returns the listen host.
func (l *ProcessLoader) host() string {
	if l.Host == "" {
		return "127.0.0.1"
	}
	return l.Host
}

// Load starts a process for model on gpu and waits until it is ready. On timeout or early exit the
// process is killed and an error returned: a half-started model is never routed.
func (l *ProcessLoader) Load(ctx context.Context, model, gpu string) (string, error) {
	if len(l.Command) == 0 {
		return "", errors.New("pool: loader has no command")
	}
	port, err := freePort(ctx, l.host())
	if err != nil {
		return "", err
	}
	device := l.Devices[gpu]
	repl := strings.NewReplacer("{model}", model, "{port}", fmt.Sprint(port), "{gpu}", gpu, "{device}", device)
	argv := make([]string, len(l.Command))
	for i, a := range l.Command {
		argv[i] = repl.Replace(a)
	}
	// The process outlives the request that caused the load: only Unload stops it.
	cmd := exec.CommandContext(context.WithoutCancel(ctx), argv[0], argv[1:]...) //nolint:gosec // argv is operator configuration; model ids come from the measured profiles
	cmd.Env = append(os.Environ(), "MODEL_ID="+model, fmt.Sprintf("PORT=%d", port), "CUDA_VISIBLE_DEVICES="+device)
	cmd.Stdout, cmd.Stderr = os.Stderr, os.Stderr
	if err := cmd.Start(); err != nil {
		return "", fmt.Errorf("pool: start %s: %w", model, err)
	}
	url := fmt.Sprintf("http://%s:%d", l.host(), port)
	p := &proc{cmd: cmd, done: make(chan struct{})}
	l.mu.Lock()
	if l.procs == nil {
		l.procs = map[string]*proc{}
	}
	l.procs[url] = p
	l.mu.Unlock()
	go l.reap(model, gpu, url, p)

	timeout := l.ReadyTimeout
	if timeout <= 0 {
		timeout = 10 * time.Minute
	}
	if err := waitReady(ctx, url, p.done, timeout); err != nil {
		_ = l.Unload(context.WithoutCancel(ctx), model, gpu, url)
		return "", fmt.Errorf("pool: %s not ready: %w", model, err)
	}
	return url, nil
}

// reap waits for a process to exit and reports an unexpected exit.
func (l *ProcessLoader) reap(model, gpu, url string, p *proc) {
	err := p.cmd.Wait()
	close(p.done)
	l.mu.Lock()
	expected := p.stopping
	delete(l.procs, url)
	l.mu.Unlock()
	if !expected {
		slog.Warn("runtime process exited", "model", model, "gpu", gpu, "err", err)
		if l.OnExit != nil {
			l.OnExit(model, gpu)
		}
	}
}

// Unload stops the process at url: SIGTERM, then SIGKILL after 10 s. Stopping one that is already
// gone succeeds.
func (l *ProcessLoader) Unload(ctx context.Context, _, _, url string) error {
	l.mu.Lock()
	p, ok := l.procs[url]
	if ok {
		p.stopping = true
	}
	l.mu.Unlock()
	if !ok {
		return nil
	}
	_ = p.cmd.Process.Signal(syscall.SIGTERM)
	select {
	case <-p.done:
		return nil
	case <-time.After(10 * time.Second):
	case <-ctx.Done():
	}
	if err := p.cmd.Process.Kill(); err != nil && !errors.Is(err, os.ErrProcessDone) {
		return fmt.Errorf("pool: kill: %w", err)
	}
	<-p.done
	return nil
}

// StopAll stops every process (gateway shutdown).
func (l *ProcessLoader) StopAll() {
	l.mu.Lock()
	urls := make([]string, 0, len(l.procs))
	for u := range l.procs {
		urls = append(urls, u)
	}
	l.mu.Unlock()
	for _, u := range urls {
		_ = l.Unload(context.Background(), "", "", u)
	}
}

// freePort asks the OS for an unused TCP port on host.
func freePort(ctx context.Context, host string) (int, error) {
	ln, err := (&net.ListenConfig{}).Listen(ctx, "tcp", net.JoinHostPort(host, "0"))
	if err != nil {
		return 0, fmt.Errorf("pool: free port: %w", err)
	}
	defer ln.Close()
	return ln.Addr().(*net.TCPAddr).Port, nil
}

// waitReady polls url/readyz until it answers 200, the process exits, or the timeout passes.
func waitReady(ctx context.Context, url string, exited <-chan struct{}, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	client := &http.Client{Timeout: 2 * time.Second}
	for {
		req, _ := http.NewRequestWithContext(ctx, http.MethodGet, url+"/readyz", nil)
		if resp, err := client.Do(req); err == nil {
			resp.Body.Close()
			if resp.StatusCode == http.StatusOK {
				return nil
			}
		}
		select {
		case <-exited:
			return errors.New("process exited while loading")
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(200 * time.Millisecond):
		}
		if time.Now().After(deadline) {
			return fmt.Errorf("not ready after %s", timeout)
		}
	}
}
