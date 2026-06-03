// Package domain — instance.go holds the customer-facing on-demand GPU instance model (F13): its
// lifecycle states, the versioned Exascale ML-Stack image catalog, and connection info. An instance
// draws GPUs of one tier from the same shared pool the internal scheduler places jobs on.
package domain

import (
	"fmt"
	"time"
)

// Instance lifecycle states. provisioning → running; running → stopping → stopped; stopped →
// starting → running; any → terminated (irreversible). States that hold GPUs: provisioning, running,
// starting (and momentarily stopping). stopped + terminated hold none.
const (
	InstanceProvisioning = "provisioning"
	InstanceRunning      = "running"
	InstanceStopping     = "stopping"
	InstanceStopped      = "stopped"
	InstanceStarting     = "starting"
	InstanceTerminated   = "terminated"
)

// Exascale ML-Stack images (Ubuntu + CUDA + PyTorch + common libs), newest last. `stable` and
// `latest` are aliases customers can pin against; a concrete id (e.g. exascale-ml-stack-2026.05)
// pins exactly. Default on create is the latest stable.
const (
	ImageStable = "exascale-ml-stack-2026.05" // default; latest GA
	ImageLatest = "exascale-ml-stack-2026.06" // newest (may include preview libs)
	aliasStable = "stable"
	aliasLatest = "latest"
)

// mlStackImages is the set of pinnable concrete image ids.
var mlStackImages = []string{"exascale-ml-stack-2026.03", "exascale-ml-stack-2026.05", "exascale-ml-stack-2026.06"}

// ConnectInfo is how a customer reaches a running instance; fields are empty until it is running.
type ConnectInfo struct {
	SSH     string `json:"ssh"`
	Jupyter string `json:"jupyter"`
	HTTP    string `json:"http"`
}

// Instance is one customer on-demand GPU instance.
type Instance struct {
	ID              string
	TenantID        string
	SubAccountID    string
	GPUType         string // gpu_h100 | gpu_h200 (== credit_type debited)
	Count           int    // GPUs in this instance
	State           string
	Image           string // resolved concrete image id
	Region          string
	Connect         ConnectInfo
	SupplySourceID  string
	IdleStopMinutes *int // auto-stop after this many idle minutes; nil = none
	IsPaper         bool
	CreatedAt       time.Time
	StartedAt       time.Time // last time it entered running (zero if never)
	LastMeteredAt   time.Time // watermark for per-second metering
}

// ResolveImage maps an image reference (alias or concrete id) to a concrete image id. Unknown pinned
// ids are rejected so a typo can't silently boot the wrong stack.
func ResolveImage(ref string) (string, error) {
	switch ref {
	case "", aliasStable:
		return ImageStable, nil
	case aliasLatest:
		return ImageLatest, nil
	default:
		for _, v := range mlStackImages {
			if v == ref {
				return v, nil
			}
		}
		return "", fmt.Errorf("unknown image %q", ref)
	}
}

// HoldsGPU reports whether an instance in state s is occupying GPUs from the pool (so they must be
// released when it leaves that set).
func HoldsGPU(s string) bool {
	return s == InstanceProvisioning || s == InstanceRunning || s == InstanceStarting
}

// CanStop reports whether an instance in state s may be stopped.
func CanStop(s string) bool {
	return s == InstanceRunning || s == InstanceProvisioning || s == InstanceStarting
}

// CanStart reports whether an instance in state s may be (re)started.
func CanStart(s string) bool { return s == InstanceStopped }

// BuildConnect returns the connection endpoints for a running instance id in a region.
func BuildConnect(id, region string) ConnectInfo {
	host := fmt.Sprintf("%s.%s.gpu.exascale.io", id, region)
	return ConnectInfo{
		SSH:     "ssh ubuntu@" + host,
		Jupyter: "https://" + host + "/jupyter",
		HTTP:    "https://" + host,
	}
}
