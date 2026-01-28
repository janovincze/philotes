package scaleway

import (
	"fmt"

	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
	"github.com/pulumiverse/pulumi-scaleway/sdk/go/scaleway"

	"github.com/janovincze/philotes/deployments/pulumi/pkg/provider"
)

// CreateVolume creates a Scaleway block storage volume.
func (p *Provider) CreateVolume(ctx *pulumi.Context, name string, sizeGB int, opts provider.VolumeOptions) (*provider.VolumeResult, error) {
	zone := regionToZone(p.region)

	volume, err := scaleway.NewInstanceVolume(ctx, name, &scaleway.InstanceVolumeArgs{
		Name:     pulumi.String(name),
		SizeInGb: pulumi.Int(sizeGB),
		Type:     pulumi.String("b_ssd"),
		Zone:     pulumi.String(zone),
		Tags:     pulumi.ToStringArray([]string{"managed-by=pulumi", "project=philotes"}),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create volume: %w", err)
	}

	return &provider.VolumeResult{
		VolumeID: volume.ID(),
	}, nil
}
