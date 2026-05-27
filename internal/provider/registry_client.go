package provider

import (
	"context"
	"crypto/tls"
	"encoding/base64"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/docker/docker/api/types/registry"
)

// registryHostAndRepo parses an image reference into the registry host (as it
// should appear in HTTP URLs — docker.io maps to registry-1.docker.io) and the
// repository path, dropping any tag/digest.
func registryHostAndRepo(name string) (host, repo, ref string) {
	// Capture digest or tag for later (used by manifests endpoint).
	if i := strings.Index(name, "@"); i >= 0 {
		ref = name[i+1:]
		name = name[:i]
	} else {
		if i := strings.LastIndex(name, ":"); i >= 0 && !strings.Contains(name[i:], "/") {
			ref = name[i+1:]
			name = name[:i]
		}
	}
	if ref == "" {
		ref = "latest"
	}
	parts := strings.SplitN(name, "/", 2)
	if len(parts) == 1 {
		return "registry-1.docker.io", "library/" + parts[0], ref
	}
	if strings.ContainsAny(parts[0], ".:") || parts[0] == "localhost" {
		host = parts[0]
		repo = parts[1]
		if host == "docker.io" {
			host = "registry-1.docker.io"
		}
		return host, repo, ref
	}
	return "registry-1.docker.io", name, ref
}

// registryHTTPClient builds an HTTP client appropriate for the given insecure
// flag (TLS verification skipped when true).
func registryHTTPClient(insecure bool) *http.Client {
	return &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: insecure}, //nolint:gosec
		},
		Timeout: 30 * time.Second,
	}
}

// registryManifestRequest constructs a manifests request with auth, choosing
// HTTP or HTTPS, and automatically falling back to the opposite scheme on
// TLS/HTTP mismatch. Returns the response which the caller must close.
func registryManifestRequest(ctx context.Context, method, host, repo, ref string, basicAuth, bearerAuth string, insecure bool) (*http.Response, error) {
	cli := registryHTTPClient(insecure)
	do := func(scheme string) (*http.Response, error) {
		url := fmt.Sprintf("%s://%s/v2/%s/manifests/%s", scheme, host, repo, ref)
		req, err := http.NewRequestWithContext(ctx, method, url, nil)
		if err != nil {
			return nil, err
		}
		req.Header.Set("Accept", strings.Join([]string{
			"application/vnd.oci.image.index.v1+json",
			"application/vnd.docker.distribution.manifest.list.v2+json",
			"application/vnd.oci.image.manifest.v1+json",
			"application/vnd.docker.distribution.manifest.v2+json",
		}, ", "))
		if basicAuth != "" {
			req.Header.Set("Authorization", "Basic "+basicAuth)
		}
		if bearerAuth != "" {
			req.Header.Set("Authorization", "Bearer "+bearerAuth)
		}
		return cli.Do(req)
	}

	scheme := "https"
	if insecure && (host == "localhost" || strings.HasPrefix(host, "localhost:") || strings.HasPrefix(host, "127.0.0.1")) {
		scheme = "http"
	}
	resp, err := do(scheme)
	if err != nil {
		if scheme == "https" && strings.Contains(err.Error(), "http: server gave HTTP response to HTTPS client") {
			return do("http")
		}
		if scheme == "http" && strings.Contains(err.Error(), "tls: first record does not look like a TLS handshake") {
			return do("https")
		}
		return nil, err
	}

	// 401: try a bearer challenge if we don't already have one.
	if resp.StatusCode == http.StatusUnauthorized && bearerAuth == "" {
		challenge := resp.Header.Get("WWW-Authenticate")
		resp.Body.Close()
		token, terr := fetchBearerToken(ctx, cli, challenge)
		if terr == nil && token != "" {
			return registryManifestRequest(ctx, method, host, repo, ref, basicAuth, token, insecure)
		}
	}
	return resp, nil
}

// buildBasicAuthHeader returns the base64-encoded "user:pass" string for HTTP
// Basic Auth — empty if either field is missing.
func buildBasicAuthHeader(user, pass string) string {
	if user == "" && pass == "" {
		return ""
	}
	return base64.StdEncoding.EncodeToString([]byte(user + ":" + pass))
}

// resolveBasicAuthForImage returns the basic-auth header for a configured
// registry credential. Empty if none.
func resolveBasicAuthForImage(config *ProviderConfig, imageName string) string {
	host, _, _ := registryHostAndRepo(imageName)
	// Try the host we resolve to in URLs, plus the literal as written.
	if ra, ok := config.RegistryAuth[host]; ok {
		return buildBasicAuthHeader(ra.Username, ra.Password)
	}
	if ra, ok := config.RegistryAuth[getRegistryFromImageName(imageName)]; ok {
		return buildBasicAuthHeader(ra.Username, ra.Password)
	}
	return ""
}

// registryDigest returns the digest of a manifest in the registry using a
// HEAD request, plus the manifest body and Content-Type for the GET variant.
type registryManifest struct {
	Digest      string
	ContentType string
	Body        []byte
}

func fetchRegistryManifest(ctx context.Context, host, repo, ref, basicAuth string, insecure bool) (*registryManifest, error) {
	resp, err := registryManifestRequest(ctx, http.MethodGet, host, repo, ref, basicAuth, "", insecure)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("registry returned status %d: %s", resp.StatusCode, string(body))
	}
	b, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	return &registryManifest{
		Digest:      resp.Header.Get("Docker-Content-Digest"),
		ContentType: resp.Header.Get("Content-Type"),
		Body:        b,
	}, nil
}

// encodeBasicAuthFromRegistryConfig converts a registry.AuthConfig into a
// base64 Basic Auth header (returns empty string when no creds).
func encodeBasicAuthFromRegistryConfig(c registry.AuthConfig) string {
	return buildBasicAuthHeader(c.Username, c.Password)
}
