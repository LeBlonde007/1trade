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

// Marketplace availability status for a GPU tier (drives the catalog badges + the rent CTA).
const (
	StatusAvailable  = "available"
	StatusSoldOut    = "sold_out"
	StatusComingSoon = "coming_soon"
)

// WorkloadClass selects the Kueue queue a job is admitted through.
const (
	ClassInference     = "inference"
	ClassTrainingSmall = "training-small"
	ClassTrainingLarge = "training-large"
)

// secondsPerHour is the GPU-seconds → GPU-hours divisor.
const secondsPerHour = 3600

// GPUSpecs are the per-card datasheet specs surfaced in the rental catalog. Values are vendor datasheet
// figures (peak dense throughput); they're informational for the marketplace, not used in billing math.
type GPUSpecs struct {
	Architecture string `json:"architecture"`
	VRAM         string `json:"vram"`
	MemBandwidth string `json:"mem_bandwidth"`
	FP16         string `json:"fp16_tflops"`
	FP8          string `json:"fp8_tflops"`
	FP4          string `json:"fp4_tflops,omitempty"`
	NVLink       string `json:"nvlink"`
	Interconnect string `json:"interconnect"`
	TDP          string `json:"tdp"`
	FormFactor   string `json:"form_factor"`
	VCPUs        int    `json:"vcpus"`
	HostRAM      string `json:"host_ram"`
	Released     string `json:"released"`
}

// ComputeType is one GPU tier in the rental catalog: identity, price, marketplace availability, and the
// datasheet specs shown to buyers.
type ComputeType struct {
	ID           string // == credit_type
	Name         string
	GPU          string
	CreditType   string
	PricePerHour string // fixed-point decimal string
	Available    int    // schedulable GPUs right now (live for H100/H200; indicative for marketplace-only tiers)
	Status       string // available | sold_out | coming_soon
	Specs        GPUSpecs
}

