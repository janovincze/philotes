package contabo

import (
	"fmt"

	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"

	"github.com/janovincze/philotes/deployments/pulumi/pkg/provider"
)

// CreateVolume creates block storage for Contabo.
// Note: Contabo VPS plans include storage by default. Additional block storage
// must be purchased separately through the control panel. This implementation
// documents the storage allocation rather than provisioning new storage.
func (p *Provider) CreateVolume(ctx *pulumi.Context, name string, sizeGB int, opts provider.VolumeOptions) (*provider.VolumeResult, error) {
	ctx.Log.Warn(fmt.Sprintf("Contabo: %dGB storage should be provisioned via control panel. VPS includes default storage.", sizeGB), nil)

	// Return synthetic ID - actual storage is part of VPS
	return &provider.VolumeResult{
		VolumeID: pulumi.ID(fmt.Sprintf("%s-volume-%dGB", name, sizeGB)).ToIDOutput(),
	}, nil
}
