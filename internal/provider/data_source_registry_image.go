package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func dataSourcePodmanRegistryImage() *schema.Resource {
	return &schema.Resource{
		Description: "Reads the image metadata from a container registry.",
		ReadContext: dataSourcePodmanRegistryImageRead,
		Schema: map[string]*schema.Schema{
			"name": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "The name of the image, such as `nginx:latest`.",
			},
			"insecure_skip_verify": {
				Type:        schema.TypeBool,
				Optional:    true,
				Default:     false,
				Description: "Whether to skip TLS verification. Defaults to `false`.",
			},
			"id": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The ID of the data source.",
			},
			"sha256_digest": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The content digest of the image in the registry.",
			},
		},
	}
}

func dataSourcePodmanRegistryImageRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	config := getClient(meta)
	cli := config.Client

	name := d.Get("name").(string)

	authStr, err := getRegistryAuth(config, name)
	if err != nil {
		return diag.Errorf("error getting registry auth for %s: %s", name, err)
	}

	dist, err := cli.DistributionInspect(ctx, name, authStr)
	if err != nil {
		return diag.Errorf("error inspecting registry image %s: %s", name, err)
	}

	digest := string(dist.Descriptor.Digest)

	d.SetId(name)
	if err := d.Set("sha256_digest", digest); err != nil {
		return diag.FromErr(err)
	}

	return nil
}
