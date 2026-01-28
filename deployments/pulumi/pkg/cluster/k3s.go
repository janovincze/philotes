// Package cluster provides K3s cluster provisioning via cloud-init.
package cluster

import (
	"fmt"
	"strings"
)

// K3sVersion is the version of K3s to install.
const K3sVersion = "v1.31.4+k3s1"

// ControlPlaneCloudInit generates a cloud-init script for a K3s control plane node.
// It installs K3s server with Traefik disabled (we use ingress-nginx instead).
func ControlPlaneCloudInit(publicIP string, clusterToken string) string {
	return fmt.Sprintf(`#!/bin/bash
set -euo pipefail

# Update system
apt-get update -y
apt-get upgrade -y
apt-get install -y curl open-iscsi nfs-common

# Disable swap
swapoff -a
sed -i '/swap/d' /etc/fstab

# Configure kernel parameters for Kubernetes
cat > /etc/sysctl.d/99-kubernetes.conf << 'SYSCTL'
net.bridge.bridge-nf-call-iptables = 1
net.bridge.bridge-nf-call-ip6tables = 1
net.ipv4.ip_forward = 1
fs.inotify.max_user_instances = 8192
fs.inotify.max_user_watches = 524288
SYSCTL
sysctl --system

# Install K3s server
curl -sfL https://get.k3s.io | INSTALL_K3S_VERSION="%s" sh -s - server \
  --disable traefik \
  --disable servicelb \
  --tls-san "%s" \
  --token "%s" \
  --write-kubeconfig-mode "0644" \
  --node-label "node.kubernetes.io/role=control-plane" \
  --kubelet-arg "max-pods=110"

# Wait for K3s to be ready
echo "Waiting for K3s to be ready..."
until kubectl get nodes 2>/dev/null; do
  sleep 5
done

echo "K3s control plane installation complete."
`, K3sVersion, publicIP, clusterToken)
}

// WorkerCloudInit generates a cloud-init script for a K3s worker node.
// It installs K3s agent and joins the control plane.
func WorkerCloudInit(controlPlaneIP string, clusterToken string) string {
	return fmt.Sprintf(`#!/bin/bash
set -euo pipefail

# Update system
apt-get update -y
apt-get upgrade -y
apt-get install -y curl open-iscsi nfs-common

# Disable swap
swapoff -a
sed -i '/swap/d' /etc/fstab

# Configure kernel parameters for Kubernetes
cat > /etc/sysctl.d/99-kubernetes.conf << 'SYSCTL'
net.bridge.bridge-nf-call-iptables = 1
net.bridge.bridge-nf-call-ip6tables = 1
net.ipv4.ip_forward = 1
fs.inotify.max_user_instances = 8192
fs.inotify.max_user_watches = 524288
SYSCTL
sysctl --system

# Wait for control plane to be reachable
echo "Waiting for control plane at %s..."
until curl -sk https://%s:6443/ping 2>/dev/null; do
  sleep 10
done

# Install K3s agent
curl -sfL https://get.k3s.io | INSTALL_K3S_VERSION="%s" K3S_URL="https://%s:6443" K3S_TOKEN="%s" sh -s - agent \
  --node-label "node.kubernetes.io/role=worker" \
  --kubelet-arg "max-pods=110"

echo "K3s worker node installation complete."
`, controlPlaneIP, controlPlaneIP, K3sVersion, controlPlaneIP, clusterToken)
}

// GenerateClusterToken generates a deterministic cluster token from the environment name.
func GenerateClusterToken(environment string) string {
	// Use a deterministic but reasonably complex token based on the environment name.
	// In a real deployment, this should be a secret managed by Pulumi.
	return fmt.Sprintf("philotes-%s-k3s-token", strings.ReplaceAll(environment, " ", "-"))
}
