// Package provider defines the cloud provider abstraction for Philotes deployments.
package provider

import (
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

// CloudProvider defines the interface that each cloud provider must implement.
type CloudProvider interface {
	// CreateNetwork creates a VPC/private network.
	CreateNetwork(ctx *pulumi.Context, name string, opts NetworkOptions) (*NetworkResult, error)

	// CreateFirewall creates firewall rules.
	CreateFirewall(ctx *pulumi.Context, name string, rules []FirewallRule) (*FirewallResult, error)

	// CreateServer creates a virtual machine.
	CreateServer(ctx *pulumi.Context, name string, opts ServerOptions) (*ServerResult, error)

	// CreateVolume creates a block storage volume.
	CreateVolume(ctx *pulumi.Context, name string, sizeGB int, opts VolumeOptions) (*VolumeResult, error)

	// CreateLoadBalancer creates a load balancer.
	CreateLoadBalancer(ctx *pulumi.Context, name string, opts LBOptions) (*LBResult, error)

	// Name returns the provider name.
	Name() string
}

// NetworkOptions configures a network.
type NetworkOptions struct {
	// CIDRBlock is the IP range for the network (e.g., "10.0.0.0/16").
	CIDRBlock string
	// SubnetCIDR is the IP range for the subnet (e.g., "10.0.1.0/24").
	SubnetCIDR string
	// Region is the cloud provider region.
	Region string
}

// NetworkResult contains the created network resources.
type NetworkResult struct {
	// NetworkID is the provider-specific network ID.
	NetworkID pulumi.IDOutput
	// SubnetID is the provider-specific subnet ID.
	SubnetID pulumi.IDOutput
}

// FirewallRule defines a single firewall rule.
type FirewallRule struct {
	// Description is a human-readable description.
	Description string
	// Direction is "in" or "out".
	Direction string
	// Protocol is "tcp", "udp", or "icmp".
	Protocol string
	// Port is the port number or range (e.g., "443" or "8000-9000").
	Port string
	// SourceIPs is a list of allowed source CIDRs.
	SourceIPs []string
}

// FirewallResult contains the created firewall resource.
type FirewallResult struct {
	// FirewallID is the provider-specific firewall ID.
	FirewallID pulumi.IDOutput
}

// ServerOptions configures a server.
type ServerOptions struct {
	// ServerType is the instance type (e.g., "cx22" for Hetzner).
	ServerType string
	// Region is the cloud provider region or zone.
	Region string
	// Image is the OS image name.
	Image string
	// SSHPublicKey is the SSH public key content.
	SSHPublicKey string
	// UserData is the cloud-init script content.
	// Accepts both string (pulumi.String) and dynamic outputs (pulumi.StringOutput).
	UserData pulumi.StringInput
	// NetworkID is the private network to attach.
	NetworkID pulumi.IDOutput
	// FirewallID is the firewall to apply.
	FirewallID pulumi.IDOutput
	// Labels are key-value labels for the server.
	Labels map[string]string
}

// ServerResult contains the created server resources.
type ServerResult struct {
	// ServerID is the provider-specific server ID.
	ServerID pulumi.IDOutput
	// PublicIP is the server's public IPv4 address.
	PublicIP pulumi.StringOutput
	// PrivateIP is the server's private network IP.
	PrivateIP pulumi.StringOutput
	// SSHKeyID is the provider-specific SSH key ID.
	SSHKeyID pulumi.IDOutput
}

// VolumeOptions configures a block storage volume.
type VolumeOptions struct {
	// Region is the cloud provider region or zone.
	Region string
	// ServerID is the server to attach the volume to (optional).
	ServerID pulumi.IDOutput
}

// VolumeResult contains the created volume resource.
type VolumeResult struct {
	// VolumeID is the provider-specific volume ID.
	VolumeID pulumi.IDOutput
}

// LBOptions configures a load balancer.
type LBOptions struct {
	// Region is the cloud provider region or zone.
	Region string
	// NetworkID is the private network to attach.
	NetworkID pulumi.IDOutput
	// TargetIPs are the backend server private IPs.
	TargetIPs []pulumi.StringOutput
	// TargetServerIDs are the backend server IDs.
	TargetServerIDs []pulumi.IDOutput
	// Ports maps frontend ports to backend ports.
	Ports []LBPort
}

// LBPort defines a load balancer port mapping.
type LBPort struct {
	// ListenPort is the frontend port.
	ListenPort int
	// TargetPort is the backend port.
	TargetPort int
	// Protocol is "tcp" or "http".
	Protocol string
}

// LBResult contains the created load balancer resource.
type LBResult struct {
	// LBID is the provider-specific load balancer ID.
	LBID pulumi.IDOutput
	// PublicIP is the load balancer's public IP.
	PublicIP pulumi.StringOutput
}

// DefaultFirewallRules returns the standard firewall rules for a Philotes cluster.
func DefaultFirewallRules() []FirewallRule {
	return []FirewallRule{
		{
			Description: "Allow SSH",
			Direction:   "in",
			Protocol:    "tcp",
			Port:        "22",
			SourceIPs:   []string{"0.0.0.0/0", "::/0"},
		},
		{
			Description: "Allow Kubernetes API",
			Direction:   "in",
			Protocol:    "tcp",
			Port:        "6443",
			SourceIPs:   []string{"0.0.0.0/0", "::/0"},
		},
		{
			Description: "Allow HTTP",
			Direction:   "in",
			Protocol:    "tcp",
			Port:        "80",
			SourceIPs:   []string{"0.0.0.0/0", "::/0"},
		},
		{
			Description: "Allow HTTPS",
			Direction:   "in",
			Protocol:    "tcp",
			Port:        "443",
			SourceIPs:   []string{"0.0.0.0/0", "::/0"},
		},
		{
			Description: "Allow Kubelet API",
			Direction:   "in",
			Protocol:    "tcp",
			Port:        "10250",
			SourceIPs:   []string{"10.0.0.0/8"},
		},
	}
}

// DefaultLBPorts returns the standard load balancer port mappings.
func DefaultLBPorts() []LBPort {
	return []LBPort{
		{ListenPort: 80, TargetPort: 80, Protocol: "tcp"},
		{ListenPort: 443, TargetPort: 443, Protocol: "tcp"},
	}
}
