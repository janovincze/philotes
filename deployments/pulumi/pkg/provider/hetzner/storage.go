package hetzner

import (
	"fmt"

	"github.com/pulumi/pulumi-hcloud/sdk/go/hcloud"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"

	"github.com/janovincze/philotes/deployments/pulumi/pkg/provider"
)

// CreateVolume creates a Hetzner Cloud block storage volume.
func (p *Provider) CreateVolume(ctx *pulumi.Context, name string, sizeGB int, opts provider.VolumeOptions) (*provider.VolumeResult, error) {
	region := opts.Region
	if region == "" {
		region = p.region
	}

	volumeArgs := &hcloud.VolumeArgs{
		Name:     pulumi.String(name),
		Size:     pulumi.Int(sizeGB),
		Location: pulumi.String(region),
		Format:   pulumi.String("ext4"),
		Labels: pulumi.StringMap{
			"managed-by": pulumi.String("pulumi"),
			"project":    pulumi.String("philotes"),
		},
	}

	// Attach to server if specified
	if opts.ServerID != (pulumi.IDOutput{}) {
		volumeArgs.ServerId = opts.ServerID.ToStringOutput().ApplyT(func(id string) int {
			var i int
			fmt.Sscanf(id, "%d", &i)
			return i
		}).(pulumi.IntOutput)
	}

	volume, err := hcloud.NewVolume(ctx, name, volumeArgs)
	if err != nil {
		return nil, fmt.Errorf("failed to create volume: %w", err)
	}

	return &provider.VolumeResult{
		VolumeID: volume.ID(),
	}, nil
}
