// Package scheduler places gang-scheduled GPU jobs and tracks their lifecycle. The Scheduler
// interface is the seam between the control plane and the real stack (Kueue admission + Volcano
// gang-scheduling + the NVIDIA GPU Operator); MockScheduler is the in-memory backend that lets the
// whole service build, run, and be exercised end-to-end with zero GPUs (the M2 "mock-GPU mode").
package scheduler

import (
	"errors"
	"sort"
	"sync"
	"time"

	"github.com/trade1/compute-control/internal/domain"
	"github.com/trade1/compute-control/internal/events"
	"github.com/trade1/compute-control/internal/metrics"
	"github.com/trade1/compute-control/internal/pool"
	"github.com/google/uuid"
)

// Submit-time errors surfaced to the API layer (mapped to 4xx).
var (
	ErrBadRequest    = errors.New("invalid job request")
	ErrUnknownGPU    = errors.New("unknown gpu type")
	ErrUnknownClass  = errors.New("unknown workload class")
	ErrCapacity      = errors.New("insufficient gpu capacity")
	ErrNotFound      = errors.New("job not found")
	ErrForbidden     = errors.New("not your job")
	ErrNotCancelable = errors.New("job not in a cancelable state")
)

// JobSpec is a validated request to schedule a gang of pods, each requesting GPUs of one tier.
type JobSpec struct {
	TenantID      string
	SubAccountID  string
	IsPaper       bool
	WorkloadClass string
	GPUType       string
	GPUs          int    // GPUs per pod
	Pods          int    // gang size (all-or-nothing)
	Reserved      bool   // true == reserved capacity (billed even while pending), false == on-demand
	ReferenceID   string // caller correlation id (e.g. inference deployment id); used for idempotency
}

// TotalGPUs is the all-or-nothing gang footprint.
func (s JobSpec) TotalGPUs() int { return s.GPUs * s.Pods }

// Quota is one tenant's GPU usage against capacity for a single tier.
type Quota struct {
	GPUType       string
	Capacity      int
	InUse         int // GPUs this tenant currently holds (scheduled/running)
	RemainingPool int // cluster-wide free GPUs of this tier
}

// Scheduler places and tracks jobs. Implementations must be safe for concurrent use.
type Scheduler interface {
	// Submit places (or returns the existing, on idempotency key) a job for the principal.
	Submit(spec JobSpec, idempotencyKey string) (domain.Job, error)
	// Get returns one job, scoped to the tenant.
	Get(tenantID, jobID string) (domain.Job, error)
	// List returns the tenant's jobs, newest first.
	List(tenantID string) []domain.Job
	// Cancel transitions a pending/running job to cancelled and frees its GPUs.
	Cancel(tenantID, jobID string) (domain.Job, error)
	// Quotas returns per-tier capacity/usage for the tenant.
	Quotas(tenantID string) []Quota
	// Instances returns the tenant's currently-live placements (scheduled/running jobs).
	Instances(tenantID string) []domain.Job
	// Capacity returns (h100Available, h200Available) cluster-wide — feeds the catalog.
	Capacity() (int, int)
}

// idemKey scopes idempotency to a tenant so two tenants can reuse the same key value.
type idemKey struct {
	tenant string
	key    string
}

// MockScheduler is an in-memory Scheduler: it admits a gang only if the whole footprint fits the
// shared GPU pool for its tier (the same pool customer instances draw from), attributes every
// placement to the configured supply source, meters nothing until a job is cancelled/completed, and
// is idempotent on (tenant, Idempotency-Key).
type MockScheduler struct {
	mu       sync.Mutex
	jobs     map[string]*domain.Job // jobID → job
	idem     map[idemKey]string     // (tenant, key) → jobID
	pool     *pool.Pool             // shared GPU capacity (jobs + instances)
	supplyID string
	pub      events.Publisher
	now      func() time.Time
}

