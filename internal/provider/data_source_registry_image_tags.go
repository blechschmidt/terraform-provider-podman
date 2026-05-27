package provider

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/hashicorp/go-version"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func dataSourcePodmanRegistryImageTags() *schema.Resource {
	return &schema.Resource{
		Description: "Lists all tags for a repository in a container registry.",
		ReadContext: dataSourcePodmanRegistryImageTagsRead,
		Schema: map[string]*schema.Schema{
			"name": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "The repository name, such as `nginx` or `quay.io/podman/stable`.",
			},
			"insecure_skip_verify": {
				Type:        schema.TypeBool,
				Optional:    true,
				Default:     false,
				Description: "Whether to skip TLS verification.",
			},
			"strict_semver": {
				Type:        schema.TypeBool,
				Optional:    true,
				Default:     false,
				Description: "If true, only return tags that parse as valid semantic versions.",
			},
			"id": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"tags": {
				Type:        schema.TypeList,
				Computed:    true,
				Elem:        &schema.Schema{Type: schema.TypeString},
				Description: "All tags for the named repository.",
			},
		},
	}
}

func dataSourcePodmanRegistryImageTagsRead(ctx context.Context, d *schema.ResourceData, _ interface{}) diag.Diagnostics {
	name := d.Get("name").(string)
	insecure := d.Get("insecure_skip_verify").(bool)
	strict := d.Get("strict_semver").(bool)

	host, repo := splitRegistryRepo(name)
	scheme := "https"
	if insecure || strings.HasPrefix(host, "localhost") || strings.HasPrefix(host, "127.0.0.1") {
		// keep https first; the transport below disables verify when insecure is set
	}
	url := fmt.Sprintf("%s://%s/v2/%s/tags/list", scheme, host, repo)

	tr := &http.Transport{}
	if insecure {
		tr.TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
	}
	cli := &http.Client{Transport: tr, Timeout: 30 * time.Second}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return diag.FromErr(err)
	}
	resp, err := cli.Do(req)
	if err != nil {
		return diag.FromErr(fmt.Errorf("error querying registry tags: %w", err))
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusUnauthorized {
		// Honor a Bearer challenge for public docker.io reads.
		token, terr := fetchBearerToken(ctx, cli, resp.Header.Get("WWW-Authenticate"))
		if terr != nil {
			return diag.FromErr(fmt.Errorf("registry %s requires authentication and bearer fetch failed: %w", host, terr))
		}
		req, _ = http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
		req.Header.Set("Authorization", "Bearer "+token)
		resp, err = cli.Do(req)
		if err != nil {
			return diag.FromErr(err)
		}
		defer resp.Body.Close()
	}

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return diag.Errorf("registry %s returned status %d for tags list: %s", host, resp.StatusCode, string(body))
	}

	var payload struct {
		Name string   `json:"name"`
		Tags []string `json:"tags"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return diag.FromErr(fmt.Errorf("error decoding registry response: %w", err))
	}

	tags := payload.Tags
	if strict {
		filtered := tags[:0]
		for _, t := range tags {
			if _, err := version.NewSemver(t); err == nil {
				filtered = append(filtered, t)
			}
		}
		tags = filtered
	}
	sort.Strings(tags)

	if err := d.Set("tags", tags); err != nil {
		return diag.FromErr(err)
	}
	d.SetId(name)
	return nil
}

// splitRegistryRepo splits an image reference like "quay.io/foo/bar:tag" into
// the registry host and the repo path. Defaults to docker.io and prepends the
// "library/" namespace for short refs (e.g. "alpine").
func splitRegistryRepo(name string) (host, repo string) {
	// Strip optional tag/digest
	if i := strings.IndexAny(name, "@"); i >= 0 {
		name = name[:i]
	}
	if i := strings.LastIndex(name, ":"); i >= 0 && !strings.Contains(name[i:], "/") {
		name = name[:i]
	}
	parts := strings.SplitN(name, "/", 2)
	if len(parts) == 1 {
		return "registry-1.docker.io", "library/" + parts[0]
	}
	if strings.ContainsAny(parts[0], ".:") || parts[0] == "localhost" {
		host = parts[0]
		repo = parts[1]
		if host == "docker.io" {
			host = "registry-1.docker.io"
		}
		return
	}
	return "registry-1.docker.io", name
}

// fetchBearerToken parses a "Www-Authenticate: Bearer realm=...,service=...,scope=..."
// challenge and returns a token from the realm.
func fetchBearerToken(ctx context.Context, cli *http.Client, challenge string) (string, error) {
	if !strings.HasPrefix(strings.ToLower(challenge), "bearer ") {
		return "", fmt.Errorf("unsupported challenge: %s", challenge)
	}
	rest := strings.TrimSpace(challenge[len("bearer "):])
	params := map[string]string{}
	for _, kv := range strings.Split(rest, ",") {
		eq := strings.IndexByte(kv, '=')
		if eq < 0 {
			continue
		}
		key := strings.TrimSpace(kv[:eq])
		val := strings.Trim(strings.TrimSpace(kv[eq+1:]), `"`)
		params[key] = val
	}
	realm := params["realm"]
	if realm == "" {
		return "", fmt.Errorf("no realm in challenge")
	}
	q := []string{}
	if s := params["service"]; s != "" {
		q = append(q, "service="+s)
	}
	if s := params["scope"]; s != "" {
		q = append(q, "scope="+s)
	}
	tokenURL := realm
	if len(q) > 0 {
		tokenURL += "?" + strings.Join(q, "&")
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, tokenURL, nil)
	if err != nil {
		return "", err
	}
	resp, err := cli.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("token endpoint %s returned %d: %s", tokenURL, resp.StatusCode, string(body))
	}
	var tr struct {
		Token       string `json:"token"`
		AccessToken string `json:"access_token"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&tr); err != nil {
		return "", err
	}
	if tr.Token != "" {
		return tr.Token, nil
	}
	return tr.AccessToken, nil
}
