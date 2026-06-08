package main

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/exascale/cli/internal/client"
	"github.com/exascale/cli/internal/config"
	"github.com/exascale/cli/internal/ui"
)

// instance mirrors the compute.yaml Instance shape (the fields the CLI prints).
type instance struct {
	ID      string `json:"id"`
	GPUType string `json:"gpu_type"`
	Count   int    `json:"count"`
	State   string `json:"state"`
	Image   string `json:"image"`
	Region  string `json:"region"`
	Connect struct {
		SSH     string `json:"ssh"`
		Jupyter string `json:"jupyter"`
		HTTP    string `json:"http"`
	} `json:"connect"`
	CreatedAt string `json:"created_at"`
}

// normalizeGPUType accepts a friendly tier ("h100") or the full credit type ("gpu_h100").
func normalizeGPUType(t string) string {
	t = strings.ToLower(strings.TrimSpace(t))
	if t == "" {
		return ""
	}
	if strings.HasPrefix(t, "gpu_") {
		return t
	}
	return "gpu_" + t
}

// cmdGPU dispatches the GPU instance subcommands (F13).
func cmdGPU(cfg config.Config, args []string) error {
	if len(args) == 0 {
		return errors.New("usage: exascale gpu types|create|list|get|stop|start|delete")
	}
	switch args[0] {
	case "types":
		return gpuTypes(cfg)
	case "create":
		return gpuCreate(cfg, args[1:])
	case "list":
		return gpuList(cfg, args[1:])
	case "get":
		return gpuGet(cfg, args[1:])
	case "stop":
		return gpuAction(cfg, "stop", args[1:])
	case "start":
		return gpuAction(cfg, "start", args[1:])
	case "delete":
		return gpuDelete(cfg, args[1:])
	default:
		return fmt.Errorf("unknown gpu subcommand %q", args[0])
	}
}

// gpuTypes lists GPU tiers with price + current availability (no auth required).
func gpuTypes(cfg config.Config) error {
	var out struct {
		Types []struct {
			ID           string `json:"id"`
			Name         string `json:"name"`
			PricePerHour string `json:"price_per_hour"`
			Available    int    `json:"available"`
		} `json:"types"`
	}
	if err := client.Do("GET", cfg.ComputeURL, "/v1/compute/types", cfg.Token, nil, nil, &out); err != nil {
		return err
	}
	fmt.Printf("%-12s %-14s %12s %10s\n", "TYPE", "NAME", "PRICE/HR", "AVAILABLE")
	for _, t := range out.Types {
		fmt.Printf("%-12s %-14s %12s %10d\n", t.ID, t.Name, t.PricePerHour, t.Available)
	}
	return nil
}

// gpuCreate launches an on-demand instance and prints its connection info.
func gpuCreate(cfg config.Config, args []string) error {
	if err := requireToken(cfg); err != nil {
		return err
	}
	fs := flag.NewFlagSet("create", flag.ExitOnError)
	gtype := fs.String("type", "", "GPU type: h100 | h200")
	count := fs.Int("count", 1, "number of GPUs")
	image := fs.String("image", "stable", "ML-Stack image: stable | latest | pinned id")
	region := fs.String("region", "us-east-1", "region")
	_ = fs.Parse(args)
	if *gtype == "" {
		return errors.New("usage: exascale gpu create --type h100 [--count N] [--image stable] [--region R]")
	}
	body := map[string]any{"type": normalizeGPUType(*gtype), "count": *count, "image": *image, "region": *region}
	var inst instance
	sp := ui.StartSpinner("Provisioning…")
	err := client.Do("POST", cfg.ComputeURL, "/v1/compute/instances", cfg.Token,
		map[string]string{"Idempotency-Key": newIdempotencyKey()}, body, &inst)
	sp.Stop()
	if err != nil {
		return err
	}
	fmt.Printf("%s Instance %s — %s (%d × %s), image %s\n", ui.Green("✓"), inst.ID, inst.State, inst.Count, inst.GPUType, inst.Image)
	printConnect(inst)
	return nil
}