// NewMock builds a MockScheduler over a fresh private pool of the given per-tier capacity. Use this
// when the scheduler owns all the GPUs (e.g. unit tests with no instance manager).
func NewMock(h100, h200 int, supplyID string, pub events.Publisher) *MockScheduler {
	return NewMockWithPool(pool.New(map[string]int{domain.CreditH100: h100, domain.CreditH200: h200}), supplyID, pub)
}

// NewMockWithPool builds a MockScheduler sharing an existing pool — so jobs and customer instances
// contend for the same GPUs (the F13 wiring).
func NewMockWithPool(p *pool.Pool, supplyID string, pub events.Publisher) *MockScheduler {
	return &MockScheduler{
		jobs:     make(map[string]*domain.Job),
		idem:     make(map[idemKey]string),
		pool:     p,
		supplyID: supplyID,
		pub:      pub,
		now:      time.Now,
	}
}

// tenantInUse returns GPUs of a tier currently held by one tenant's live jobs (for quota display).
// Caller must hold m.mu.
func (m *MockScheduler) tenantInUse(gpuType, tenantID string) int {
	n := 0
	for _, j := range m.jobs {
		if j.GPUType != gpuType || j.TenantID != tenantID {
			continue
		}
		if j.Status == domain.StatusScheduled || j.Status == domain.StatusRunning {
			n += j.TotalGPUs()
		}
	}
	return n
}

