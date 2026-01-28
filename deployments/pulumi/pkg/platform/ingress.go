package platform

import (
	"github.com/pulumi/pulumi-kubernetes/sdk/v4/go/kubernetes"
	helmv4 "github.com/pulumi/pulumi-kubernetes/sdk/v4/go/kubernetes/helm/v4"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"

	"github.com/janovincze/philotes/deployments/pulumi/pkg/config"
)

// DeployIngressNginx deploys the ingress-nginx controller via Helm.
func DeployIngressNginx(ctx *pulumi.Context, cfg *config.Config, k8s *kubernetes.Provider, certManager *helmv4.Chart) (*helmv4.Chart, error) {
	var deps []pulumi.Resource
	if certManager != nil {
		deps = append(deps, certManager)
	}

	chart, err := helmv4.NewChart(ctx, cfg.ResourceName("ingress-nginx"), &helmv4.ChartArgs{
		Chart:   pulumi.String("ingress-nginx"),
		Version: pulumi.String("4.12.0"),
		RepositoryOpts: &helmv4.RepositoryOptsArgs{
			Repo: pulumi.String("https://kubernetes.github.io/ingress-nginx"),
		},
		Namespace: pulumi.String("ingress-nginx"),
		Values: pulumi.Map{
			"controller": pulumi.Map{
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
				"service": pulumi.Map{
					"type": pulumi.String("NodePort"),
				},
				"metrics": pulumi.Map{
					"enabled": pulumi.Bool(true),
				},
			},
		},
	}, pulumi.Provider(k8s), pulumi.DependsOn(deps))
	if err != nil {
		return nil, err
	}

	return chart, nil
}
