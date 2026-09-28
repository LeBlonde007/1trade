// Package instance manages the customer-facing on-demand GPU instance lifecycle (F13): create / get /
// list / stop / start / delete, plus per-second GPU metering. Instances draw GPUs from the same
// shared pool the internal scheduler places jobs on, so capacity is never double-counted. The Manager
// is the in-memory (mock-GPU) backend behind the same seam the real K8s provisioner will implement.
package instance

import (
	"errors"
	"fmt"
	"sort"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/trade1/compute-control/internal/domain"
	"github.com/trade1/compute-control/internal/events"
	"github.com/trade1/compute-control/internal/metrics"
	"github.com/trade1/compute-control/internal/pool"
)

// Lifecycle errors surfaced to the API layer (mapped to 4xx).
var (
	ErrUnknownGPU   = errors.New("unknown gpu type")
	ErrBadImage     = errors.New("unknown image")
	ErrBadRequest   = errors.New("invalid instance request")
	ErrCapacity     = errors.New("insufficient gpu capacity")
	ErrNotFound     = errors.New("instance not found")
	ErrForbidden    = errors.New("not your instance")
	ErrNotStoppable = errors.New("instance not in a stoppable state")
	ErrNotStartable = errors.New("instance not in a startable state")
	ErrTerminated   = errors.New("instance is terminated")
)

// MaxCount is the most GPUs one self-serve instance may have (compute.yaml InstanceRequest.count).
const MaxCount = 32

// holdKey is the reservation hold an instance of this tenant, side and tier draws from.
func holdKey(tenant string, isPaper bool, tier string) pool.HoldKey {
	return pool.HoldKey{Tenant: tenant, IsPaper: isPaper, Tier: tier}
}

// release returns an instance's GPUs (and, if they came from its reservation, the hold).
func (m *Manager) release(inst domain.Instance) {
	m.pool.ReleaseFor(holdKey(inst.TenantID, inst.IsPaper, inst.GPUType), inst.SupplySourceID, inst.Count, inst.Reserved)
}

// validShape checks an instance's or cluster's GPU count and, for a cluster, its fabric.
func validShape(spec Spec) error {
	switch spec.Kind {
	case domain.KindInstance:
		if spec.Count > MaxCount {
			return ErrBadRequest
		}
	case domain.KindCluster:
		if spec.Count < domain.MinClusterGPUs || spec.Count > domain.MaxClusterGPUs || spec.Count%domain.GPUsPerNode != 0 {
			return fmt.Errorf("%w: a cluster is %d-%d GPUs in whole %d-GPU nodes", ErrBadRequest, domain.MinClusterGPUs, domain.MaxClusterGPUs, domain.GPUsPerNode)
		}
		if spec.Network != "infiniband" || (spec.Topology != "fat-tree" && spec.Topology != "rail-optimized") {
			return fmt.Errorf("%w: network must be infiniband; topology fat-tree or rail-optimized", ErrBadRequest)
		}
	default:
		return ErrBadRequest
	}
	return nil
}

// Spec is a validated request to create an instance.
type Spec struct {
	TenantID        string
	SubAccountID    string
	IsPaper         bool
	GPUType         string
	Count           int
	Image           string // alias ("stable"/"latest") or a pinned id
	Region          string
	IdleStopMinutes *int
	// Clusters (F15): Kind = domain.KindCluster with a network and topology; Count is the GPUs.
	Kind     string
	Network  string
	Topology string
}

// idemKey scopes idempotency to a tenant so two tenants can reuse the same key value.
type idemKey struct {
	tenant string
	key    string
}

// Manager is the in-memory instance backend. Safe for concurrent use.
type Manager struct {
	mu        sync.Mutex
	instances map[string]*domain.Instance // id → instance
	idem      map[idemKey]string          // (tenant, key) → id
	pool      *pool.Pool
	pub       events.Publisher
	now       func() time.Time
}

