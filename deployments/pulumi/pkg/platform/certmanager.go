package platform

import (
	"github.com/pulumi/pulumi-kubernetes/sdk/v4/go/kubernetes"
	helmv4 "github.com/pulumi/pulumi-kubernetes/sdk/v4/go/kubernetes/helm/v4"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"

	"github.com/janovincze/philotes/deployments/pulumi/pkg/config"
)

// DeployCertManager deploys cert-manager via Helm for TLS certificate management.
func DeployCertManager(ctx *pulumi.Context, cfg *config.Config, k8s *kubernetes.Provider) (*helmv4.Chart, error) {
	chart, err := helmv4.NewChart(ctx, cfg.ResourceName("cert-manager"), &helmv4.ChartArgs{
		Chart:   pulumi.String("cert-manager"),
		Version: pulumi.String("v1.16.3"),
		RepositoryOpts: &helmv4.RepositoryOptsArgs{
			Repo: pulumi.String("https://charts.jetstack.io"),
		},
		Namespace: pulumi.String("cert-manager"),
		Values: pulumi.Map{
			"installCRDs": pulumi.Bool(true),
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
	}, pulumi.Provider(k8s))
	if err != nil {
		return nil, err
	}

	return chart, nil
}
