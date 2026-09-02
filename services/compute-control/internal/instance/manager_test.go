package instance

import (
	"sync"
	"testing"
	"time"

	"github.com/trade1/compute-control/internal/domain"
	"github.com/trade1/compute-control/internal/events"
	"github.com/trade1/compute-control/internal/pool"
)

// capturePublisher records emitted usage events for assertions.
type capturePublisher struct {
	mu sync.Mutex
	ev []events.ComputeUsage
}

func (c *capturePublisher) PublishUsage(u events.ComputeUsage) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.ev = append(c.ev, u)
}

func (c *capturePublisher) count() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return len(c.ev)
}

// newManager builds a manager over a fresh pool of h100 H100s.
func newManager(h100 int, pub events.Publisher) (*Manager, *pool.Pool) {
	p := pool.New(map[string]int{domain.CreditH100: h100, domain.CreditH200: 0})
	return NewManager(p, "dc-owned-1", pub), p
}

func spec(tenant string) Spec {
	return Spec{TenantID: tenant, GPUType: domain.CreditH100, Count: 1}
}

// TestCreate_RunningWithConnect checks a created instance is running, carries connect info + supply,
// and defaults to the latest stable image.
func TestCreate_RunningWithConnect(t *testing.T) {
	m, _ := newManager(8, &capturePublisher{})
	inst, err := m.Create(spec("t1"), "k1")
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if inst.State != domain.InstanceRunning {
		t.Fatalf("state = %q", inst.State)
	}
	if inst.Connect.SSH == "" || inst.Connect.Jupyter == "" {
		t.Fatalf("missing connect info: %+v", inst.Connect)
	}
	if inst.Image != domain.ImageStable {
		t.Fatalf("default image = %q, want %q", inst.Image, domain.ImageStable)
	}
	if inst.SupplySourceID != "dc-owned-1" {
		t.Fatalf("supply = %q", inst.SupplySourceID)
	}
}

// TestCreate_Idempotent checks a repeated (tenant, key) returns the same instance.
func TestCreate_Idempotent(t *testing.T) {
	m, _ := newManager(8, &capturePublisher{})
	a, _ := m.Create(spec("t1"), "same")
	b, err := m.Create(spec("t1"), "same")
	if err != nil {
		t.Fatalf("create2: %v", err)
	}
	if a.ID != b.ID {
		t.Fatalf("ids differ: %s vs %s", a.ID, b.ID)
	}
	if got := len(m.List("t1", "")); got != 1 {
		t.Fatalf("instance count = %d, want 1", got)
	}
}

// TestCreate_Validation rejects unknown gpu + unknown pinned image.
func TestCreate_Validation(t *testing.T) {
	m, _ := newManager(8, &capturePublisher{})
	bad := spec("t1")
	bad.GPUType = "gpu_x"
	if _, err := m.Create(bad, ""); err != ErrUnknownGPU {
		t.Fatalf("err = %v, want ErrUnknownGPU", err)
	}
	img := spec("t1")
	img.Image = "1trade-ml-stack-1999.01"
	if _, err := m.Create(img, ""); err != ErrBadImage {
		t.Fatalf("err = %v, want ErrBadImage", err)
	}
}

// TestCreate_Capacity rejects when the pool is exhausted.
func TestCreate_Capacity(t *testing.T) {
	m, _ := newManager(1, &capturePublisher{})
	if _, err := m.Create(spec("t1"), "k1"); err != nil {
		t.Fatalf("first create: %v", err)
	}
	if _, err := m.Create(spec("t2"), "k2"); err != ErrCapacity {
		t.Fatalf("err = %v, want ErrCapacity", err)
	}
}

// TestStop_ReleasesAndMeters checks stop frees the GPUs and emits one final usage event.
func TestStop_ReleasesAndMeters(t *testing.T) {
	pub := &capturePublisher{}
	m, p := newManager(2, pub)
	clock := time.Date(2026, 6, 2, 12, 0, 0, 0, time.UTC)
	m.now = func() time.Time { return clock }
	inst, _ := m.Create(spec("t1"), "k1")

	clock = clock.Add(2 * time.Hour) // 2h of runtime
	stopped, err := m.Stop("t1", inst.ID)
	if err != nil {
		t.Fatalf("stop: %v", err)
	}
	if stopped.State != domain.InstanceStopped {
		t.Fatalf("state = %q", stopped.State)
	}
	if p.Available(domain.CreditH100) != 2 {
		t.Fatalf("available after stop = %d, want 2", p.Available(domain.CreditH100))
	}
	if pub.count() != 1 {
		t.Fatalf("usage events = %d, want 1", pub.count())
	}
	if got := pub.ev[0].Units; got != "2.000000" { // 1 GPU × 2h
		t.Fatalf("units = %q, want 2.000000", got)
	}
	if pub.ev[0].CreditType != domain.CreditH100 {
		t.Fatalf("credit_type = %q", pub.ev[0].CreditType)
	}
}

