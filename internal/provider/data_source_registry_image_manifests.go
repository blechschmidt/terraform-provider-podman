package provider

import (
	"context"
	"encoding/base64"
	"encoding/json"

	"github.com/docker/docker/api/types/registry"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func dataSourcePodmanRegistryImageManifests() *schema.Resource {
	return &schema.Resource{
		Description: "Reads the manifest list from a container registry for a given image.",
		ReadContext: dataSourcePodmanRegistryImageManifestsRead,
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
				Description: "Whether to skip TLS verification.",
			},
			"auth_config": {
				Type:        schema.TypeList,
				Optional:    true,
				MaxItems:    1,
				Description: "Authentication configuration for the registry.",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"address": {
							Type:        schema.TypeString,
							Required:    true,
							Description: "The address of the registry.",
						},
						"username": {
							Type:        schema.TypeString,
							Required:    true,
							Description: "The username for the registry.",
						},
						"password": {
							Type:        schema.TypeString,
							Required:    true,
							Sensitive:   true,
							Description: "The password for the registry.",
						},
					},
				},
			},
			"id": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The ID of the data source.",
			},
			"manifests": {
				Type:        schema.TypeSet,
				Computed:    true,
				Description: "The list of manifests for the image.",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"architecture": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "The CPU architecture.",
						},
						"media_type": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "The media type of the manifest.",
						},
						"os": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "The operating system.",
						},
						"sha256_digest": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "The content digest of the manifest.",
						},
					},
				},
			},
		},
	}
}

func dataSourcePodmanRegistryImageManifestsRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	config := getClient(meta)
	cli := config.Client

	name := d.Get("name").(string)

	authStr, err := resolveManifestsAuth(config, d, name)
	if err != nil {
		return diag.Errorf("error getting registry auth for %s: %s", name, err)
	}

	dist, err := cli.DistributionInspect(ctx, name, authStr)
	if err != nil {
		return diag.Errorf("error inspecting registry image %s: %s", name, err)
	}

	d.SetId(name)

	manifests := make([]interface{}, 0, len(dist.Platforms))
	for i, platform := range dist.Platforms {
		m := map[string]interface{}{
			"architecture":  platform.Architecture,
			"media_type":    dist.Descriptor.MediaType,
			"os":            platform.OS,
			"sha256_digest": string(dist.Descriptor.Digest),
		}
		// If there are multiple platforms, each may have a distinct digest.
		// The distribution inspect API returns a single descriptor but multiple platforms.
		// We use the top-level digest; individual platform digests are not exposed.
		_ = i
		manifests = append(manifests, m)
	}
	if err := d.Set("manifests", manifests); err != nil {
		return diag.FromErr(err)
	}

	return nil
}

// resolveManifestsAuth determines the auth string to use for the manifests data source.
// If an auth_config block is provided, it takes precedence over the provider-level registry_auth.
func resolveManifestsAuth(config *ProviderConfig, d *schema.ResourceData, imageName string) (string, error) {
	if v, ok := d.GetOk("auth_config"); ok {
		authList := v.([]interface{})
		if len(authList) > 0 && authList[0] != nil {
			authMap := authList[0].(map[string]interface{})
			authConfig := registry.AuthConfig{
				Username:      authMap["username"].(string),
				Password:      authMap["password"].(string),
				ServerAddress: authMap["address"].(string),
			}
			encoded, err := json.Marshal(authConfig)
			if err != nil {
				return "", err
			}
			return base64.URLEncoding.EncodeToString(encoded), nil
		}
	}

	return getRegistryAuth(config, imageName)
}
