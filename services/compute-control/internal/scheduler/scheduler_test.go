package scheduler

import (
	"sync"
	"testing"

	"github.com/trade1/compute-control/internal/domain"
	"github.com/trade1/compute-control/internal/events"
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

// validSpec is a baseline single-pod inference job for tenant t.
func validSpec(t string) JobSpec {
	return JobSpec{TenantID: t, WorkloadClass: domain.ClassInference, GPUType: domain.CreditH100, GPUs: 1, Pods: 1}
}

// TestSubmit_PlacesAndAttributesSupply checks a valid job is placed, running, and tagged with the
// configured supply source.
func TestSubmit_PlacesAndAttributesSupply(t *testing.T) {
	s := NewMock(8, 0, "dc-owned-1", &capturePublisher{})
	job, err := s.Submit(validSpec("t1"), "k1")
	if err != nil {
		t.Fatalf("submit: %v", err)
	}
	if job.Status != domain.StatusRunning {
		t.Fatalf("status = %q, want running", job.Status)
	}
	if job.SupplySourceID != "dc-owned-1" {
		t.Fatalf("supply = %q", job.SupplySourceID)
	}
}

// TestSubmit_Idempotent checks a repeated (tenant, key) returns the same job, never a second placement.
func TestSubmit_Idempotent(t *testing.T) {
	s := NewMock(8, 0, "dc-owned-1", &capturePublisher{})
	a, _ := s.Submit(validSpec("t1"), "same")
	b, err := s.Submit(validSpec("t1"), "same")
	if err != nil {
		t.Fatalf("submit2: %v", err)
	}
	if a.ID != b.ID {
		t.Fatalf("ids differ: %s vs %s", a.ID, b.ID)
	}
	if got := len(s.List("t1")); got != 1 {
		t.Fatalf("job count = %d, want 1", got)
	}
}

// TestSubmit_GangCapacity checks an all-or-nothing gang larger than the pool is rejected and frees
// nothing; the next fitting gang still succeeds.
func TestSubmit_GangCapacity(t *testing.T) {
	s := NewMock(4, 0, "dc-owned-1", &capturePublisher{})
	big := validSpec("t1")
	big.GPUs, big.Pods = 2, 3 // 6 > 4
	if _, err := s.Submit(big, "k1"); err != ErrCapacity {
		t.Fatalf("err = %v, want ErrCapacity", err)
	}
	fits := validSpec("t1")
	fits.GPUs, fits.Pods = 2, 2 // 4 == 4
	if _, err := s.Submit(fits, "k2"); err != nil {
		t.Fatalf("fitting gang rejected: %v", err)
	}
	if _, err := s.Submit(validSpec("t1"), "k3"); err != ErrCapacity {
		t.Fatalf("err = %v, want ErrCapacity (pool full)", err)
	}
}

// TestCancel_FreesAndMeters checks cancel frees capacity and emits exactly one usage event tagged
// with the tenant + credit type.
func TestCancel_FreesAndMeters(t *testing.T) {
	pub := &capturePublisher{}
	s := NewMock(2, 0, "dc-owned-1", pub)
	job, _ := s.Submit(validSpec("t1"), "k1")
	if _, err := s.Cancel("t1", job.ID); err != nil {
		t.Fatalf("cancel: %v", err)
	}
	if len(pub.ev) != 1 {
		t.Fatalf("usage events = %d, want 1", len(pub.ev))
	}
	if pub.ev[0].CreditType != domain.CreditH100 || pub.ev[0].TenantID != "t1" {
		t.Fatalf("event = %+v", pub.ev[0])
	}
	// capacity is freed
	h100, _ := s.Capacity()
	if h100 != 2 {
		t.Fatalf("available after cancel = %d, want 2", h100)
	}
}

// TestGet_TenantScoped checks one tenant cannot read another's job.
func TestGet_TenantScoped(t *testing.T) {
	s := NewMock(8, 0, "dc-owned-1", &capturePublisher{})
	job, _ := s.Submit(validSpec("t1"), "k1")
	if _, err := s.Get("t2", job.ID); err != ErrForbidden {
		t.Fatalf("cross-tenant get err = %v, want ErrForbidden", err)
	}
}

// TestSubmit_Validation rejects unknown gpu/class and non-positive sizes.
func TestSubmit_Validation(t *testing.T) {
	s := NewMock(8, 0, "dc-owned-1", &capturePublisher{})
	cases := []struct {
		name string
		mut  func(*JobSpec)
		want error
	}{
		{"bad gpu", func(sp *JobSpec) { sp.GPUType = "gpu_x" }, ErrUnknownGPU},
		{"bad class", func(sp *JobSpec) { sp.WorkloadClass = "mining" }, ErrUnknownClass},
		{"zero gpus", func(sp *JobSpec) { sp.GPUs = 0 }, ErrBadRequest},
	}
	for _, c := range cases {
		sp := validSpec("t1")
		c.mut(&sp)
		if _, err := s.Submit(sp, ""); err != c.want {
			t.Fatalf("%s: err = %v, want %v", c.name, err, c.want)
		}
	}
}

// TestQuota_ReflectsUsage checks per-tier quota shows the tenant's in-use GPUs and the free pool.
func TestQuota_ReflectsUsage(t *testing.T) {
	s := NewMock(8, 0, "dc-owned-1", &capturePublisher{})
	sp := validSpec("t1")
	sp.GPUs, sp.Pods = 2, 2 // 4 in use
	if _, err := s.Submit(sp, "k1"); err != nil {
		t.Fatalf("submit: %v", err)
	}
	for _, q := range s.Quotas("t1") {
		if q.GPUType != domain.CreditH100 {
			continue
		}
		if q.InUse != 4 || q.RemainingPool != 4 || q.Capacity != 8 {
			t.Fatalf("h100 quota = %+v", q)
		}
	}
}
