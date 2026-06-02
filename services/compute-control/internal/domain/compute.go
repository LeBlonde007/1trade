// Package domain holds the compute control-plane's core types: GPU tiers, workload classes, the
// scheduled-job model, and the exact (no-float) GPU-hour billing math.
package domain

import (
	"math/big"
	"time"
)

// GPU tier == the credit_type debited for it (credit-types.md §1). 1 credit = 1 GPU-hour.
const (
	CreditH100 = "gpu_h100"
	CreditH200 = "gpu_h200"
)

// WorkloadClass selects the Kueue queue a job is admitted through.
const (
	ClassInference     = "inference"
	ClassTrainingSmall = "training-small"
	ClassTrainingLarge = "training-large"
)

// secondsPerHour is the GPU-seconds → GPU-hours divisor.
const secondsPerHour = 3600

// ComputeType is one schedulable GPU tier with its price + current availability.
type ComputeType struct {
	ID           string // == credit_type
	Name         string
	GPU          string
	CreditType   string
	PricePerHour string // fixed-point decimal string
	Available    int    // schedulable GPUs right now (mock-GPU count locally)
}

// Catalog is the static GPU-tier catalog with live availability injected by the scheduler.
func Catalog(h100Avail, h200Avail int) []ComputeType {
	return []ComputeType{
		{ID: CreditH100, Name: "H100 80GB", GPU: "NVIDIA H100 80GB", CreditType: CreditH100, PricePerHour: "2.990000", Available: h100Avail},
		{ID: CreditH200, Name: "H200 141GB", GPU: "NVIDIA H200", CreditType: CreditH200, PricePerHour: "3.490000", Available: h200Avail},
	}
}

// ValidGPUType reports whether t is a known GPU tier.
func ValidGPUType(t string) bool { return t == CreditH100 || t == CreditH200 }

// ValidWorkloadClass reports whether c is a known workload class.
func ValidWorkloadClass(c string) bool {
	return c == ClassInference || c == ClassTrainingSmall || c == ClassTrainingLarge
}

// Job status lifecycle.
const (
	StatusPending   = "pending"
	StatusQueued    = "queued"
	StatusScheduled = "scheduled"
	StatusRunning   = "running"
	StatusSucceeded = "succeeded"
	StatusFailed    = "failed"
	StatusCancelled = "cancelled"
)

// Job is a scheduled workload tracked by the control plane.
type Job struct {
	ID             string
	TenantID       string
	SubAccountID   string
	Status         string
	WorkloadClass  string
	GPUType        string
	GPUs           int
	Pods           int
	Reserved       bool
	SupplySourceID string
	Placement      string
	ReferenceID    string
	IsPaper        bool
	CreatedAt      time.Time
	StartedAt      time.Time
}

// TotalGPUs is gpus-per-pod × pods — the all-or-nothing gang size.
func (j Job) TotalGPUs() int { return j.GPUs * j.Pods }

// BillableGPUHours converts elapsed GPU-seconds across the job's GPUs to GPU-hours, exact fixed-point
// (no floats): (gpuSeconds × totalGPUs) / 3600, formatted to 6dp. Drives the compute.usage.v1 `units`.
func BillableGPUHours(gpuSeconds int64, totalGPUs int) string {
	r := new(big.Rat).SetFrac(
		big.NewInt(gpuSeconds*int64(totalGPUs)),
		big.NewInt(secondsPerHour),
	)
	return r.FloatString(6)
}

// TotalGPUSeconds renders the metered GPU-seconds (elapsed × totalGPUs) as a fixed-point 6dp string.
// Drives the compute.usage.v1 `gpu_seconds` (the raw basis the ledger keeps; `units` is the GPU-hours).
func TotalGPUSeconds(gpuSeconds int64, totalGPUs int) string {
	return new(big.Rat).SetInt64(gpuSeconds * int64(totalGPUs)).FloatString(6)
}
