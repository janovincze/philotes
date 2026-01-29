// Package installer provides provider configuration and deployment orchestration
// for the one-click cloud installer.
package installer

import (
	"github.com/janovincze/philotes/internal/api/models"
)

// GetProviders returns all supported cloud providers with their configurations.
func GetProviders() []models.Provider {
	return []models.Provider{
		getHetznerProvider(),
		getScalewayProvider(),
		getOVHProvider(),
		getExoscaleProvider(),
		getContaboProvider(),
	}
}

// GetProvider returns a single provider by ID.
func GetProvider(id string) *models.Provider {
	providers := GetProviders()
	for i := range providers {
		if providers[i].ID == id {
			return &providers[i]
		}
	}
	return nil
}

// getHetznerProvider returns the Hetzner Cloud provider configuration.
func getHetznerProvider() models.Provider {
	return models.Provider{
		ID:             "hetzner",
		Name:           "Hetzner Cloud",
		Description:    "German cloud provider with excellent price/performance ratio. Data centers in Germany and Finland.",
		LogoURL:        "/images/providers/hetzner.svg",
		OAuthSupported: true,
		Regions: []models.ProviderRegion{
			{ID: "nbg1", Name: "Nuremberg", Location: "Germany", IsDefault: true, IsAvailable: true},
			{ID: "fsn1", Name: "Falkenstein", Location: "Germany", IsAvailable: true},
			{ID: "hel1", Name: "Helsinki", Location: "Finland", IsAvailable: true},
		},
		Sizes: []models.ProviderSize{
			{
				ID:               models.DeploymentSizeSmall,
				Name:             "Small",
				Description:      "Suitable for development and small workloads",
				MonthlyCostEUR:   calculateHetznerCost("cpx21", "cpx21", 2, 50),
				ControlPlaneType: "cpx21",
				WorkerType:       "cpx21",
				WorkerCount:      2,
				StorageSizeGB:    50,
				VCPU:             6, // 2 (CP) + 2*2 (workers)
				MemoryGB:         12,
			},
			{
				ID:               models.DeploymentSizeMedium,
				Name:             "Medium",
				Description:      "Suitable for production workloads with moderate traffic",
				MonthlyCostEUR:   calculateHetznerCost("cpx31", "cpx31", 3, 100),
				ControlPlaneType: "cpx31",
				WorkerType:       "cpx31",
				WorkerCount:      3,
				StorageSizeGB:    100,
				VCPU:             16, // 4 (CP) + 3*4 (workers)
				MemoryGB:         32,
			},
			{
				ID:               models.DeploymentSizeLarge,
				Name:             "Large",
				Description:      "Suitable for high-traffic production workloads",
				MonthlyCostEUR:   calculateHetznerCost("cpx41", "cpx41", 5, 200),
				ControlPlaneType: "cpx41",
				WorkerType:       "cpx41",
				WorkerCount:      5,
				StorageSizeGB:    200,
				VCPU:             48, // 8 (CP) + 5*8 (workers)
				MemoryGB:         96,
			},
		},
	}
}

// calculateHetznerCost calculates the total monthly cost for a Hetzner deployment.
func calculateHetznerCost(cpType, workerType string, workerCount, storageGB int) float64 {
	cpCost := hetznerServerCosts[cpType]
	workerCost := hetznerServerCosts[workerType] * float64(workerCount)
	storageCost := float64(storageGB) * 0.047 // €0.047/GB/month
	lbCost := 5.39                            // lb11

	return cpCost + workerCost + storageCost + lbCost
}

// hetznerServerCosts maps server types to monthly costs in EUR.
var hetznerServerCosts = map[string]float64{
	"cx22":  4.35,
	"cx32":  7.59,
	"cx42":  14.69,
	"cx52":  28.49,
	"cpx21": 4.35,
	"cpx31": 7.99,
	"cpx41": 14.99,
	"cpx51": 28.99,
}

