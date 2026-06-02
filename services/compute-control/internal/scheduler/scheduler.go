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

	"github.com/exascale/compute-control/internal/domain"
	"github.com/exascale/compute-control/internal/events"
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

// capacity is the per-tier total GPU count.
type capacity struct{ h100, h200 int }

// MockScheduler is an in-memory Scheduler: it admits a gang only if the whole footprint fits the
// remaining pool for its tier, attributes every placement to the configured supply source, meters
// nothing until a job is cancelled/completed, and is idempotent on (tenant, Idempotency-Key).
type MockScheduler struct {
	mu       sync.Mutex
	jobs     map[string]*domain.Job // jobID → job
	idem     map[idemKey]string     // (tenant, key) → jobID
	cap      capacity
	supplyID string
	pub      events.Publisher
	now      func() time.Time
}

// NewMock builds a MockScheduler with the given per-tier capacity, supply attribution, and usage
// publisher (pass events.NoopPublisher{} when NATS is unconfigured).
func NewMock(h100, h200 int, supplyID string, pub events.Publisher) *MockScheduler {
	return &MockScheduler{
		jobs:     make(map[string]*domain.Job),
		idem:     make(map[idemKey]string),
		cap:      capacity{h100: h100, h200: h200},
		supplyID: supplyID,
		pub:      pub,
		now:      time.Now,
	}
}

// poolFor returns the total capacity for a tier.
func (m *MockScheduler) poolFor(gpuType string) int {
	switch gpuType {
	case domain.CreditH100:
		return m.cap.h100
	case domain.CreditH200:
		return m.cap.h200
	default:
		return 0
	}
}

// inUseFor returns GPUs of a tier currently held by live jobs (optionally only one tenant's).
// Caller must hold m.mu.
func (m *MockScheduler) inUseFor(gpuType, tenantID string) int {
	n := 0
	for _, j := range m.jobs {
		if j.GPUType != gpuType {
			continue
		}
		if tenantID != "" && j.TenantID != tenantID {
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

	want := spec.TotalGPUs()
	free := m.poolFor(spec.GPUType) - m.inUseFor(spec.GPUType, "")
	if want > free {
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
	m.pub.PublishUsage(events.ComputeUsage{
		UsageID:        uuid.NewString(),
		TenantID:       j.TenantID,
		SubAccountID:   sub,
		InstanceID:     j.ID,
		CreditType:     j.GPUType,
		GPUSeconds:     domain.TotalGPUSeconds(elapsedSeconds, j.TotalGPUs()),
		Units:          domain.BillableGPUHours(elapsedSeconds, j.TotalGPUs()),
		Reserved:       j.Reserved,
		SupplySourceID: j.SupplySourceID,
		IsPaper:        j.IsPaper,
		TS:             events.NewTimestamp(),
	})
}

// Quotas returns per-tier capacity + this tenant's live usage + the free pool.
func (m *MockScheduler) Quotas(tenantID string) []Quota {
	m.mu.Lock()
	defer m.mu.Unlock()
	tiers := []string{domain.CreditH100, domain.CreditH200}
	out := make([]Quota, 0, len(tiers))
	for _, t := range tiers {
		pool := m.poolFor(t)
		out = append(out, Quota{
			GPUType:       t,
			Capacity:      pool,
			InUse:         m.inUseFor(t, tenantID),
			RemainingPool: pool - m.inUseFor(t, ""),
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

// Capacity returns cluster-wide (h100Available, h200Available) for the catalog.
func (m *MockScheduler) Capacity() (int, int) {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.cap.h100 - m.inUseFor(domain.CreditH100, ""),
		m.cap.h200 - m.inUseFor(domain.CreditH200, "")
}
