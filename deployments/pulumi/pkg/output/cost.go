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
	case "ovh":
		return estimateOVH(cfg)
	case "exoscale":
		return estimateExoscale(cfg)
	case "contabo":
		return estimateContabo(cfg)
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

// estimateOVH calculates costs for OVHcloud.
// Prices as of 2024 (EUR/month, incl. VAT).
func estimateOVH(cfg *config.Config) *CostEstimate {
	cpCost := ovhServerCost(cfg.ControlPlaneType)
	workerCost := ovhServerCost(cfg.WorkerType) * float64(cfg.WorkerCount)
	storageCost := float64(cfg.StorageSizeGB) * 0.06 // €0.06/GB/month for high-speed
	lbCost := 9.99                                   // small load balancer

	return &CostEstimate{
		Provider:     "ovh",
		ControlPlane: cpCost,
		Workers:      workerCost,
		Storage:      storageCost,
		LoadBalancer: lbCost,
		Total:        cpCost + workerCost + storageCost + lbCost,
		Currency:     "EUR",
	}
}

// ovhServerCost returns the monthly cost for an OVH instance type.
func ovhServerCost(instanceType string) float64 {
	costs := map[string]float64{
		"d2-2":  6.00,
		"d2-4":  12.00,
		"d2-8":  24.00,
		"b2-7":  26.00,
		"b2-15": 52.00,
		"b2-30": 104.00,
		"b2-60": 208.00,
	}
	if cost, ok := costs[instanceType]; ok {
		return cost
	}
	return 15.0 // conservative default
}

// estimateExoscale calculates costs for Exoscale.
// Prices as of 2024 (EUR/month, incl. VAT).
func estimateExoscale(cfg *config.Config) *CostEstimate {
	cpCost := exoscaleServerCost(cfg.ControlPlaneType)
	workerCost := exoscaleServerCost(cfg.WorkerType) * float64(cfg.WorkerCount)
	storageCost := float64(cfg.StorageSizeGB) * 0.10 // €0.10/GB/month for block storage
	lbCost := 15.00                                   // NLB base cost

	return &CostEstimate{
		Provider:     "exoscale",
		ControlPlane: cpCost,
		Workers:      workerCost,
		Storage:      storageCost,
		LoadBalancer: lbCost,
		Total:        cpCost + workerCost + storageCost + lbCost,
		Currency:     "EUR",
	}
}

// exoscaleServerCost returns the monthly cost for an Exoscale instance type.
func exoscaleServerCost(instanceType string) float64 {
	costs := map[string]float64{
		"standard.micro":       7.00,
		"standard.tiny":        14.00,
		"standard.small":       28.00,
		"standard.medium":      56.00,
		"standard.large":       112.00,
		"standard.extra-large": 224.00,
	}
	if cost, ok := costs[instanceType]; ok {
		return cost
	}
	return 50.0 // conservative default
}

// estimateContabo calculates costs for Contabo.
// Prices as of 2024 (EUR/month, incl. VAT).
// Note: Contabo has no managed LB; cost is €0 for LB (uses ingress controller).
func estimateContabo(cfg *config.Config) *CostEstimate {
	cpCost := contaboServerCost(cfg.ControlPlaneType)
	workerCost := contaboServerCost(cfg.WorkerType) * float64(cfg.WorkerCount)
	storageCost := 0.0 // Storage included in VPS plans
	lbCost := 0.0      // No managed LB, uses ingress controller

	return &CostEstimate{
		Provider:     "contabo",
		ControlPlane: cpCost,
		Workers:      workerCost,
		Storage:      storageCost,
		LoadBalancer: lbCost,
		Total:        cpCost + workerCost + storageCost + lbCost,
		Currency:     "EUR",
	}
}

// contaboServerCost returns the monthly cost for a Contabo VPS plan.
// Contabo offers extremely competitive pricing for European hosting.
func contaboServerCost(planType string) float64 {
	costs := map[string]float64{
		"VPS-S":   4.99,  // 4 vCPU, 8GB RAM, 200GB SSD
		"VPS-M":   8.99,  // 6 vCPU, 16GB RAM, 400GB SSD
		"VPS-L":   14.99, // 8 vCPU, 30GB RAM, 800GB SSD
		"VPS-XL":  26.99, // 10 vCPU, 60GB RAM, 1.6TB SSD
		"VPS-XXL": 38.99, // 12 vCPU, 120GB RAM, 3.2TB SSD
	}
	if cost, ok := costs[planType]; ok {
		return cost
	}
	return 10.0 // conservative default
}