// getScalewayProvider returns the Scaleway provider configuration.
func getScalewayProvider() models.Provider {
	return models.Provider{
		ID:             "scaleway",
		Name:           "Scaleway",
		Description:    "French cloud provider with strong European presence. Data centers in Paris, Amsterdam, and Warsaw.",
		LogoURL:        "/images/providers/scaleway.svg",
		OAuthSupported: false, // Scaleway does not support OAuth
		Regions: []models.ProviderRegion{
			{ID: "fr-par", Name: "Paris", Location: "France", IsDefault: true, IsAvailable: true},
			{ID: "nl-ams", Name: "Amsterdam", Location: "Netherlands", IsAvailable: true},
			{ID: "pl-waw", Name: "Warsaw", Location: "Poland", IsAvailable: true},
		},
		Sizes: []models.ProviderSize{
			{
				ID:               models.DeploymentSizeSmall,
				Name:             "Small",
				Description:      "Suitable for development and small workloads",
				MonthlyCostEUR:   calculateScalewayCost("DEV1-S", "DEV1-S", 2, 50),
				ControlPlaneType: "DEV1-S",
				WorkerType:       "DEV1-S",
				WorkerCount:      2,
				StorageSizeGB:    50,
				VCPU:             6,
				MemoryGB:         6,
			},
			{
				ID:               models.DeploymentSizeMedium,
				Name:             "Medium",
				Description:      "Suitable for production workloads with moderate traffic",
				MonthlyCostEUR:   calculateScalewayCost("DEV1-M", "DEV1-M", 3, 100),
				ControlPlaneType: "DEV1-M",
				WorkerType:       "DEV1-M",
				WorkerCount:      3,
				StorageSizeGB:    100,
				VCPU:             12,
				MemoryGB:         16,
			},
			{
				ID:               models.DeploymentSizeLarge,
				Name:             "Large",
				Description:      "Suitable for high-traffic production workloads",
				MonthlyCostEUR:   calculateScalewayCost("DEV1-L", "DEV1-L", 5, 200),
				ControlPlaneType: "DEV1-L",
				WorkerType:       "DEV1-L",
				WorkerCount:      5,
				StorageSizeGB:    200,
				VCPU:             24,
				MemoryGB:         40,
			},
		},
	}
}

// calculateScalewayCost calculates the total monthly cost for a Scaleway deployment.
func calculateScalewayCost(cpType, workerType string, workerCount, storageGB int) float64 {
	cpCost := scalewayServerCosts[cpType]
	workerCost := scalewayServerCosts[workerType] * float64(workerCount)
	storageCost := float64(storageGB) * 0.08 // €0.08/GB/month
	lbCost := 9.99                           // standard LB

	return cpCost + workerCost + storageCost + lbCost
}

// scalewayServerCosts maps instance types to monthly costs in EUR.
var scalewayServerCosts = map[string]float64{
	"DEV1-S":  4.99,
	"DEV1-M":  9.99,
	"DEV1-L":  17.99,
	"DEV1-XL": 35.99,
	"GP1-XS":  19.99,
	"GP1-S":   39.99,
}

// getOVHProvider returns the OVHcloud provider configuration.
func getOVHProvider() models.Provider {
	return models.Provider{
		ID:             "ovh",
		Name:           "OVHcloud",
		Description:    "Major European cloud provider headquartered in France. Extensive European data center network.",
		LogoURL:        "/images/providers/ovh.svg",
		OAuthSupported: true,
		Regions: []models.ProviderRegion{
			{ID: "gra", Name: "Gravelines", Location: "France", IsDefault: true, IsAvailable: true},
			{ID: "sbg", Name: "Strasbourg", Location: "France", IsAvailable: true},
			{ID: "rbx", Name: "Roubaix", Location: "France", IsAvailable: true},
			{ID: "bhs", Name: "Beauharnois", Location: "Canada", IsAvailable: true},
			{ID: "waw", Name: "Warsaw", Location: "Poland", IsAvailable: true},
			{ID: "de1", Name: "Frankfurt", Location: "Germany", IsAvailable: true},
		},
		Sizes: []models.ProviderSize{
			{
				ID:               models.DeploymentSizeSmall,
				Name:             "Small",
				Description:      "Suitable for development and small workloads",
				MonthlyCostEUR:   calculateOVHCost("d2-2", "d2-2", 2, 50),
				ControlPlaneType: "d2-2",
				WorkerType:       "d2-2",
				WorkerCount:      2,
				StorageSizeGB:    50,
				VCPU:             6,
				MemoryGB:         6,
			},
			{
				ID:               models.DeploymentSizeMedium,
				Name:             "Medium",
				Description:      "Suitable for production workloads with moderate traffic",
				MonthlyCostEUR:   calculateOVHCost("d2-4", "d2-4", 3, 100),
				ControlPlaneType: "d2-4",
				WorkerType:       "d2-4",
				WorkerCount:      3,
				StorageSizeGB:    100,
				VCPU:             16,
				MemoryGB:         16,
			},
			{
				ID:               models.DeploymentSizeLarge,
				Name:             "Large",
				Description:      "Suitable for high-traffic production workloads",
				MonthlyCostEUR:   calculateOVHCost("d2-8", "b2-7", 5, 200),
				ControlPlaneType: "d2-8",
				WorkerType:       "b2-7",
				WorkerCount:      5,
				StorageSizeGB:    200,
				VCPU:             43,
				MemoryGB:         59,
			},
		},
	}
}

