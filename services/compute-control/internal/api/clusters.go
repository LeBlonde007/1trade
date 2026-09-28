package api

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/trade1/compute-control/internal/domain"
	"github.com/trade1/compute-control/internal/instance"
)

// clusterRoutes registers the multi-node cluster endpoints (compute.yaml v1.2, F15).
func (s *Server) clusterRoutes() {
	s.mux.HandleFunc("POST /v1/compute/clusters", s.createCluster)
	s.mux.HandleFunc("GET /v1/compute/clusters", s.listClusters)
	s.mux.HandleFunc("GET /v1/compute/clusters/{id}", s.getCluster)
	s.mux.HandleFunc("DELETE /v1/compute/clusters/{id}", s.deleteCluster)
}

// clusterJSON renders a cluster in the contract shape.
func clusterJSON(c domain.Instance) map[string]any {
	out := instanceJSON(c)
	delete(out, "count")
	delete(out, "idle_stop_minutes")
	nodes := c.Nodes
	if nodes == nil {
		nodes = []domain.Node{}
	}
	out["gpus"], out["nodes"], out["network"], out["topology"] = c.Count, nodes, c.Network, c.Topology
	return out
}

// createCluster serves POST /v1/compute/clusters: a gang of whole 8-GPU nodes on one fabric, placed
// all at once or not at all. A tenant creates up to 32 GPUs itself; larger clusters are arranged with
// sales and created by operations (service token, X-Tenant-Id) for the tenant.
func (s *Server) createCluster(w http.ResponseWriter, r *http.Request) {
	operations := s.auth.IsService(bearer(r))
	tenant, sub, isPaper, ok := s.resolveActor(w, r)
	if !ok {
		return
	}
	key := strings.TrimSpace(r.Header.Get("Idempotency-Key"))
	if key == "" || len(key) > 255 {
		writeErr(w, http.StatusUnprocessableEntity, "invalid_request", "an Idempotency-Key (at most 255 chars) is required")
		return
	}
	var b struct {
		GPUType  string `json:"gpu_type"`
		GPUs     int    `json:"gpus"`
		Network  string `json:"network"`
		Topology string `json:"topology"`
		Image    string `json:"image"`
		Region   string `json:"region"`
	}
	if !decodeStrict(w, r, &b) {
		return
	}
	if b.GPUs > domain.SelfServeClusterGPUs && !operations {
		writeErr(w, http.StatusForbidden, "contact_sales",
			fmt.Sprintf("clusters over %d GPUs are arranged with sales; 1Trade creates them for you", domain.SelfServeClusterGPUs))
		return
	}
	c, err := s.inst.Create(instance.Spec{
		TenantID: tenant, SubAccountID: sub, IsPaper: isPaper, GPUType: b.GPUType, Count: b.GPUs,
		Image: b.Image, Region: b.Region, Kind: domain.KindCluster, Network: b.Network, Topology: b.Topology,
	}, key)
	if err != nil {
		writeInstErr(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, clusterJSON(c))
}

// listClusters serves GET /v1/compute/clusters (tenant JWT).
func (s *Server) listClusters(w http.ResponseWriter, r *http.Request) {
	p, ok := s.requireTenant(w, r)
	if !ok {
		return
	}
	list := s.inst.ListKind(p.TenantID, r.URL.Query().Get("state"), domain.KindCluster)
	out := make([]map[string]any, 0, len(list))
	for _, c := range list {
		out = append(out, clusterJSON(c))
	}
	writeJSON(w, http.StatusOK, map[string]any{"clusters": out})
}

// getCluster serves GET /v1/compute/clusters/{id}.
func (s *Server) getCluster(w http.ResponseWriter, r *http.Request) {
	p, ok := s.requireTenant(w, r)
	if !ok {
		return
	}
	c, err := s.inst.GetKind(p.TenantID, r.PathValue("id"), domain.KindCluster)
	if err != nil {
		writeInstErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, clusterJSON(c))
}

// deleteCluster serves DELETE /v1/compute/clusters/{id}: every node stops together and the GPUs go
// back to the pool.
func (s *Server) deleteCluster(w http.ResponseWriter, r *http.Request) {
	p, ok := s.requireTenant(w, r)
	if !ok {
		return
	}
	c, err := s.inst.DeleteKind(p.TenantID, r.PathValue("id"), domain.KindCluster)
	if err != nil {
		writeInstErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, clusterJSON(c))
}
