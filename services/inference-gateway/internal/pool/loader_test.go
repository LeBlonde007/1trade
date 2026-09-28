package pool

import (
	"context"
	"errors"
	"net/http"
	"os/exec"
	"path/filepath"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/trade1/inference-gateway/internal/model"
	"github.com/trade1/inference-gateway/internal/routing"
)

// stubCommand is the CPU runtime stub (the same contract as vLLM) as a loader command.
func stubCommand(t *testing.T) []string {
	t.Helper()
	py, err := exec.LookPath("python3")
	if err != nil {
		t.Skip("python3 not available")
	}
	path, _ := filepath.Abs("../../../inference-runtime/stub/server.py")
	return []string{py, path}
}

// alive reports whether a runtime answers at url.
func alive(url string) bool {
	c := &http.Client{Timeout: time.Second}
	resp, err := c.Get(url + "/healthz")
	if err != nil {
		return false
	}
	resp.Body.Close()
	return resp.StatusCode == http.StatusOK
}

// TestProcessLoaderRealRuntime: a real runtime process is started, answers ready and serves its
// model, then is stopped; a process that dies is reported; one that never gets ready is killed.
func TestProcessLoaderRealRuntime(t *testing.T) {
	var exited sync.Map
	l := &ProcessLoader{Command: stubCommand(t), ReadyTimeout: 20 * time.Second, OnExit: func(m, g string) { exited.Store(m+"@"+g, true) }}
	t.Cleanup(l.StopAll)
	url, err := l.Load(context.Background(), "qwen2.5-1.5b", "g0")
	if err != nil {
		t.Fatal(err)
	}
	res, err := client(url).Chat(context.Background(), model.ChatRequest{Model: "qwen2.5-1.5b", Messages: []model.Message{{Role: "user", Content: "hi"}}})
	if err != nil || res.Content == "" {
		t.Fatalf("chat on the loaded process: %v %+v", err, res)
	}
	if err := l.Unload(context.Background(), "qwen2.5-1.5b", "g0", url); err != nil || alive(url) {
		t.Fatalf("unload: %v, still alive %v", err, alive(url))
	}
	if _, ok := exited.Load("qwen2.5-1.5b@g0"); ok {
		t.Fatal("an unload was reported as a crash")
	}

	// A crash is reported.
	url, err = l.Load(context.Background(), "m-crash", "g1")
	if err != nil {
		t.Fatal(err)
	}
	l.mu.Lock()
	_ = l.procs[url].cmd.Process.Kill()
	l.mu.Unlock()
	for i := 0; ; i++ {
		if _, ok := exited.Load("m-crash@g1"); ok {
			break
		}
		if i > 100 {
			t.Fatal("crash not reported")
		}
		time.Sleep(20 * time.Millisecond)
	}

	// A command that exits at once fails the load quickly.
	bad := &ProcessLoader{Command: []string{"false"}, ReadyTimeout: 20 * time.Second}
	start := time.Now()
	if _, err := bad.Load(context.Background(), "x", "g0"); err == nil || time.Since(start) > 5*time.Second {
		t.Fatalf("a dead process: %v after %s", err, time.Since(start))
	}
}