// calculateOVHCost calculates the total monthly cost for an OVH deployment.
func calculateOVHCost(cpType, workerType string, workerCount, storageGB int) float64 {
	cpCost := ovhServerCosts[cpType]
	workerCost := ovhServerCosts[workerType] * float64(workerCount)
	storageCost := float64(storageGB) * 0.06 // €0.06/GB/month
	lbCost := 9.99                           // small load balancer

	return cpCost + workerCost + storageCost + lbCost
}

// ovhServerCosts maps instance types to monthly costs in EUR.
var ovhServerCosts = map[string]float64{
	"d2-2":  6.00,
	"d2-4":  12.00,
	"d2-8":  24.00,
	"b2-7":  26.00,
	"b2-15": 52.00,
	"b2-30": 104.00,
	"b2-60": 208.00,
}

// getExoscaleProvider returns the Exoscale provider configuration.
func getExoscaleProvider() models.Provider {
	return models.Provider{
		ID:             "exoscale",
		Name:           "Exoscale",
		Description:    "Swiss cloud provider with focus on privacy and compliance. GDPR-friendly with Swiss data protection laws.",
		LogoURL:        "/images/providers/exoscale.svg",
		OAuthSupported: false, // Exoscale does not support OAuth
		Regions: []models.ProviderRegion{
			{ID: "ch-gva-2", Name: "Geneva", Location: "Switzerland", IsDefault: true, IsAvailable: true},
			{ID: "ch-dk-2", Name: "Zurich", Location: "Switzerland", IsAvailable: true},
			{ID: "de-fra-1", Name: "Frankfurt", Location: "Germany", IsAvailable: true},
			{ID: "de-muc-1", Name: "Munich", Location: "Germany", IsAvailable: true},
			{ID: "at-vie-1", Name: "Vienna", Location: "Austria", IsAvailable: true},
			{ID: "bg-sof-1", Name: "Sofia", Location: "Bulgaria", IsAvailable: true},
		},
		Sizes: []models.ProviderSize{
			{
				ID:               models.DeploymentSizeSmall,
				Name:             "Small",
				Description:      "Suitable for development and small workloads",
				MonthlyCostEUR:   calculateExoscaleCost("standard.micro", "standard.micro", 2, 50),
				ControlPlaneType: "standard.micro",
				WorkerType:       "standard.micro",
				WorkerCount:      2,
				StorageSizeGB:    50,
				VCPU:             3,
				MemoryGB:         3,
			},
			{
				ID:               models.DeploymentSizeMedium,
				Name:             "Medium",
				Description:      "Suitable for production workloads with moderate traffic",
				MonthlyCostEUR:   calculateExoscaleCost("standard.tiny", "standard.tiny", 3, 100),
				ControlPlaneType: "standard.tiny",
				WorkerType:       "standard.tiny",
				WorkerCount:      3,
				StorageSizeGB:    100,
				VCPU:             8,
				MemoryGB:         16,
			},
			{
				ID:               models.DeploymentSizeLarge,
				Name:             "Large",
				Description:      "Suitable for high-traffic production workloads",
				MonthlyCostEUR:   calculateExoscaleCost("standard.small", "standard.small", 5, 200),
				ControlPlaneType: "standard.small",
				WorkerType:       "standard.small",
				WorkerCount:      5,
				StorageSizeGB:    200,
				VCPU:             24,
				MemoryGB:         48,
			},
		},
	}
}

// calculateExoscaleCost calculates the total monthly cost for an Exoscale deployment.
func calculateExoscaleCost(cpType, workerType string, workerCount, storageGB int) float64 {
	cpCost := exoscaleServerCosts[cpType]
	workerCost := exoscaleServerCosts[workerType] * float64(workerCount)
	storageCost := float64(storageGB) * 0.10 // €0.10/GB/month
	lbCost := 15.00                          // NLB base cost

	return cpCost + workerCost + storageCost + lbCost
}

// exoscaleServerCosts maps instance types to monthly costs in EUR.
var exoscaleServerCosts = map[string]float64{
	"standard.micro":       7.00,
	"standard.tiny":        14.00,
	"standard.small":       28.00,
	"standard.medium":      56.00,
	"standard.large":       112.00,
	"standard.extra-large": 224.00,
}

