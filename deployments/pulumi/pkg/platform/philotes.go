package platform

import (
	"github.com/pulumi/pulumi-kubernetes/sdk/v4/go/kubernetes"
	helmv4 "github.com/pulumi/pulumi-kubernetes/sdk/v4/go/kubernetes/helm/v4"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"

	"github.com/janovincze/philotes/deployments/pulumi/pkg/config"
)

// DeployPhilotes deploys the Philotes umbrella Helm chart.
func DeployPhilotes(ctx *pulumi.Context, cfg *config.Config, k8s *kubernetes.Provider) (*helmv4.Chart, error) {
	// Environment-specific values
	values := pulumi.Map{
		"global": pulumi.Map{
			"environment": pulumi.String(cfg.Environment),
		},
		"philotes-worker": pulumi.Map{
			"replicaCount": pulumi.Int(1),
			"resources": pulumi.Map{
				"requests": pulumi.Map{
					"cpu":    pulumi.String("100m"),
					"memory": pulumi.String("256Mi"),
				},
				"limits": pulumi.Map{
					"cpu":    pulumi.String("1000m"),
					"memory": pulumi.String("1Gi"),
				},
			},
			"autoscaling": pulumi.Map{
				"enabled": pulumi.Bool(true),
				"keda": pulumi.Map{
					"enabled": pulumi.Bool(true),
				},
			},
		},
		"philotes-api": pulumi.Map{
			"replicaCount": pulumi.Int(2),
			"resources": pulumi.Map{
				"requests": pulumi.Map{
					"cpu":    pulumi.String("100m"),
					"memory": pulumi.String("128Mi"),
				},
				"limits": pulumi.Map{
					"cpu":    pulumi.String("500m"),
					"memory": pulumi.String("512Mi"),
				},
			},
		},
		"philotes-dashboard": pulumi.Map{
			"replicaCount": pulumi.Int(1),
			"resources": pulumi.Map{
				"requests": pulumi.Map{
					"cpu":    pulumi.String("50m"),
					"memory": pulumi.String("64Mi"),
				},
				"limits": pulumi.Map{
					"cpu":    pulumi.String("200m"),
					"memory": pulumi.String("256Mi"),
				},
			},
		},
		"postgresql": pulumi.Map{
			"enabled": pulumi.Bool(true),
		},
		"minio": pulumi.Map{
			"enabled": pulumi.Bool(true),
			"persistence": pulumi.Map{
				"size": pulumi.String("50Gi"),
			},
		},
	}

	// Deploy from local chart or OCI registry depending on environment
	// For local development, use the charts directory relative to repo root
	// For production, this would use an OCI registry URL
	chartPath := "../charts/philotes"
	if cfg.Environment == "production" || cfg.Environment == "staging" {
		// In production/staging, use versioned chart from registry
		// TODO: Configure OCI registry URL when available
		chartPath = "../charts/philotes"
	}

	chart, err := helmv4.NewChart(ctx, cfg.ResourceName("philotes"), &helmv4.ChartArgs{
		Chart:     pulumi.String(chartPath),
		Values:    values,
		Namespace: pulumi.String("philotes"),
	}, pulumi.Provider(k8s))
	if err != nil {
		return nil, err
	}

	return chart, nil
}
