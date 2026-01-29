package exoscale

import (
	"fmt"

	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
	"github.com/pulumiverse/pulumi-exoscale/sdk/go/exoscale"

	"github.com/janovincze/philotes/deployments/pulumi/pkg/provider"
)

// CreateVolume creates an Exoscale block storage volume.
func (p *Provider) CreateVolume(ctx *pulumi.Context, name string, sizeGB int, opts provider.VolumeOptions) (*provider.VolumeResult, error) {
	zone := opts.Region
	if zone == "" {
		zone = p.zone
	}

	// Create block storage volume
	volume, err := exoscale.NewBlockStorageVolume(ctx, name, &exoscale.BlockStorageVolumeArgs{
		Zone: pulumi.String(zone),
		Name: pulumi.String(name),
		Size: pulumi.Int(sizeGB),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create volume: %w", err)
	}

	// Note: Volume attachment to instances is done at instance creation time
	// via the BlockStorageVolumeIds field, or through cloud-init to mount the volume.
	// Exoscale volumes are automatically available once created in the same zone.
	if opts.ServerID != (pulumi.IDOutput{}) {
		ctx.Log.Info(fmt.Sprintf("Volume %s created. Attach via instance BlockStorageVolumeIds or cloud-init", name), nil)
	}

	return &provider.VolumeResult{
		VolumeID: volume.ID(),
	}, nil
}