// TestHotSwapWithRealProcesses: the static model is started at boot; a cold tail model is loaded on
// demand and served; a second tail model that does not fit evicts the first (its process stops).
func TestHotSwapWithRealProcesses(t *testing.T) {
	s := Spec{
		MinResidency: "1ms",
		LoadWait:     "30s",
		GPUs:         []GPUSpec{{ID: "g0", VRAMMiB: 100000, Device: "0"}},
		Profiles: []ProfSpec{
			{Model: "llama-3.1-8b", PeakVRAMMiB: 50000, MeasuredAt: measured, Static: true},
			{Model: "qwen2.5-1.5b", PeakVRAMMiB: 30000, MeasuredAt: measured},
			{Model: "gemma-2-2b", PeakVRAMMiB: 30000, MeasuredAt: measured},
		},
		Loader: &LoaderSpec{Kind: "process", Command: stubCommand(t), ReadyTimeout: "20s"},
	}
	b, err := New(s, nil, client)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(b.loader.(*ProcessLoader).StopAll)
	if !b.warm("llama-3.1-8b") {
		t.Fatal("the static model was not started at boot")
	}
	chat := func(m string) error {
		_, err := b.Chat(context.Background(), model.ChatRequest{Model: m, Messages: []model.Message{{Role: "user", Content: "hi"}}})
		return err
	}
	if err := chat("qwen2.5-1.5b"); err != nil {
		t.Fatalf("cold tail model: %v", err)
	}
	var qwenURL string
	for _, r := range b.router.Replicas() {
		if r.Model == "qwen2.5-1.5b" {
			qwenURL = r.URL
		}
	}
	if !alive(qwenURL) {
		t.Fatal("qwen process not running")
	}
	time.Sleep(5 * time.Millisecond) // past min_residency
	if err := chat("gemma-2-2b"); err != nil {
		t.Fatalf("second tail model: %v", err)
	}
	if alive(qwenURL) || b.warm("qwen2.5-1.5b") || !b.warm("gemma-2-2b") || !b.warm("llama-3.1-8b") {
		t.Fatalf("eviction: qwen alive %v, replicas %+v", alive(qwenURL), b.router.Replicas())
	}
}

// fakeLoader counts loads; each takes delay; fail makes loads fail; unloadFail makes evictions fail.
type fakeLoader struct {
	loads      atomic.Int64
	unloads    atomic.Int64
	delay      time.Duration
	fail       bool
	unloadFail bool
	url        string
}

// Load pretends to start a process.
func (f *fakeLoader) Load(_ context.Context, _, _ string) (string, error) {
	f.loads.Add(1)
	time.Sleep(f.delay)
	if f.fail {
		return "", errors.New("image pull failed")
	}
	return f.url, nil
}

// Unload pretends to stop one.
func (f *fakeLoader) Unload(context.Context, string, string, string) error {
	f.unloads.Add(1)
	if f.unloadFail {
		return errors.New("process will not stop")
	}
	return nil
}

// fakeSpec is one GPU with a running static model and room for exactly one of two tail models.
func fakeSpec(u string) Spec {
	return Spec{
		MinResidency: "1ms",
		GPUs:         []GPUSpec{{ID: "g0", VRAMMiB: 100000}},
		Profiles: []ProfSpec{
			{Model: "llama-3.1-8b", PeakVRAMMiB: 50000, MeasuredAt: measured, Static: true},
			{Model: "qwen2.5-1.5b", PeakVRAMMiB: 30000, MeasuredAt: measured},
			{Model: "gemma-2-2b", PeakVRAMMiB: 30000, MeasuredAt: measured},
		},
		Replicas: []ReplSpec{{Model: "llama-3.1-8b", GPU: "g0", URL: u}},
	}
}

// TestColdLoadIsSingleFlight: many requests for one cold model start one load; each waits for it
// (load_wait) and is served.
func TestColdLoadIsSingleFlight(t *testing.T) {
	p := newPod(t, "qwen", nil)
	s := fakeSpec(p.URL)
	s.LoadWait = "5s"
	b, err := New(s, nil, client)
	if err != nil {
		t.Fatal(err)
	}
	fl := &fakeLoader{delay: 200 * time.Millisecond, url: p.URL}
	b.SetLoader(fl)
	var wg sync.WaitGroup
	var failed atomic.Int64
	for range 8 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if _, err := b.Chat(context.Background(), model.ChatRequest{Model: "qwen2.5-1.5b"}); err != nil {
				failed.Add(1)
			}
		}()
	}
	wg.Wait()
	if fl.loads.Load() != 1 || failed.Load() != 0 {
		t.Fatalf("loads %d (want 1), failed requests %d", fl.loads.Load(), failed.Load())
	}
}

