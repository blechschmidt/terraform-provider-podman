package provider

import (
	"context"
	"encoding/json"

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
	name := d.Get("name").(string)
	insecure := d.Get("insecure_skip_verify").(bool)

	basicAuth := resolveBasicAuthForManifests(config, d, name)
	host, repo, ref := registryHostAndRepo(name)
	m, err := fetchRegistryManifest(ctx, host, repo, ref, basicAuth, insecure)
	if err != nil {
		return diag.Errorf("error inspecting registry image %s: %s", name, err)
	}

	d.SetId(name)

	// Try to parse as an index/manifest list. Fall back to a single entry
	// (single-platform manifest) if it doesn't parse as a list.
	var index struct {
		Manifests []struct {
			Digest    string `json:"digest"`
			MediaType string `json:"mediaType"`
			Platform  struct {
				Architecture string `json:"architecture"`
				OS           string `json:"os"`
			} `json:"platform"`
		} `json:"manifests"`
	}
	manifests := make([]interface{}, 0)
	if err := json.Unmarshal(m.Body, &index); err == nil && len(index.Manifests) > 0 {
		for _, e := range index.Manifests {
			manifests = append(manifests, map[string]interface{}{
				"architecture":  e.Platform.Architecture,
				"media_type":    e.MediaType,
				"os":            e.Platform.OS,
				"sha256_digest": e.Digest,
			})
		}
	} else {
		// Single manifest — try to read the architecture/os from the config blob via
		// the manifest's `config.architecture`/`os` if present.
		var single struct {
			Architecture string `json:"architecture"`
			OS           string `json:"os"`
		}
		_ = json.Unmarshal(m.Body, &single)
		manifests = append(manifests, map[string]interface{}{
			"architecture":  single.Architecture,
			"media_type":    m.ContentType,
			"os":            single.OS,
			"sha256_digest": m.Digest,
		})
	}
	if err := d.Set("manifests", manifests); err != nil {
		return diag.FromErr(err)
	}

	return nil
}

// resolveBasicAuthForManifests returns the HTTP Basic auth string to use for
// the manifests data source. If an auth_config block is provided, it takes
// precedence over the provider-level registry_auth.
func resolveBasicAuthForManifests(config *ProviderConfig, d *schema.ResourceData, imageName string) string {
	if v, ok := d.GetOk("auth_config"); ok {
		authList := v.([]interface{})
		if len(authList) > 0 && authList[0] != nil {
			authMap := authList[0].(map[string]interface{})
			return buildBasicAuthHeader(authMap["username"].(string), authMap["password"].(string))
		}
	}
	return resolveBasicAuthForImage(config, imageName)
}
