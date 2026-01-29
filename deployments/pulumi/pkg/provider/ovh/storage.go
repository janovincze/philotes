package ovh

import (
	"fmt"

	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"

	"github.com/janovincze/philotes/deployments/pulumi/pkg/provider"
)

// CreateVolume creates block storage for OVHcloud.
// Note: OVH Managed Kubernetes uses persistent volume claims (PVC) for storage.
// Block storage would be provisioned through the Kubernetes storage class.
// This returns a synthetic ID as storage is managed via K8s PVCs.
func (p *Provider) CreateVolume(ctx *pulumi.Context, name string, sizeGB int, opts provider.VolumeOptions) (*provider.VolumeResult, error) {
	// OVH Managed Kubernetes provides storage classes for PVC provisioning
	// Block storage is handled through CSI driver, not direct API calls
	// Return a synthetic ID indicating managed storage

	ctx.Log.Info(fmt.Sprintf("OVH: Storage (%dGB) will be provisioned via Kubernetes PVCs using csi-cinder-high-speed storage class", sizeGB), nil)

	return &provider.VolumeResult{
		VolumeID: pulumi.ID(fmt.Sprintf("%s-volume-%dGB", name, sizeGB)).ToIDOutput(),
	}, nil
}
