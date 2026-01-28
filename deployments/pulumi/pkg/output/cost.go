// Package output provides cost estimation and deployment output formatting.
package output

import (
	"fmt"

	"github.com/janovincze/philotes/deployments/pulumi/pkg/config"
)

// CostEstimate holds monthly cost estimates for a deployment.
type CostEstimate struct {
	// Provider is the cloud provider name.
	Provider string
	// ControlPlane is the monthly cost of the control plane node.
	ControlPlane float64
	// Workers is the monthly cost of all worker nodes.
	Workers float64
	// Storage is the monthly cost of block storage.
	Storage float64
	// LoadBalancer is the monthly cost of the load balancer.
	LoadBalancer float64
	// Total is the total monthly cost.
	Total float64
	// Currency is the cost currency (EUR).
	Currency string
}

// Summary returns a human-readable cost summary.
func (c *CostEstimate) Summary() string {
	return fmt.Sprintf(
		"%s estimated monthly cost: %.2f %s (control-plane: %.2f, workers: %.2f, storage: %.2f, lb: %.2f)",
		c.Provider, c.Total, c.Currency,
		c.ControlPlane, c.Workers, c.Storage, c.LoadBalancer,
	)
}

// EstimateCost calculates the estimated monthly cost for a deployment.
func EstimateCost(providerName string, cfg *config.Config) *CostEstimate {
	switch providerName {
	case "hetzner":
		return estimateHetzner(cfg)
	case "scaleway":
		return estimateScaleway(cfg)
	default:
		return &CostEstimate{
			Provider: providerName,
			Currency: "EUR",
		}
	}
}

// estimateHetzner calculates costs for Hetzner Cloud.
// Prices as of 2024 (EUR/month, incl. VAT).
func estimateHetzner(cfg *config.Config) *CostEstimate {
	cpCost := hetznerServerCost(cfg.ControlPlaneType)
	workerCost := hetznerServerCost(cfg.WorkerType) * float64(cfg.WorkerCount)
	storageCost := float64(cfg.StorageSizeGB) * 0.047 // €0.047/GB/month
	lbCost := 5.39                                    // lb11

	return &CostEstimate{
		Provider:     "hetzner",
		ControlPlane: cpCost,
		Workers:      workerCost,
		Storage:      storageCost,
		LoadBalancer: lbCost,
		Total:        cpCost + workerCost + storageCost + lbCost,
		Currency:     "EUR",
	}
}

// hetznerServerCost returns the monthly cost for a Hetzner server type.
func hetznerServerCost(serverType string) float64 {
	costs := map[string]float64{
		"cx22":  4.35,
		"cx32":  7.59,
		"cx42":  14.69,
		"cx52":  28.49,
		"cpx21": 4.35,
		"cpx31": 7.99,
		"cpx41": 14.99,
		"cpx51": 28.99,
	}
	if cost, ok := costs[serverType]; ok {
		return cost
	}
	return 10.0 // conservative default
}

// estimateScaleway calculates costs for Scaleway.
// Prices as of 2024 (EUR/month, incl. VAT).
func estimateScaleway(cfg *config.Config) *CostEstimate {
	cpCost := scalewayServerCost(cfg.ControlPlaneType)
	workerCost := scalewayServerCost(cfg.WorkerType) * float64(cfg.WorkerCount)
	storageCost := float64(cfg.StorageSizeGB) * 0.08 // €0.08/GB/month
	lbCost := 9.99                                   // standard LB

	return &CostEstimate{
		Provider:     "scaleway",
		ControlPlane: cpCost,
		Workers:      workerCost,
		Storage:      storageCost,
		LoadBalancer: lbCost,
		Total:        cpCost + workerCost + storageCost + lbCost,
		Currency:     "EUR",
	}
}

// scalewayServerCost returns the monthly cost for a Scaleway instance type.
func scalewayServerCost(instanceType string) float64 {
	costs := map[string]float64{
		"DEV1-S":  4.99,
		"DEV1-M":  9.99,
		"DEV1-L":  17.99,
		"DEV1-XL": 35.99,
		"GP1-XS":  19.99,
		"GP1-S":   39.99,
	}
	if cost, ok := costs[instanceType]; ok {
		return cost
	}
	return 15.0 // conservative default
}