// TestStartStop_StateGuards checks the state machine rejects illegal transitions.
func TestStartStop_StateGuards(t *testing.T) {
	m, _ := newManager(2, &capturePublisher{})
	inst, _ := m.Create(spec("t1"), "k1")
	if _, err := m.Start("t1", inst.ID); err != ErrNotStartable {
		t.Fatalf("start running err = %v, want ErrNotStartable", err)
	}
	if _, err := m.Stop("t1", inst.ID); err != nil {
		t.Fatalf("stop: %v", err)
	}
	if _, err := m.Stop("t1", inst.ID); err != ErrNotStoppable {
		t.Fatalf("stop stopped err = %v, want ErrNotStoppable", err)
	}
	back, err := m.Start("t1", inst.ID)
	if err != nil {
		t.Fatalf("start: %v", err)
	}
	if back.State != domain.InstanceRunning || back.Connect.SSH == "" {
		t.Fatalf("restarted = %+v", back)
	}
}

// TestStart_ReReserves checks a restart fails when capacity was taken while stopped.
func TestStart_ReReserves(t *testing.T) {
	m, _ := newManager(1, &capturePublisher{})
	a, _ := m.Create(spec("t1"), "k1")
	if _, err := m.Stop("t1", a.ID); err != nil {
		t.Fatalf("stop: %v", err)
	}
	// Another tenant grabs the freed GPU.
	if _, err := m.Create(spec("t2"), "k2"); err != nil {
		t.Fatalf("t2 create: %v", err)
	}
	if _, err := m.Start("t1", a.ID); err != ErrCapacity {
		t.Fatalf("restart err = %v, want ErrCapacity", err)
	}
}

// TestDelete_TerminatesAndMeters checks delete frees GPUs + meters the final interval.
func TestDelete_TerminatesAndMeters(t *testing.T) {
	pub := &capturePublisher{}
	m, p := newManager(2, pub)
	clock := time.Date(2026, 6, 2, 12, 0, 0, 0, time.UTC)
	m.now = func() time.Time { return clock }
	inst, _ := m.Create(spec("t1"), "k1")
	clock = clock.Add(30 * time.Minute)
	del, err := m.Delete("t1", inst.ID)
	if err != nil {
		t.Fatalf("delete: %v", err)
	}
	if del.State != domain.InstanceTerminated {
		t.Fatalf("state = %q", del.State)
	}
	if p.Available(domain.CreditH100) != 2 {
		t.Fatalf("available after delete = %d", p.Available(domain.CreditH100))
	}
	if pub.count() != 1 || pub.ev[0].Units != "0.500000" { // 1 GPU × 0.5h
		t.Fatalf("usage = %+v", pub.ev)
	}
}

// TestMeterTick_PerInterval checks the ticker meters running instances and advances the watermark
// (no double-count): two ticks one hour apart emit one event each.
func TestMeterTick_PerInterval(t *testing.T) {
	pub := &capturePublisher{}
	m, _ := newManager(2, pub)
	clock := time.Date(2026, 6, 2, 12, 0, 0, 0, time.UTC)
	m.now = func() time.Time { return clock }
	if _, err := m.Create(spec("t1"), "k1"); err != nil {
		t.Fatalf("create: %v", err)
	}

	clock = clock.Add(time.Hour)
	m.MeterTick()
	clock = clock.Add(time.Hour)
	m.MeterTick()
	if pub.count() != 2 {
		t.Fatalf("events = %d, want 2", pub.count())
	}
	for _, e := range pub.ev {
		if e.Units != "1.000000" {
			t.Fatalf("interval units = %q, want 1.000000", e.Units)
		}
	}
}

// TestList_StateFilter + tenant scoping.
func TestList_StateFilter(t *testing.T) {
	m, _ := newManager(4, &capturePublisher{})
	a, _ := m.Create(spec("t1"), "k1")
	if _, err := m.Create(spec("t1"), "k2"); err != nil {
		t.Fatalf("create: %v", err)
	}
	if _, err := m.Stop("t1", a.ID); err != nil {
		t.Fatalf("stop: %v", err)
	}
	if got := len(m.List("t1", domain.InstanceRunning)); got != 1 {
		t.Fatalf("running = %d, want 1", got)
	}
	if got := len(m.List("t1", domain.InstanceStopped)); got != 1 {
		t.Fatalf("stopped = %d, want 1", got)
	}
	if got := len(m.List("t2", "")); got != 0 {
		t.Fatalf("t2 sees %d, want 0", got)
	}
}

// TestGet_TenantScoped checks cross-tenant reads are forbidden.
func TestGet_TenantScoped(t *testing.T) {
	m, _ := newManager(2, &capturePublisher{})
	inst, _ := m.Create(spec("t1"), "k1")
	if _, err := m.Get("t2", inst.ID); err != ErrForbidden {
		t.Fatalf("err = %v, want ErrForbidden", err)
	}
}