// gpuList lists the tenant's instances, optionally filtered by --state.
func gpuList(cfg config.Config, args []string) error {
	if err := requireToken(cfg); err != nil {
		return err
	}
	fs := flag.NewFlagSet("list", flag.ExitOnError)
	state := fs.String("state", "", "filter by state (running|stopped|…)")
	_ = fs.Parse(args)
	path := "/v1/compute/instances"
	if *state != "" {
		path += "?state=" + *state
	}
	var out struct {
		Instances []instance `json:"instances"`
	}
	if err := client.Do("GET", cfg.ComputeURL, path, cfg.Token, nil, nil, &out); err != nil {
		return err
	}
	if len(out.Instances) == 0 {
		fmt.Println("No instances.")
		return nil
	}
	fmt.Printf("%-12s %-10s %-5s %-12s %-22s %s\n", "ID", "TYPE", "GPUS", "STATE", "IMAGE", "REGION")
	for _, i := range out.Instances {
		fmt.Printf("%-12s %-10s %-5d %-12s %-22s %s\n", i.ID, i.GPUType, i.Count, i.State, i.Image, i.Region)
	}
	return nil
}

// gpuGet shows one instance with its connection info.
func gpuGet(cfg config.Config, args []string) error {
	if err := requireToken(cfg); err != nil {
		return err
	}
	if len(args) < 1 {
		return errors.New("usage: exascale gpu get <id>")
	}
	var inst instance
	if err := client.Do("GET", cfg.ComputeURL, "/v1/compute/instances/"+args[0], cfg.Token, nil, nil, &inst); err != nil {
		return err
	}
	fmt.Printf("%-10s %s\n%-10s %s\n%-10s %d\n%-10s %s\n%-10s %s\n%-10s %s\n",
		"id", inst.ID, "type", inst.GPUType, "gpus", inst.Count,
		"state", inst.State, "image", inst.Image, "region", inst.Region)
	printConnect(inst)
	return nil
}

// gpuAction runs a stop/start action on an instance.
func gpuAction(cfg config.Config, action string, args []string) error {
	if err := requireToken(cfg); err != nil {
		return err
	}
	if len(args) < 1 {
		return fmt.Errorf("usage: exascale gpu %s <id>", action)
	}
	var inst instance
	if err := client.Do("POST", cfg.ComputeURL, "/v1/compute/instances/"+args[0]+"/"+action, cfg.Token, nil, nil, &inst); err != nil {
		return err
	}
	fmt.Printf("Instance %s — %s\n", inst.ID, inst.State)
	if inst.State == "running" {
		printConnect(inst)
	}
	return nil
}

// gpuDelete terminates an instance.
func gpuDelete(cfg config.Config, args []string) error {
	if err := requireToken(cfg); err != nil {
		return err
	}
	yes, rest := stripYes(args)
	if len(rest) < 1 {
		return errors.New("usage: exascale gpu delete <id> [--yes]")
	}
	if !yes && !ui.Confirm("Terminate instance "+rest[0]+"? This frees its GPUs and is permanent.") {
		fmt.Fprintln(os.Stderr, "aborted")
		return nil
	}
	var inst instance
	if err := client.Do("DELETE", cfg.ComputeURL, "/v1/compute/instances/"+rest[0], cfg.Token, nil, nil, &inst); err != nil {
		return err
	}
	fmt.Printf("%s Instance %s — %s\n", ui.Green("✓"), inst.ID, inst.State)
	return nil
}

// stripYes removes --yes/-y from args and reports whether it was present (skip the [y/N] confirm).
func stripYes(args []string) (bool, []string) {
	yes := false
	rest := make([]string, 0, len(args))
	for _, a := range args {
		if a == "--yes" || a == "-y" {
			yes = true
			continue
		}
		rest = append(rest, a)
	}
	return yes, rest
}

// printConnect prints the connection endpoints when present.
func printConnect(inst instance) {
	if inst.Connect.SSH == "" && inst.Connect.Jupyter == "" && inst.Connect.HTTP == "" {
		return
	}
	fmt.Println("Connect:")
	if inst.Connect.SSH != "" {
		fmt.Printf("  ssh      %s\n", inst.Connect.SSH)
	}
	if inst.Connect.Jupyter != "" {
		fmt.Printf("  jupyter  %s\n", inst.Connect.Jupyter)
	}
	if inst.Connect.HTTP != "" {
		fmt.Printf("  http     %s\n", inst.Connect.HTTP)
	}
}