// NewManager builds a Manager sharing the given GPU pool, supply attribution, and usage publisher
// (pass events.NoopPublisher{} when NATS is unconfigured).
func NewManager(p *pool.Pool, pub events.Publisher) *Manager {
	return &Manager{
		instances: make(map[string]*domain.Instance),
		idem:      make(map[idemKey]string),
		pool:      p,
		pub:       pub,
		now:       time.Now,
	}
}

// Create validates the spec, reserves GPUs from the shared pool, and boots the instance. Mock-GPU:
// a created instance is immediately running with connection info (the real backend moves through
// provisioning → running as the node boots, target <90s P95). Idempotent on (tenant, idempotencyKey).
func (m *Manager) Create(spec Spec, idempotencyKey string) (domain.Instance, error) {
	if spec.Count < 1 {
		spec.Count = 1
	}
	if !domain.ValidGPUType(spec.GPUType) {
		return domain.Instance{}, ErrUnknownGPU
	}
	image, err := domain.ResolveImage(spec.Image)
	if err != nil {
		return domain.Instance{}, ErrBadImage
	}
	region := spec.Region
	if region == "" {
		region = "us-east-1"
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	if idempotencyKey != "" {
		idempotencyKey = spec.Kind + ":" + idempotencyKey // an instance key never replays a cluster
		if id, ok := m.idem[idemKey{spec.TenantID, idempotencyKey}]; ok {
			if inst, ok := m.instances[id]; ok {
				return *inst, nil
			}
		}
	}

	if spec.Kind == "" {
		spec.Kind = domain.KindInstance
	}
	if err := validShape(spec); err != nil {
		return domain.Instance{}, err
	}
	source, reserved, ok := m.pool.ReserveFor(holdKey(spec.TenantID, spec.IsPaper, spec.GPUType), spec.Count)
	if !ok {
		return domain.Instance{}, ErrCapacity
	}

	now := m.now().UTC()
	id := "i-" + uuid.NewString()[:8]
	if spec.Kind == domain.KindCluster {
		id = "c-" + uuid.NewString()[:8]
	}
	inst := &domain.Instance{
		ID:              id,
		TenantID:        spec.TenantID,
		SubAccountID:    spec.SubAccountID,
		GPUType:         spec.GPUType,
		Count:           spec.Count,
		State:           domain.InstanceRunning, // mock boots instantly
		Image:           image,
		Region:          region,
		Connect:         domain.BuildConnect(id, region),
		SupplySourceID:  source, // whichever datacenter took it (F16): drives the partner payout
		IdleStopMinutes: spec.IdleStopMinutes,
		IsPaper:         spec.IsPaper,
		Reserved:        reserved,
		Kind:            spec.Kind,
		Network:         spec.Network,
		Topology:        spec.Topology,
		CreatedAt:       now,
		StartedAt:       now,
		LastMeteredAt:   now,
	}
	if spec.Kind == domain.KindCluster {
		inst.Nodes = domain.BuildNodes(id, region, spec.Count)
	}
	m.instances[id] = inst
	if idempotencyKey != "" {
		m.idem[idemKey{spec.TenantID, idempotencyKey}] = id
	}
	return *inst, nil
}

// Get returns one tenant-scoped instance (not a cluster).
func (m *Manager) Get(tenantID, id string) (domain.Instance, error) {
	return m.GetKind(tenantID, id, domain.KindInstance)
}

// GetKind returns one tenant-scoped instance or cluster.
func (m *Manager) GetKind(tenantID, id, kind string) (domain.Instance, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	inst, err := m.ownedKind(tenantID, id, kind)
	if err != nil {
		return domain.Instance{}, err
	}
	return *inst, nil
}

// List returns the tenant's instances newest-first, optionally filtered to one state ("" = all).
func (m *Manager) List(tenantID, state string) []domain.Instance {
	return m.ListKind(tenantID, state, domain.KindInstance)
}

// ListKind returns the tenant's instances or clusters newest-first, optionally filtered by state.
func (m *Manager) ListKind(tenantID, state, kind string) []domain.Instance {
	m.mu.Lock()
	defer m.mu.Unlock()
	out := make([]domain.Instance, 0)
	for _, inst := range m.instances {
		if inst.TenantID != tenantID || inst.Kind != kind {
			continue
		}
		if state != "" && inst.State != state {
			continue
		}
		out = append(out, *inst)
	}
	sort.Slice(out, func(i, k int) bool { return out[i].CreatedAt.After(out[k].CreatedAt) })
	return out
}

// Stop releases a running instance's GPUs back to the pool, meters the final running interval, and
// transitions it to stopped. Idempotent-ish: stopping an already-stopped instance is a conflict.
func (m *Manager) Stop(tenantID, id string) (domain.Instance, error) {
	m.mu.Lock()
	inst, err := m.owned(tenantID, id)
	if err != nil {
		m.mu.Unlock()
		return domain.Instance{}, err
	}
	if !domain.CanStop(inst.State) {
		m.mu.Unlock()
		return domain.Instance{}, ErrNotStoppable
	}
	elapsed := m.meterSince(inst) // final partial interval while still holding the lock
	inst.State = domain.InstanceStopped
	inst.Connect = domain.ConnectInfo{}
	snapshot := *inst
	m.mu.Unlock()

	m.release(snapshot)
	m.emit(snapshot, elapsed)
	return snapshot, nil
}

// Start re-reserves GPUs for a stopped instance (capacity permitting) and resumes it.
func (m *Manager) Start(tenantID, id string) (domain.Instance, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	inst, err := m.owned(tenantID, id)
	if err != nil {
		return domain.Instance{}, err
	}
	if !domain.CanStart(inst.State) {
		return domain.Instance{}, ErrNotStartable
	}
	source, reserved, ok := m.pool.ReserveFor(holdKey(inst.TenantID, inst.IsPaper, inst.GPUType), inst.Count)
	if !ok {
		return domain.Instance{}, ErrCapacity
	}
	now := m.now().UTC()
	inst.SupplySourceID = source // a restart may land on a different datacenter
	inst.Reserved = reserved     // … and on or off the tenant's reservation
	inst.State = domain.InstanceRunning
	inst.Connect = domain.BuildConnect(inst.ID, inst.Region)
	inst.StartedAt = now
	inst.LastMeteredAt = now
	return *inst, nil
}

// Delete terminates an instance. If it was live, its GPUs are released and the final running interval
// is metered. Terminating an already-terminated instance is a no-op success.
func (m *Manager) Delete(tenantID, id string) (domain.Instance, error) {
	return m.DeleteKind(tenantID, id, domain.KindInstance)
}

// DeleteKind terminates an instance or a cluster (all its nodes at once).
func (m *Manager) DeleteKind(tenantID, id, kind string) (domain.Instance, error) {
	m.mu.Lock()
	inst, err := m.ownedKind(tenantID, id, kind)
	if err != nil {
		m.mu.Unlock()
		return domain.Instance{}, err
	}
	wasLive := domain.HoldsGPU(inst.State)
	var elapsed int64
	if wasLive {
		elapsed = m.meterSince(inst)
	}
	inst.State = domain.InstanceTerminated
	inst.Connect = domain.ConnectInfo{}
	inst.Nodes = nil
	snapshot := *inst
	m.mu.Unlock()

	if wasLive {
		m.release(snapshot)
		m.emit(snapshot, elapsed)
	}
	return snapshot, nil
}

// MeterTick emits a usage event for every running instance for the GPU-seconds elapsed since each was
// last metered, advancing the watermark. Call it on a fixed interval (e.g. every 60s) so debits flow
// per-interval while instances run; Stop/Delete emit the final partial interval.
func (m *Manager) MeterTick() {
	m.mu.Lock()
	type pending struct {
		inst    domain.Instance
		elapsed int64
	}
	var batch []pending
	ids := make([]string, 0, len(m.instances))
	for id := range m.instances {
		ids = append(ids, id)
	}
	sort.Strings(ids) // which instances leave an ended reservation first is deterministic
	for _, id := range ids {
		inst := m.instances[id]
		if inst.State != domain.InstanceRunning {
			continue
		}
		elapsed := m.meterSince(inst)
		if elapsed > 0 {
			batch = append(batch, pending{inst: *inst, elapsed: elapsed})
		}
		// A reservation that ended (or shrank) leaves its running work overdrawn: from the next
		// interval that work is billed on-demand. The interval just metered stays prepaid. An instance
		// moves only when the whole of it is overdrawn, so the customer is never billed on-demand for
		// GPUs a reservation still covers.
		key := holdKey(inst.TenantID, inst.IsPaper, inst.GPUType)
		if inst.Reserved && m.pool.Overdrawn(key) >= inst.Count {
			m.pool.Detach(key, inst.Count)
			inst.Reserved = false
		}
	}
	m.mu.Unlock()

	for _, p := range batch {
		m.emit(p.inst, p.elapsed)
	}
}

// owned returns the tenant's instance (not a cluster: those are not stopped or started) or an error.
// Caller must hold m.mu.
func (m *Manager) owned(tenantID, id string) (*domain.Instance, error) {
	return m.ownedKind(tenantID, id, domain.KindInstance)
}

// ownedKind returns the tenant's instance or cluster of the given kind; another kind is not found.
// Caller must hold m.mu.
func (m *Manager) ownedKind(tenantID, id, kind string) (*domain.Instance, error) {
	inst, ok := m.instances[id]
	if !ok || inst.Kind != kind {
		return nil, ErrNotFound
	}
	if inst.TenantID != tenantID {
		return nil, ErrForbidden
	}
	return inst, nil
}

// meterSince returns the GPU-seconds elapsed since the instance was last metered and advances its
// watermark to now. Caller must hold m.mu.
func (m *Manager) meterSince(inst *domain.Instance) int64 {
	now := m.now().UTC()
	elapsed := max(int64(now.Sub(inst.LastMeteredAt).Seconds()), 0)
	inst.LastMeteredAt = now
	return elapsed
}

// emit publishes one compute.usage.v1 for an instance's elapsed GPU-seconds across its GPUs. Units is
// the billable GPU-hours; the ledger debits the instance's gpu_* tier, idempotent on usage_id.
func (m *Manager) emit(inst domain.Instance, elapsedSeconds int64) {
	if elapsedSeconds <= 0 {
		return
	}
	var sub *string
	if inst.SubAccountID != "" {
		sub = &inst.SubAccountID
	}
	gpuSeconds := domain.TotalGPUSeconds(elapsedSeconds, inst.Count)
	units := domain.BillableGPUHours(elapsedSeconds, inst.Count)
	if inst.Reserved {
		units = "0.000000" // prepaid by the reservation; the GPU time is still reported
	}
	m.pub.PublishUsage(events.ComputeUsage{
		UsageID:        uuid.NewString(),
		TenantID:       inst.TenantID,
		SubAccountID:   sub,
		InstanceID:     inst.ID,
		CreditType:     inst.GPUType,
		GPUSeconds:     gpuSeconds,
		Units:          units,
		Reserved:       inst.Reserved,
		SupplySourceID: inst.SupplySourceID,
		IsPaper:        inst.IsPaper,
		TS:             events.NewTimestamp(),
	})
	metrics.RecordComputeUsage(inst.GPUType, gpuSeconds, inst.Reserved)
}

// Available is the on-demand GPUs of a tier free right now (capacity set aside for reservations is
// not counted).
func (m *Manager) Available(tier string) int { return m.pool.Available(tier) }

// Holds snapshots the pool's reservation holds: GPUs set aside and GPUs occupied, per key.
func (m *Manager) Holds() map[pool.HoldKey][2]int { return m.pool.Holds() }
