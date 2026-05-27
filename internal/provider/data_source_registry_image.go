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
	name := d.Get("name").(string)
	insecure := d.Get("insecure_skip_verify").(bool)

	host, repo, ref := registryHostAndRepo(name)
	basicAuth := resolveBasicAuthForImage(config, name)
	m, err := fetchRegistryManifest(ctx, host, repo, ref, basicAuth, insecure)
	if err != nil {
		return diag.Errorf("error inspecting registry image %s: %s", name, err)
	}

	d.SetId(name)
	if err := d.Set("sha256_digest", m.Digest); err != nil {
		return diag.FromErr(err)
	}
	return nil
}
