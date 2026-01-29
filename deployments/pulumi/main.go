// Package main is the entry point for the Philotes infrastructure deployment.
package main

import (
	"fmt"

	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"

	"github.com/janovincze/philotes/deployments/pulumi/pkg/config"
	"github.com/janovincze/philotes/deployments/pulumi/pkg/output"
	"github.com/janovincze/philotes/deployments/pulumi/pkg/platform"
	"github.com/janovincze/philotes/deployments/pulumi/pkg/provider"
	"github.com/janovincze/philotes/deployments/pulumi/pkg/provider/contabo"
	"github.com/janovincze/philotes/deployments/pulumi/pkg/provider/exoscale"
	"github.com/janovincze/philotes/deployments/pulumi/pkg/provider/hetzner"
	"github.com/janovincze/philotes/deployments/pulumi/pkg/provider/ovh"
	"github.com/janovincze/philotes/deployments/pulumi/pkg/provider/scaleway"
)

func main() {
	pulumi.Run(func(ctx *pulumi.Context) error {
		// Load configuration
		cfg, err := config.LoadConfig(ctx)
		if err != nil {
			return fmt.Errorf("failed to load config: %w", err)
		}

		// Select cloud provider
		cloudProvider, err := selectProvider(cfg)
		if err != nil {
			return err
		}

		ctx.Log.Info(fmt.Sprintf("Deploying Philotes to %s (%s) in %s environment",
			cloudProvider.Name(), cfg.Region, cfg.Environment), nil)

		// Deploy the full platform
		result, err := platform.Deploy(ctx, cloudProvider, cfg)
		if err != nil {
			return fmt.Errorf("failed to deploy platform: %w", err)
		}

		// Export outputs
		ctx.Export("provider", pulumi.String(cloudProvider.Name()))
		ctx.Export("environment", pulumi.String(cfg.Environment))
		ctx.Export("region", pulumi.String(cfg.Region))
		ctx.Export("controlPlaneIP", result.ControlPlaneIP)
		ctx.Export("kubeconfig", pulumi.ToSecret(result.Kubeconfig).(pulumi.StringOutput))
		ctx.Export("loadBalancerIP", result.LoadBalancerIP)

		// Export cost estimate
		estimate := output.EstimateCost(cloudProvider.Name(), cfg)
		ctx.Export("estimatedMonthlyCost", pulumi.String(estimate.Summary()))

		return nil
	})
}

// selectProvider creates the appropriate cloud provider based on configuration.
func selectProvider(cfg *config.Config) (provider.CloudProvider, error) {
	switch cfg.Provider {
	case "hetzner":
		return hetzner.New(cfg.Region), nil
	case "scaleway":
		return scaleway.New(cfg.Region), nil
	case "ovh":
		return ovh.New(cfg.Region), nil
	case "exoscale":
		return exoscale.New(cfg.Region), nil
	case "contabo":
		return contabo.New(cfg.Region), nil
	default:
		return nil, fmt.Errorf("unsupported provider: %s (supported: hetzner, scaleway, ovh, exoscale, contabo)", cfg.Provider)
	}
}