// Submit validates the spec, enforces capacity, and places the gang. Idempotent: a repeated
// (tenant, idempotencyKey) returns the original job rather than double-scheduling.
func (m *MockScheduler) Submit(spec JobSpec, idempotencyKey string) (domain.Job, error) {
	if spec.GPUs < 1 || spec.Pods < 1 {
		return domain.Job{}, ErrBadRequest
	}
	if !domain.ValidGPUType(spec.GPUType) {
		return domain.Job{}, ErrUnknownGPU
	}
	if !domain.ValidWorkloadClass(spec.WorkloadClass) {
		return domain.Job{}, ErrUnknownClass
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	if idempotencyKey != "" {
		if id, ok := m.idem[idemKey{spec.TenantID, idempotencyKey}]; ok {
			if j, ok := m.jobs[id]; ok {
				return *j, nil
			}
		}
	}

	// Reserve the whole gang atomically from the shared pool (all-or-nothing). A repeated idem key
	// already returned above, so this never double-reserves.
	if !m.pool.Reserve(spec.GPUType, spec.TotalGPUs()) {
		return domain.Job{}, ErrCapacity
	}

	now := m.now().UTC()
	// Mock-GPU: a placed gang schedules + starts instantly (the real Kueue+Volcano backend will move
	// through pending → queued → scheduled → running as pods bind).
	job := &domain.Job{
		ID:             uuid.NewString(),
		TenantID:       spec.TenantID,
		SubAccountID:   spec.SubAccountID,
		Status:         domain.StatusRunning,
		WorkloadClass:  spec.WorkloadClass,
		GPUType:        spec.GPUType,
		GPUs:           spec.GPUs,
		Pods:           spec.Pods,
		Reserved:       spec.Reserved,
		SupplySourceID: m.supplyID,
		Placement:      m.supplyID + "/mock",
		ReferenceID:    spec.ReferenceID,
		IsPaper:        spec.IsPaper,
		CreatedAt:      now,
		StartedAt:      now,
	}
	m.jobs[job.ID] = job
	if idempotencyKey != "" {
		m.idem[idemKey{spec.TenantID, idempotencyKey}] = job.ID
	}
	return *job, nil
}

// Get returns one tenant-scoped job.
func (m *MockScheduler) Get(tenantID, jobID string) (domain.Job, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	j, ok := m.jobs[jobID]
	if !ok {
		return domain.Job{}, ErrNotFound
	}
	if j.TenantID != tenantID {
		return domain.Job{}, ErrForbidden
	}
	return *j, nil
}

// List returns the tenant's jobs, newest first.
func (m *MockScheduler) List(tenantID string) []domain.Job {
	m.mu.Lock()
	defer m.mu.Unlock()
	out := make([]domain.Job, 0)
	for _, j := range m.jobs {
		if j.TenantID == tenantID {
			out = append(out, *j)
		}
	}
	sort.Slice(out, func(i, k int) bool { return out[i].CreatedAt.After(out[k].CreatedAt) })
	return out
}

// Cancel frees a live job's GPUs and emits a final usage event for the elapsed runtime.
func (m *MockScheduler) Cancel(tenantID, jobID string) (domain.Job, error) {
	m.mu.Lock()
	j, ok := m.jobs[jobID]
	if !ok {
		m.mu.Unlock()
		return domain.Job{}, ErrNotFound
	}
	if j.TenantID != tenantID {
		m.mu.Unlock()
		return domain.Job{}, ErrForbidden
	}
	if j.Status != domain.StatusRunning && j.Status != domain.StatusScheduled && j.Status != domain.StatusQueued {
		m.mu.Unlock()
		return domain.Job{}, ErrNotCancelable
	}
	j.Status = domain.StatusCancelled
	elapsed := int64(m.now().UTC().Sub(j.StartedAt).Seconds())
	snapshot := *j
	m.mu.Unlock()

	m.pool.Release(snapshot.GPUType, snapshot.TotalGPUs()) // GPUs go back to the shared pool
	m.emitUsage(snapshot, elapsed)
	return snapshot, nil
}

// emitUsage publishes one compute.usage.v1 for the job's elapsed GPU-seconds.
func (m *MockScheduler) emitUsage(j domain.Job, elapsedSeconds int64) {
	if elapsedSeconds < 0 {
		elapsedSeconds = 0
	}
	var sub *string
	if j.SubAccountID != "" {
		sub = &j.SubAccountID
	}
	gpuSeconds := domain.TotalGPUSeconds(elapsedSeconds, j.TotalGPUs())
	m.pub.PublishUsage(events.ComputeUsage{
		UsageID:        uuid.NewString(),
		TenantID:       j.TenantID,
		SubAccountID:   sub,
		InstanceID:     j.ID,
		CreditType:     j.GPUType,
		GPUSeconds:     gpuSeconds,
		Units:          domain.BillableGPUHours(elapsedSeconds, j.TotalGPUs()),
		Reserved:       j.Reserved,
		SupplySourceID: j.SupplySourceID,
		IsPaper:        j.IsPaper,
		TS:             events.NewTimestamp(),
	})
	metrics.RecordComputeUsage(j.GPUType, gpuSeconds, j.Reserved)
}

// Quotas returns per-tier capacity + this tenant's live usage + the free pool.
func (m *MockScheduler) Quotas(tenantID string) []Quota {
	m.mu.Lock()
	defer m.mu.Unlock()
	tiers := []string{domain.CreditH100, domain.CreditH200}
	out := make([]Quota, 0, len(tiers))
	for _, t := range tiers {
		out = append(out, Quota{
			GPUType:       t,
			Capacity:      m.pool.Capacity(t),
			InUse:         m.tenantInUse(t, tenantID),
			RemainingPool: m.pool.Available(t),
		})
	}
	return out
}

// Instances returns the tenant's live placements (scheduled/running).
func (m *MockScheduler) Instances(tenantID string) []domain.Job {
	m.mu.Lock()
	defer m.mu.Unlock()
	out := make([]domain.Job, 0)
	for _, j := range m.jobs {
		if j.TenantID != tenantID {
			continue
		}
		if j.Status == domain.StatusScheduled || j.Status == domain.StatusRunning {
			out = append(out, *j)
		}
	}
	sort.Slice(out, func(i, k int) bool { return out[i].CreatedAt.After(out[k].CreatedAt) })
	return out
}

// Capacity returns cluster-wide (h100Available, h200Available) from the shared pool — for the catalog.
func (m *MockScheduler) Capacity() (int, int) {
	return m.pool.Available(domain.CreditH100), m.pool.Available(domain.CreditH200)
}
