// Package domain — instance.go holds the customer-facing on-demand GPU instance model (F13): its
// lifecycle states, the versioned 1Trade ML-Stack image catalog, and connection info. An instance
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

// 1Trade ML-Stack images (Ubuntu + CUDA + PyTorch + common libs), newest last. `stable` and
// `latest` are aliases customers can pin against; a concrete id (e.g. 1trade-ml-stack-2026.05)
// pins exactly. Default on create is the latest stable.
const (
	ImageStable = "1trade-ml-stack-2026.05" // default; latest GA
	ImageLatest = "1trade-ml-stack-2026.06" // newest (may include preview libs)
	aliasStable = "stable"
	aliasLatest = "latest"
)

// mlStackImages is the set of pinnable concrete image ids.
var mlStackImages = []string{"1trade-ml-stack-2026.03", "1trade-ml-stack-2026.05", "1trade-ml-stack-2026.06"}

// ConnectInfo is how a customer reaches a running instance; fields are empty until it is running.
type ConnectInfo struct {
	SSH     string `json:"ssh"`
	Jupyter string `json:"jupyter"`
	HTTP    string `json:"http"`
}

// Kinds of instance.
const (
	KindInstance = "instance"
	KindCluster  = "cluster"
)

// Cluster shape (F15). A node is 8 GPUs; a cluster is 2 or more nodes on one fabric.
const (
	GPUsPerNode          = 8
	MinClusterGPUs       = 16
	MaxClusterGPUs       = 256
	SelfServeClusterGPUs = 32 // larger clusters are arranged with sales
)

// Node is one machine of a running cluster.
type Node struct {
	Name string `json:"name"`
	SSH  string `json:"ssh"`
}

// BuildNodes names a running cluster's nodes and how to reach each (node 0 is the head node).
func BuildNodes(id, region string, gpus int) []Node {
	out := make([]Node, 0, gpus/GPUsPerNode)
	for i := range gpus / GPUsPerNode {
		out = append(out, Node{Name: fmt.Sprintf("node-%d", i), SSH: fmt.Sprintf("ssh ubuntu@node-%d.%s.%s.gpu.1trade.io", i, id, region)})
	}
	return out
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
	Reserved        bool   // its GPUs come from the tenant's prepaid reservation (F14): usage bills zero
	Kind            string // KindInstance, or KindCluster (F15: multi-node, one fabric)
	Network         string // clusters: "infiniband"
	Topology        string // clusters: "fat-tree" | "rail-optimized"
	Nodes           []Node // clusters: one entry per 8-GPU node, while running
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
	host := fmt.Sprintf("%s.%s.gpu.1trade.io", id, region)
	return ConnectInfo{
		SSH:     "ssh ubuntu@" + host,
		Jupyter: "https://" + host + "/jupyter",
		HTTP:    "https://" + host,
	}
}
