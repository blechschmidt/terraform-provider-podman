package provider

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/docker/docker/api/types/image"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func resourcePodmanRegistryImage() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourcePodmanRegistryImageCreate,
		ReadContext:   resourcePodmanRegistryImageRead,
		UpdateContext: resourcePodmanRegistryImageUpdate,
		DeleteContext: resourcePodmanRegistryImageDelete,

		Schema: map[string]*schema.Schema{
			"name": {
				Type:             schema.TypeString,
				Required:         true,
				Description:      "The name of the image to push to the registry.",
				DiffSuppressFunc: suppressIfIDOrNameEqual,
			},
			"insecure_skip_verify": {
				Type:        schema.TypeBool,
				Optional:    true,
				Default:     false,
				Description: "If true, the verification of TLS certificates of the server/registry is disabled.",
			},
			"keep_remotely": {
				Type:        schema.TypeBool,
				Optional:    true,
				Default:     false,
				Description: "If true, the image will not be deleted from the registry on destroy.",
			},
			"triggers": {
				Type:     schema.TypeMap,
				Optional: true,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
				Description: "A map of arbitrary strings that, when changed, will force the image to be re-pushed.",
			},
			"sha256_digest": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The sha256 digest of the image in the registry.",
			},
		},
	}
}

func resourcePodmanRegistryImageCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	config := getClient(meta)
	imageName := d.Get("name").(string)

	digest, err := pushImage(ctx, config, imageName)
	if err != nil {
		return diag.FromErr(err)
	}

	d.SetId(imageName)
	d.Set("sha256_digest", digest)

	return nil
}

func resourcePodmanRegistryImageRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	config := getClient(meta)
	cli := config.Client
	imageName := d.Get("name").(string)

	inspectResult, _, err := cli.ImageInspectWithRaw(ctx, imageName)
	if err != nil {
		d.SetId("")
		return nil
	}

	if len(inspectResult.RepoDigests) > 0 {
		for _, rd := range inspectResult.RepoDigests {
			parts := strings.SplitN(rd, "@", 2)
			if len(parts) == 2 {
				d.Set("sha256_digest", parts[1])
				break
			}
		}
	}

	return nil
}

func resourcePodmanRegistryImageUpdate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	if d.HasChange("triggers") || d.HasChange("name") {
		config := getClient(meta)
		imageName := d.Get("name").(string)

		digest, err := pushImage(ctx, config, imageName)
		if err != nil {
			return diag.FromErr(err)
		}

		d.Set("sha256_digest", digest)
	}

	return resourcePodmanRegistryImageRead(ctx, d, meta)
}

func resourcePodmanRegistryImageDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	if d.Get("keep_remotely").(bool) {
		return nil
	}

	config := getClient(meta)
	imageName := d.Get("name").(string)
	digest := d.Get("sha256_digest").(string)

	if digest == "" {
		return nil
	}

	insecure := d.Get("insecure_skip_verify").(bool)

	err := deleteRegistryImage(config, imageName, digest, insecure)
	if err != nil {
		return diag.FromErr(fmt.Errorf("unable to delete image from registry: %w", err))
	}

	return nil
}

func pushImage(ctx context.Context, config *ProviderConfig, imageName string) (string, error) {
	cli := config.Client

	pushOpts := image.PushOptions{}

	authStr, err := getRegistryAuth(config, imageName)
	if err != nil {
		return "", fmt.Errorf("unable to get registry auth: %w", err)
	}
	if authStr != "" {
		pushOpts.RegistryAuth = authStr
	}

	reader, err := cli.ImagePush(ctx, imageName, pushOpts)
	if err != nil {
		return "", fmt.Errorf("unable to push image %s: %w", imageName, err)
	}
	defer reader.Close()

	type pushMessage struct {
		Status string `json:"status"`
		Error  string `json:"error"`
		Aux    *struct {
			Tag    string `json:"Tag"`
			Digest string `json:"Digest"`
			Size   int64  `json:"Size"`
		} `json:"aux"`
	}

	var digest string
	decoder := json.NewDecoder(reader)
	for {
		var msg pushMessage
		if err := decoder.Decode(&msg); err != nil {
			if err == io.EOF {
				break
			}
			return "", fmt.Errorf("error reading push output: %w", err)
		}
		if msg.Error != "" {
			return "", fmt.Errorf("push error: %s", msg.Error)
		}
		if msg.Aux != nil && msg.Aux.Digest != "" {
			digest = msg.Aux.Digest
		}
	}

	return digest, nil
}

func deleteRegistryImage(config *ProviderConfig, imageName string, digest string, insecure bool) error {
	registryHost := getRegistryFromImageName(imageName)

	// Extract the repository name (without the registry host and tag)
	repo := imageName
	parts := strings.SplitN(imageName, "/", 2)
	if len(parts) == 2 && (strings.Contains(parts[0], ".") || strings.Contains(parts[0], ":") || parts[0] == "localhost") {
		repo = parts[1]
	}

	// Remove tag from repo if present
	if idx := strings.LastIndex(repo, ":"); idx >= 0 {
		lastSlash := strings.LastIndex(repo, "/")
		if idx > lastSlash {
			repo = repo[:idx]
		}
	}

	scheme := "https"
	if insecure {
		scheme = "http"
	}

	url := fmt.Sprintf("%s://%s/v2/%s/manifests/%s", scheme, registryHost, repo, digest)

	req, err := http.NewRequest("DELETE", url, nil)
	if err != nil {
		return fmt.Errorf("failed to create delete request: %w", err)
	}

	req.Header.Set("Accept", "application/vnd.docker.distribution.manifest.v2+json")

	if auth, ok := config.RegistryAuth[registryHost]; ok && auth.Username != "" && auth.Password != "" {
		req.SetBasicAuth(auth.Username, auth.Password)
	}

	httpClient := &http.Client{}
	if insecure {
		httpClient.Transport = &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true}, //nolint:gosec
		}
	}

	resp, err := httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to delete image from registry: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusAccepted && resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNotFound {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("registry returned status %d: %s", resp.StatusCode, string(body))
	}

	return nil
}
