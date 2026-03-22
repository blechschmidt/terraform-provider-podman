package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func dataSourcePodmanImage() *schema.Resource {
	return &schema.Resource{
		Description: "Reads the image metadata from the Podman host.",
		ReadContext: dataSourcePodmanImageRead,
		Schema: map[string]*schema.Schema{
			"name": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "The name of the image, such as `nginx:latest`.",
			},
			"id": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The ID of the image.",
			},
			"repo_digest": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The image repo digest.",
			},
		},
	}
}

func dataSourcePodmanImageRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	config := getClient(meta)
	cli := config.Client

	name := d.Get("name").(string)

	image, _, err := cli.ImageInspectWithRaw(ctx, name)
	if err != nil {
		return diag.Errorf("error inspecting image %s: %s", name, err)
	}

	d.SetId(image.ID)

	repoDigest := ""
	if len(image.RepoDigests) > 0 {
		repoDigest = image.RepoDigests[0]
	}
	if err := d.Set("repo_digest", repoDigest); err != nil {
		return diag.FromErr(err)
	}

	return nil
}
