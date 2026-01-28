package cluster

import (
	"fmt"
	"os"
	"strings"

	"github.com/pulumi/pulumi-command/sdk/go/command/remote"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

// KubeconfigOptions configures the kubeconfig retrieval.
type KubeconfigOptions struct {
	// ControlPlaneIP is the public IP of the control plane node.
	ControlPlaneIP pulumi.StringOutput
	// SSHPrivateKeyPath is the path to the SSH private key file (on the machine running Pulumi).
	SSHPrivateKeyPath string
	// DependsOn is a list of resources that must be created first.
	DependsOn []pulumi.Resource
}

// GetKubeconfig retrieves the kubeconfig from the K3s control plane node.
// It replaces the internal cluster address with the public IP for external access.
func GetKubeconfig(ctx *pulumi.Context, name string, opts KubeconfigOptions) (pulumi.StringOutput, error) {
	// Read the SSH private key content
	privateKeyContent, err := os.ReadFile(opts.SSHPrivateKeyPath)
	if err != nil {
		return pulumi.StringOutput{}, fmt.Errorf("failed to read SSH private key from %s: %w", opts.SSHPrivateKeyPath, err)
	}

	// Use remote command to fetch kubeconfig from the control plane
	fetchCmd, err := remote.NewCommand(ctx, name+"-kubeconfig", &remote.CommandArgs{
		Connection: &remote.ConnectionArgs{
			Host:       opts.ControlPlaneIP,
			User:       pulumi.String("root"),
			PrivateKey: pulumi.String(string(privateKeyContent)),
		},
		Create: pulumi.String("cat /etc/rancher/k3s/k3s.yaml"),
		// Triggers re-read if the control plane IP changes
		Triggers: pulumi.Array{opts.ControlPlaneIP},
	}, pulumi.DependsOn(opts.DependsOn))
	if err != nil {
		return pulumi.StringOutput{}, fmt.Errorf("failed to fetch kubeconfig: %w", err)
	}

	// Replace 127.0.0.1 with the public IP in the kubeconfig
	kubeconfig := pulumi.All(fetchCmd.Stdout, opts.ControlPlaneIP).ApplyT(
		func(args []interface{}) string {
			rawConfig, ok := args[0].(string)
			if !ok {
				return ""
			}
			publicIP, ok := args[1].(string)
			if !ok {
				return rawConfig
			}
			return replaceKubeconfigServer(rawConfig, publicIP)
		},
	).(pulumi.StringOutput)

	return kubeconfig, nil
}

// replaceKubeconfigServer replaces the server address in a kubeconfig.
func replaceKubeconfigServer(kubeconfig, publicIP string) string {
	// K3s defaults to 127.0.0.1:6443 in the kubeconfig.
	// Replace with the public IP for external access.
	result := strings.ReplaceAll(kubeconfig, "https://127.0.0.1:6443", fmt.Sprintf("https://%s:6443", publicIP))
	result = strings.ReplaceAll(result, "https://localhost:6443", fmt.Sprintf("https://%s:6443", publicIP))
	return result
}
