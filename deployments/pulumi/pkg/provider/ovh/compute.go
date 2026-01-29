package ovh

import (
	"fmt"

	"github.com/ovh/pulumi-ovh/sdk/go/ovh/cloudproject"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"

	"github.com/janovincze/philotes/deployments/pulumi/pkg/provider"
)

// CreateServer creates compute resources for OVHcloud.
// Since OVH Pulumi provider doesn't support raw VM instances directly,
// this uses OVH Managed Kubernetes with node pools.
// The "server" becomes a Kubernetes node pool with specified characteristics.
func (p *Provider) CreateServer(ctx *pulumi.Context, name string, opts provider.ServerOptions) (*provider.ServerResult, error) {
	serviceName, err := getServiceName(ctx)
	if err != nil {
		return nil, err
	}

	region := opts.Region
	if region == "" {
		region = p.region
	}

	flavorName := opts.ServerType
	if flavorName == "" {
		flavorName = "d2-4" // 2 vCPU, 4GB RAM
	}

	// Check if this is the first server (control plane) - create the cluster
	// Or if it's a worker, add a node pool
	isControlPlane := opts.Labels != nil && opts.Labels["role"] == "control-plane"

	if isControlPlane {
		// Create the OVH Managed Kubernetes cluster
		kube, err := cloudproject.NewKube(ctx, name, &cloudproject.KubeArgs{
			ServiceName: pulumi.String(serviceName),
			Name:        pulumi.String(name),
			Region:      pulumi.String(region),
		})
		if err != nil {
			return nil, fmt.Errorf("failed to create kubernetes cluster: %w", err)
		}

		// For OVH Managed K8s, we don't have direct public IP
		// The kubeconfig provides access
		return &provider.ServerResult{
			ServerID:  kube.ID(),
			PublicIP:  kube.KubeProxyMode.ToStringOutput(), // placeholder
			PrivateIP: pulumi.String("managed").ToStringOutput(),
			SSHKeyID:  pulumi.ID("managed").ToIDOutput(),
		}, nil
	}

	// For worker nodes, create a node pool
	// Determine the Kubernetes cluster ID for this worker node pool.
	// Expect it to be provided via labels to avoid passing new fields through ServerOptions.
	var kubeID string
	if opts.Labels != nil {
		kubeID = opts.Labels["kubeId"]
	}
	if kubeID == "" {
		return nil, fmt.Errorf("missing Kubernetes cluster ID for worker node pool (expected label \"kubeId\")")
	}

	// Create a node pool
	nodePool, err := cloudproject.NewKubeNodePool(ctx, name, &cloudproject.KubeNodePoolArgs{
		ServiceName:  pulumi.String(serviceName),
		KubeId:       pulumi.String(kubeID),
		Name:         pulumi.String(name),
		FlavorName:   pulumi.String(flavorName),
		DesiredNodes: pulumi.Int(1),
		MinNodes:     pulumi.Int(1),
		MaxNodes:     pulumi.Int(3),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create node pool: %w", err)
	}

	return &provider.ServerResult{
		ServerID:  nodePool.ID(),
		PublicIP:  pulumi.String("managed").ToStringOutput(),
		PrivateIP: pulumi.String("managed").ToStringOutput(),
		SSHKeyID:  pulumi.ID("managed").ToIDOutput(),
	}, nil
}