// TestNoWaitAnswersColdButLoads: with load_wait 0 the request gets ErrModelCold at once, the load runs
// anyway, and the next request is served.
func TestNoWaitAnswersColdButLoads(t *testing.T) {
	p := newPod(t, "qwen", nil)
	b, err := New(fakeSpec(p.URL), nil, client)
	if err != nil {
		t.Fatal(err)
	}
	b.SetLoader(&fakeLoader{url: p.URL})
	if _, err := b.Chat(context.Background(), model.ChatRequest{Model: "qwen2.5-1.5b"}); !errors.Is(err, model.ErrModelCold) {
		t.Fatalf("first request: %v", err)
	}
	for i := 0; !b.warm("qwen2.5-1.5b"); i++ {
		if i > 100 {
			t.Fatal("background load never finished")
		}
		time.Sleep(10 * time.Millisecond)
	}
	if _, err := b.Chat(context.Background(), model.ChatRequest{Model: "qwen2.5-1.5b"}); err != nil {
		t.Fatalf("after the load: %v", err)
	}
}

// TestFailedLoadReleasesEverything: a load that fails frees the VRAM it claimed; an eviction that fails
// abandons the load and puts the victim back into service.
func TestFailedLoadReleasesEverything(t *testing.T) {
	p := newPod(t, "x", nil)
	s := fakeSpec(p.URL)
	s.LoadWait = "5s"
	b, err := New(s, nil, client)
	if err != nil {
		t.Fatal(err)
	}
	fl := &fakeLoader{fail: true, url: p.URL}
	b.SetLoader(fl)
	if _, err := b.Chat(context.Background(), model.ChatRequest{Model: "qwen2.5-1.5b"}); !errors.Is(err, model.ErrModelCold) {
		t.Fatalf("failed load: %v", err)
	}
	for _, r := range b.router.Replicas() {
		if r.Loading {
			t.Fatalf("a failed load kept its VRAM: %+v", b.router.Replicas())
		}
	}
	// qwen loads; gemma must evict it; the eviction fails → qwen serves again, gemma is not loaded.
	fl.fail = false
	if _, err := b.Chat(context.Background(), model.ChatRequest{Model: "qwen2.5-1.5b"}); err != nil {
		t.Fatal(err)
	}
	time.Sleep(5 * time.Millisecond)
	fl.unloadFail = true
	if _, err := b.Chat(context.Background(), model.ChatRequest{Model: "gemma-2-2b"}); !errors.Is(err, model.ErrModelCold) {
		t.Fatalf("load behind a failed eviction: %v", err)
	}
	if b.warm("gemma-2-2b") || !b.warm("qwen2.5-1.5b") {
		t.Fatalf("after a failed eviction: %+v", b.router.Replicas())
	}
	if _, err := b.Chat(context.Background(), model.ChatRequest{Model: "qwen2.5-1.5b"}); err != nil {
		t.Fatalf("the victim should serve again: %v", err)
	}
}

// TestDrainingReplicaTakesNoRequests: once chosen for eviction a replica gets no new request, and a
// second load of the same model is refused while one runs.
func TestDrainingReplicaTakesNoRequests(t *testing.T) {
	r := routing.NewRouter(routing.Config{MinResidency: time.Nanosecond}, []routing.GPU{{ID: "g0", VRAMMiB: 100000}}, []routing.Profile{
		{Model: "a", PeakVRAMMiB: 60000, MeasuredAt: measured}, {Model: "b", PeakVRAMMiB: 60000, MeasuredAt: measured},
	}, nil)
	if err := r.Loaded("a", "g0", "http://a"); err != nil {
		t.Fatal(err)
	}
	time.Sleep(time.Millisecond)
	ld, err := r.BeginLoad("b")
	if err != nil || len(ld.Evict) != 1 || ld.Evict[0].Model != "a" {
		t.Fatalf("plan: %+v %v", ld, err)
	}
	if d, _ := r.Route("a"); d.Replica != nil {
		t.Fatal("a draining replica took a request")
	}
	if _, err := r.BeginLoad("b"); !errors.Is(err, routing.ErrLoading) {
		t.Fatalf("second load of b: %v", err)
	}
	if d, _ := r.Route("b"); !d.Loading {
		t.Fatal("route to a loading model should say it is loading")
	}
	r.Unloaded("a", "g0")
	r.FinishLoad("b", "g0", "http://b")
	if d, _ := r.Route("b"); d.Replica == nil || d.Replica.URL != "http://b" {
		t.Fatalf("after the load: %+v", d)
	}
	if _, err := r.BeginLoad("b"); !errors.Is(err, routing.ErrWarm) {
		t.Fatalf("load of a warm model: %v", err)
	}
}