// getContaboProvider returns the Contabo provider configuration.
func getContaboProvider() models.Provider {
	return models.Provider{
		ID:             "contabo",
		Name:           "Contabo",
		Description:    "German hosting provider with extremely competitive pricing. Great for budget-conscious deployments.",
		LogoURL:        "/images/providers/contabo.svg",
		OAuthSupported: false, // Contabo does not support OAuth
		Regions: []models.ProviderRegion{
			{ID: "EU", Name: "Europe", Location: "Germany", IsDefault: true, IsAvailable: true},
			{ID: "US-central", Name: "US Central", Location: "United States", IsAvailable: true},
			{ID: "US-east", Name: "US East", Location: "United States", IsAvailable: true},
			{ID: "US-west", Name: "US West", Location: "United States", IsAvailable: true},
			{ID: "SIN", Name: "Singapore", Location: "Singapore", IsAvailable: true},
			{ID: "AUS", Name: "Australia", Location: "Australia", IsAvailable: true},
		},
		Sizes: []models.ProviderSize{
			{
				ID:               models.DeploymentSizeSmall,
				Name:             "Small",
				Description:      "Suitable for development and small workloads. Storage included.",
				MonthlyCostEUR:   calculateContaboCost("VPS-S", "VPS-S", 2),
				ControlPlaneType: "VPS-S",
				WorkerType:       "VPS-S",
				WorkerCount:      2,
				StorageSizeGB:    200, // Storage included in VPS plans
				VCPU:             12,  // 4 (CP) + 2*4 (workers)
				MemoryGB:         24,  // 8 (CP) + 2*8 (workers)
			},
			{
				ID:               models.DeploymentSizeMedium,
				Name:             "Medium",
				Description:      "Suitable for production workloads with moderate traffic. Storage included.",
				MonthlyCostEUR:   calculateContaboCost("VPS-M", "VPS-M", 3),
				ControlPlaneType: "VPS-M",
				WorkerType:       "VPS-M",
				WorkerCount:      3,
				StorageSizeGB:    400, // Storage included in VPS plans
				VCPU:             24,  // 6 (CP) + 3*6 (workers)
				MemoryGB:         64,  // 16 (CP) + 3*16 (workers)
			},
			{
				ID:               models.DeploymentSizeLarge,
				Name:             "Large",
				Description:      "Suitable for high-traffic production workloads. Storage included.",
				MonthlyCostEUR:   calculateContaboCost("VPS-L", "VPS-L", 5),
				ControlPlaneType: "VPS-L",
				WorkerType:       "VPS-L",
				WorkerCount:      5,
				StorageSizeGB:    800, // Storage included in VPS plans
				VCPU:             48,  // 8 (CP) + 5*8 (workers)
				MemoryGB:         180, // 30 (CP) + 5*30 (workers)
			},
		},
	}
}

// calculateContaboCost calculates the total monthly cost for a Contabo deployment.
// Note: Contabo has no managed LB and storage is included in VPS plans.
func calculateContaboCost(cpType, workerType string, workerCount int) float64 {
	cpCost := contaboServerCosts[cpType]
	workerCost := contaboServerCosts[workerType] * float64(workerCount)
	// Storage is included in VPS plans
	// No managed LB available, uses ingress controller

	return cpCost + workerCost
}

// contaboServerCosts maps VPS plan types to monthly costs in EUR.
var contaboServerCosts = map[string]float64{
	"VPS-S":   4.99,  // 4 vCPU, 8GB RAM, 200GB SSD
	"VPS-M":   8.99,  // 6 vCPU, 16GB RAM, 400GB SSD
	"VPS-L":   14.99, // 8 vCPU, 30GB RAM, 800GB SSD
	"VPS-XL":  26.99, // 10 vCPU, 60GB RAM, 1.6TB SSD
	"VPS-XXL": 38.99, // 12 vCPU, 120GB RAM, 3.2TB SSD
}

// ValidateProvider checks if the given provider ID is supported.
func ValidateProvider(providerID string) bool {
	validProviders := map[string]bool{
		"hetzner":  true,
		"scaleway": true,
		"ovh":      true,
		"exoscale": true,
		"contabo":  true,
	}
	return validProviders[providerID]
}

// ValidateRegion checks if the given region is valid for the provider.
func ValidateRegion(providerID, regionID string) bool {
	provider := GetProvider(providerID)
	if provider == nil {
		return false
	}
	for _, region := range provider.Regions {
		if region.ID == regionID && region.IsAvailable {
			return true
		}
	}
	return false
}

// GetDefaultRegion returns the default region for a provider.
func GetDefaultRegion(providerID string) string {
	provider := GetProvider(providerID)
	if provider == nil {
		return ""
	}
	for _, region := range provider.Regions {
		if region.IsDefault {
			return region.ID
		}
	}
	if len(provider.Regions) > 0 {
		return provider.Regions[0].ID
	}
	return ""
}

// GetSizeConfig returns the size configuration for a provider and size ID.
func GetSizeConfig(providerID string, sizeID models.DeploymentSize) *models.ProviderSize {
	provider := GetProvider(providerID)
	if provider == nil {
		return nil
	}
	for i := range provider.Sizes {
		if provider.Sizes[i].ID == sizeID {
			return &provider.Sizes[i]
		}
	}
	return nil
}