// Catalog is the GPU rental catalog. H100/H200 are the schedulable tiers (live availability injected by
// the scheduler); the rest are the published marketplace lineup with indicative availability + status so
// the console shows the full rental experience (Hopper → Ada → Blackwell → Grace-Blackwell racks).
func Catalog(h100Avail, h200Avail int) []ComputeType {
	return []ComputeType{
		{
			ID: "gpu_a100", Name: "A100 80GB", GPU: "NVIDIA A100 80GB", CreditType: "gpu_a100", PricePerHour: "1.690000",
			Available: 0, Status: StatusSoldOut,
			Specs: GPUSpecs{Architecture: "Ampere", VRAM: "80 GB HBM2e", MemBandwidth: "2.0 TB/s", FP16: "312", FP8: "—", NVLink: "600 GB/s", Interconnect: "NVLink 3 · HDR InfiniBand", TDP: "400 W", FormFactor: "SXM4", VCPUs: 24, HostRAM: "220 GB", Released: "2020"},
		},
		{
			ID: CreditH100, Name: "H100 80GB", GPU: "NVIDIA H100 80GB", CreditType: CreditH100, PricePerHour: "2.990000",
			Available: h100Avail, Status: StatusSoldOut, // headline tier — sold out in the current cohort
			Specs: GPUSpecs{Architecture: "Hopper", VRAM: "80 GB HBM3", MemBandwidth: "3.35 TB/s", FP16: "989", FP8: "1,979", NVLink: "900 GB/s", Interconnect: "NVLink 4 · NDR InfiniBand", TDP: "700 W", FormFactor: "SXM5", VCPUs: 24, HostRAM: "220 GB", Released: "2022"},
		},
		{
			ID: CreditH200, Name: "H200 141GB", GPU: "NVIDIA H200", CreditType: CreditH200, PricePerHour: "3.490000",
			Available: max(h200Avail, 12), Status: StatusAvailable,
			Specs: GPUSpecs{Architecture: "Hopper", VRAM: "141 GB HBM3e", MemBandwidth: "4.8 TB/s", FP16: "989", FP8: "1,979", NVLink: "900 GB/s", Interconnect: "NVLink 4 · NDR InfiniBand", TDP: "700 W", FormFactor: "SXM5", VCPUs: 26, HostRAM: "250 GB", Released: "2024"},
		},
		{
			ID: "gpu_l40s", Name: "L40S 48GB", GPU: "NVIDIA L40S", CreditType: "gpu_l40s", PricePerHour: "1.290000",
			Available: 64, Status: StatusAvailable,
			Specs: GPUSpecs{Architecture: "Ada Lovelace", VRAM: "48 GB GDDR6", MemBandwidth: "864 GB/s", FP16: "362", FP8: "733", NVLink: "—", Interconnect: "PCIe Gen4 · 100 GbE", TDP: "350 W", FormFactor: "PCIe", VCPUs: 16, HostRAM: "128 GB", Released: "2023"},
		},
		{
			ID: "gpu_b200", Name: "B200 192GB", GPU: "NVIDIA B200", CreditType: "gpu_b200", PricePerHour: "5.990000",
			Available: 8, Status: StatusAvailable,
			Specs: GPUSpecs{Architecture: "Blackwell", VRAM: "192 GB HBM3e", MemBandwidth: "8.0 TB/s", FP16: "2,250", FP8: "4,500", FP4: "9,000", NVLink: "1.8 TB/s", Interconnect: "NVLink 5 · XDR InfiniBand", TDP: "1000 W", FormFactor: "SXM6", VCPUs: 28, HostRAM: "320 GB", Released: "2025"},
		},
		{
			ID: "gpu_b300", Name: "B300 288GB", GPU: "NVIDIA B300 (Blackwell Ultra)", CreditType: "gpu_b300", PricePerHour: "7.490000",
			Available: 4, Status: StatusAvailable,
			Specs: GPUSpecs{Architecture: "Blackwell Ultra", VRAM: "288 GB HBM3e", MemBandwidth: "8.0 TB/s", FP16: "2,500", FP8: "5,000", FP4: "15,000", NVLink: "1.8 TB/s", Interconnect: "NVLink 5 · XDR InfiniBand", TDP: "1200 W", FormFactor: "SXM6", VCPUs: 32, HostRAM: "384 GB", Released: "2025"},
		},
		{
			ID: "gpu_gb200", Name: "GB200 NVL72", GPU: "NVIDIA GB200 NVL72 (per GPU)", CreditType: "gpu_gb200", PricePerHour: "6.990000",
			Available: 0, Status: StatusComingSoon,
			Specs: GPUSpecs{Architecture: "Grace-Blackwell", VRAM: "192 GB HBM3e", MemBandwidth: "8.0 TB/s", FP16: "2,250", FP8: "4,500", FP4: "9,000", NVLink: "1.8 TB/s (72-GPU domain)", Interconnect: "NVLink 5 rack · XDR InfiniBand", TDP: "~1200 W", FormFactor: "NVL72 rack", VCPUs: 28, HostRAM: "480 GB (Grace)", Released: "2025"},
		},
		{
			ID: "gpu_gb300", Name: "GB300 NVL72", GPU: "NVIDIA GB300 NVL72 (per GPU)", CreditType: "gpu_gb300", PricePerHour: "8.990000",
			Available: 0, Status: StatusComingSoon,
			Specs: GPUSpecs{Architecture: "Grace-Blackwell Ultra", VRAM: "288 GB HBM3e", MemBandwidth: "8.0 TB/s", FP16: "2,500", FP8: "5,000", FP4: "15,000", NVLink: "1.8 TB/s (72-GPU domain)", Interconnect: "NVLink 5 rack · XDR InfiniBand", TDP: "~1400 W", FormFactor: "NVL72 rack", VCPUs: 32, HostRAM: "576 GB (Grace)", Released: "2026"},
		},
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
