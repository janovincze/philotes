package platform

import (
	"github.com/pulumi/pulumi-kubernetes/sdk/v4/go/kubernetes"
	helmv4 "github.com/pulumi/pulumi-kubernetes/sdk/v4/go/kubernetes/helm/v4"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"

	"github.com/janovincze/philotes/deployments/pulumi/pkg/config"
)

// DeployMonitoring deploys the kube-prometheus-stack for monitoring and alerting.
func DeployMonitoring(ctx *pulumi.Context, cfg *config.Config, k8s *kubernetes.Provider) (*helmv4.Chart, error) {
	chart, err := helmv4.NewChart(ctx, cfg.ResourceName("monitoring"), &helmv4.ChartArgs{
		Chart:   pulumi.String("kube-prometheus-stack"),
		Version: pulumi.String("67.9.0"),
		RepositoryOpts: &helmv4.RepositoryOptsArgs{
			Repo: pulumi.String("https://prometheus-community.github.io/helm-charts"),
		},
		Namespace: pulumi.String("monitoring"),
		Values: pulumi.Map{
			"prometheus": pulumi.Map{
				"prometheusSpec": pulumi.Map{
					"retention": pulumi.String("7d"),
					"resources": pulumi.Map{
						"requests": pulumi.Map{
							"cpu":    pulumi.String("200m"),
							"memory": pulumi.String("512Mi"),
						},
						"limits": pulumi.Map{
							"cpu":    pulumi.String("1000m"),
							"memory": pulumi.String("2Gi"),
						},
					},
					"storageSpec": pulumi.Map{
						"volumeClaimTemplate": pulumi.Map{
							"spec": pulumi.Map{
								"accessModes": pulumi.ToStringArray([]string{"ReadWriteOnce"}),
								"resources": pulumi.Map{
									"requests": pulumi.Map{
										"storage": pulumi.String("10Gi"),
									},
								},
							},
						},
					},
				},
			},
			"grafana": pulumi.Map{
				"enabled": pulumi.Bool(true),
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
			"alertmanager": pulumi.Map{
				"enabled": pulumi.Bool(true),
				"alertmanagerSpec": pulumi.Map{
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
			},
		},
	}, pulumi.Provider(k8s))
	if err != nil {
		return nil, err
	}

	return chart, nil
}
