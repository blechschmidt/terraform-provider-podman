package provider

import (
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

var testAccProviders map[string]*schema.Provider
var testAccProvider *schema.Provider

func init() {
	testAccProvider = New()
	testAccProviders = map[string]*schema.Provider{
		"podman": testAccProvider,
	}
}

func testAccPreCheck(t *testing.T) {
	host := os.Getenv("DOCKER_HOST")
	if host == "" {
		host = "unix:///run/user/1000/podman/podman.sock"
	}
	// Check podman socket exists
	if len(host) > 7 && host[:7] == "unix://" {
		if _, err := os.Stat(host[7:]); err != nil {
			t.Skipf("Podman socket not available at %s", host)
		}
	}
}

func providerConfig() string {
	// The test framework configures the provider automatically via Providers map.
	// We must NOT include a provider block in the HCL to avoid "Duplicate provider
	// configuration" errors. The DOCKER_HOST env var configures the host.
	return ""
}

func TestProvider(t *testing.T) {
	p := New()
	if err := p.InternalValidate(); err != nil {
		t.Fatalf("provider schema validation failed: %v", err)
	}
}

func TestProviderConfigure(t *testing.T) {
	host := os.Getenv("DOCKER_HOST")
	if host == "" {
		host = "unix:///run/user/1000/podman/podman.sock"
	}

	raw := map[string]interface{}{
		"host":          host,
		"cert_path":     "",
		"ca_material":   "",
		"cert_material": "",
		"key_material":  "",
	}

	p := New()
	d := schema.TestResourceDataRaw(t, p.Schema, raw)
	meta, diags := providerConfigure(nil, d)
	if diags.HasError() {
		t.Fatalf("provider configuration returned errors: %v", diags)
	}
	if meta == nil {
		t.Fatal("expected non-nil meta after configuration")
	}
	config, ok := meta.(*ProviderConfig)
	if !ok {
		t.Fatal("expected meta to be *ProviderConfig")
	}
	if config.Client == nil {
		t.Fatal("expected non-nil Client in ProviderConfig")
	}
}

func TestProviderSchemaHasAllResources(t *testing.T) {
	p := New()
	expectedResources := []string{
		"podman_container",
		"podman_image",
		"podman_network",
		"podman_volume",
		"podman_tag",
		"podman_plugin",
		"podman_secret",
		"podman_registry_image",
	}
	for _, name := range expectedResources {
		if _, ok := p.ResourcesMap[name]; !ok {
			t.Errorf("expected resource %q to be registered, but it was not found", name)
		}
	}
	if len(p.ResourcesMap) != len(expectedResources) {
		t.Errorf("expected %d resources, got %d", len(expectedResources), len(p.ResourcesMap))
	}
}

func TestProviderSchemaHasAllDataSources(t *testing.T) {
	p := New()
	expectedDataSources := []string{
		"podman_image",
		"podman_network",
		"podman_plugin",
		"podman_registry_image",
		"podman_logs",
		"podman_registry_image_manifests",
	}
	for _, name := range expectedDataSources {
		if _, ok := p.DataSourcesMap[name]; !ok {
			t.Errorf("expected data source %q to be registered, but it was not found", name)
		}
	}
	if len(p.DataSourcesMap) != len(expectedDataSources) {
		t.Errorf("expected %d data sources, got %d", len(expectedDataSources), len(p.DataSourcesMap))
	}
}
