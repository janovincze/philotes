// Package platform orchestrates the full Philotes deployment on Kubernetes.
package platform

import (
	"fmt"

	"github.com/pulumi/pulumi-kubernetes/sdk/v4/go/kubernetes"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"

	"github.com/janovincze/philotes/deployments/pulumi/pkg/cluster"
	"github.com/janovincze/philotes/deployments/pulumi/pkg/config"
	"github.com/janovincze/philotes/deployments/pulumi/pkg/provider"
)

// DeployResult contains the outputs of a full platform deployment.
type DeployResult struct {
	// ControlPlaneIP is the public IP of the control plane node.
	ControlPlaneIP pulumi.StringOutput
	// LoadBalancerIP is the public IP of the load balancer.
	LoadBalancerIP pulumi.StringOutput
	// Kubeconfig is the kubeconfig for the deployed cluster.
	Kubeconfig pulumi.StringOutput
}

// Deploy orchestrates the full deployment:
// 1. Create cloud infrastructure (network, firewall, servers, storage, LB)
// 2. Bootstrap K3s cluster
// 3. Retrieve kubeconfig
// 4. Deploy platform Helm charts (cert-manager, ingress-nginx, monitoring, philotes)
func Deploy(ctx *pulumi.Context, cp provider.CloudProvider, cfg *config.Config) (*DeployResult, error) {
	// --- Cloud Infrastructure ---

	// Create private network
	network, err := cp.CreateNetwork(ctx, cfg.ResourceName("network"), provider.NetworkOptions{
		CIDRBlock:  "10.0.0.0/16",
		SubnetCIDR: "10.0.1.0/24",
		Region:     cfg.Region,
	})
	if err != nil {
		return nil, fmt.Errorf("network creation failed: %w", err)
	}

	// Create firewall
	firewall, err := cp.CreateFirewall(ctx, cfg.ResourceName("firewall"), provider.DefaultFirewallRules())
	if err != nil {
		return nil, fmt.Errorf("firewall creation failed: %w", err)
	}

	// Generate cluster token
	clusterToken := cluster.GenerateClusterToken(cfg.Environment)

	// Create control plane node
	// Note: We use a placeholder IP in the cloud-init TLS-SAN. K3s will automatically
	// add the server's actual IP to the TLS SANs when it starts. The --tls-san flag
	// in cloud-init is for additional SANs (like a domain name).
	controlPlane, err := cp.CreateServer(ctx, cfg.ResourceName("control-plane"), provider.ServerOptions{
		ServerType:   cfg.ControlPlaneType,
		Region:       cfg.Region,
		SSHPublicKey: cfg.SSHPublicKey,
		UserData:     pulumi.String(cluster.ControlPlaneCloudInit("0.0.0.0", clusterToken)),
		NetworkID:    network.NetworkID,
		FirewallID:   firewall.FirewallID,
		Labels: map[string]string{
			"role":       "control-plane",
			"managed-by": "pulumi",
			"project":    "philotes",
			"env":        cfg.Environment,
		},
	})
	if err != nil {
		return nil, fmt.Errorf("control plane creation failed: %w", err)
	}

	// Create worker nodes
	// Workers dynamically discover the control plane's private IP using Pulumi's Apply pattern.
	// This ensures workers always join the correct control plane, regardless of IP assignment.
	var workerServerIDs []pulumi.IDOutput
	for i := 0; i < cfg.WorkerCount; i++ {
		workerName := fmt.Sprintf("%s-worker-%d", cfg.ResourceName(""), i)

		// Generate cloud-init dynamically based on control plane's actual private IP
		workerUserData := controlPlane.PrivateIP.ApplyT(func(ip interface{}) string {
			return cluster.WorkerCloudInit(ip.(string), clusterToken)
		}).(pulumi.StringOutput)

		worker, workerErr := cp.CreateServer(ctx, workerName, provider.ServerOptions{
			ServerType:   cfg.WorkerType,
			Region:       cfg.Region,
			SSHPublicKey: cfg.SSHPublicKey,
			UserData:     workerUserData,
			NetworkID:    network.NetworkID,
			FirewallID:   firewall.FirewallID,
			Labels: map[string]string{
				"role":       "worker",
				"managed-by": "pulumi",
				"project":    "philotes",
				"env":        cfg.Environment,
			},
		})
		if workerErr != nil {
			return nil, fmt.Errorf("worker %d creation failed: %w", i, workerErr)
		}
		workerServerIDs = append(workerServerIDs, worker.ServerID)
	}

	// Create block storage volume for MinIO
	_, err = cp.CreateVolume(ctx, cfg.ResourceName("storage"), cfg.StorageSizeGB, provider.VolumeOptions{
		Region:   cfg.Region,
		ServerID: controlPlane.ServerID,
	})
	if err != nil {
		return nil, fmt.Errorf("storage creation failed: %w", err)
	}

	// Create load balancer
	lb, err := cp.CreateLoadBalancer(ctx, cfg.ResourceName("lb"), provider.LBOptions{
		Region:          cfg.Region,
		NetworkID:       network.NetworkID,
		TargetServerIDs: workerServerIDs,
		Ports:           provider.DefaultLBPorts(),
	})
	if err != nil {
		return nil, fmt.Errorf("load balancer creation failed: %w", err)
	}

	// --- K3s Cluster Bootstrap ---

	// Retrieve kubeconfig from control plane
	kubeconfig, err := cluster.GetKubeconfig(ctx, cfg.ResourceName("cluster"), cluster.KubeconfigOptions{
		ControlPlaneIP:    controlPlane.PublicIP,
		SSHPrivateKeyPath: cfg.SSHPrivateKeyPath,
	})
	if err != nil {
		return nil, fmt.Errorf("kubeconfig retrieval failed: %w", err)
	}

	// --- Kubernetes Provider & Helm Charts ---

	// Create Kubernetes provider from kubeconfig
	k8sProvider, err := kubernetes.NewProvider(ctx, cfg.ResourceName("k8s"), &kubernetes.ProviderArgs{
		Kubeconfig: kubeconfig,
	})
	if err != nil {
		return nil, fmt.Errorf("k8s provider creation failed: %w", err)
	}

	// Deploy cert-manager
	certManager, err := DeployCertManager(ctx, cfg, k8sProvider)
	if err != nil {
		return nil, fmt.Errorf("cert-manager deployment failed: %w", err)
	}

	// Deploy ingress-nginx (depends on cert-manager)
	_, err = DeployIngressNginx(ctx, cfg, k8sProvider, certManager)
	if err != nil {
		return nil, fmt.Errorf("ingress-nginx deployment failed: %w", err)
	}

	// Deploy monitoring stack
	_, err = DeployMonitoring(ctx, cfg, k8sProvider)
	if err != nil {
		return nil, fmt.Errorf("monitoring deployment failed: %w", err)
	}

	// Deploy Philotes application
	_, err = DeployPhilotes(ctx, cfg, k8sProvider)
	if err != nil {
		return nil, fmt.Errorf("philotes deployment failed: %w", err)
	}

	return &DeployResult{
		ControlPlaneIP: controlPlane.PublicIP,
		LoadBalancerIP: lb.PublicIP,
		Kubeconfig:     kubeconfig,
	}, nil
}
