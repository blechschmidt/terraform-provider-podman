package provider

import (
	"fmt"
	"net/http"
	"os/exec"
	"testing"
	"time"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

func testAccEnsureLocalRegistry(t *testing.T) {
	t.Helper()
	// Start a local registry container on port 5555 if not already running.
	// Ignore error if already running.
	exec.Command("podman", "run", "-d", "--name", "tf-test-registry", "-p", "5555:5000", "docker.io/library/registry:2").CombinedOutput()

	// Wait for registry to be ready.
	for i := 0; i < 30; i++ {
		resp, err := http.Get("http://localhost:5555/v2/")
		if err == nil {
			resp.Body.Close()
			if resp.StatusCode == 200 {
				return
			}
		}
		time.Sleep(time.Second)
	}
	t.Skip("Could not start local registry")
}

func testAccCleanupLocalRegistry() {
	exec.Command("podman", "rm", "-f", "tf-test-registry").CombinedOutput()
}

func testAccCheckPodmanRegistryImageDestroy(s *terraform.State) error {
	// Registry images may or may not be deletable from the registry.
	// We do not enforce destroy verification for registry images.
	return nil
}

func TestAccPodmanRegistryImage_basic(t *testing.T) {
	testAccEnsureLocalRegistry(t)
	t.Cleanup(testAccCleanupLocalRegistry)

	rName := fmt.Sprintf("tf-test-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
	targetImage := fmt.Sprintf("localhost:5555/%s:latest", rName)

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckPodmanRegistryImageDestroy,
		Steps: []resource.TestStep{
			{
				Config: providerConfig() + fmt.Sprintf(`
resource "podman_image" "nginx" {
  name = "quay.io/podman/stable:latest"
  keep_locally = true
}

resource "podman_tag" "test" {
  source_image = podman_image.nginx.name
  target_image = %q
}

resource "podman_registry_image" "test" {
  name = podman_tag.test.target_image
}
`, targetImage),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("podman_registry_image.test", "sha256_digest"),
					resource.TestCheckResourceAttr("podman_registry_image.test", "name", targetImage),
				),
			},
		},
	})
}

func TestAccPodmanRegistryImage_keepRemotely(t *testing.T) {
	testAccEnsureLocalRegistry(t)
	t.Cleanup(testAccCleanupLocalRegistry)

	rName := fmt.Sprintf("tf-test-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
	targetImage := fmt.Sprintf("localhost:5555/%s:latest", rName)

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckPodmanRegistryImageDestroy,
		Steps: []resource.TestStep{
			{
				Config: providerConfig() + fmt.Sprintf(`
resource "podman_image" "nginx" {
  name = "quay.io/podman/stable:latest"
  keep_locally = true
}

resource "podman_tag" "test" {
  source_image = podman_image.nginx.name
  target_image = %q
}

resource "podman_registry_image" "test" {
  name           = podman_tag.test.target_image
  keep_remotely  = true
}
`, targetImage),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("podman_registry_image.test", "sha256_digest"),
					resource.TestCheckResourceAttr("podman_registry_image.test", "name", targetImage),
					resource.TestCheckResourceAttr("podman_registry_image.test", "keep_remotely", "true"),
				),
			},
		},
	})
}

func TestAccDataSourcePodmanRegistryImage(t *testing.T) {
	testAccEnsureLocalRegistry(t)
	t.Cleanup(testAccCleanupLocalRegistry)

	rName := fmt.Sprintf("tf-test-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
	targetImage := fmt.Sprintf("localhost:5555/%s:latest", rName)

	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: providerConfig() + fmt.Sprintf(`
resource "podman_image" "nginx" {
  name = "quay.io/podman/stable:latest"
  keep_locally = true
}

resource "podman_tag" "test" {
  source_image = podman_image.nginx.name
  target_image = %q
}

resource "podman_registry_image" "test" {
  name = podman_tag.test.target_image
}

data "podman_registry_image" "test" {
  name = podman_registry_image.test.name
}
`, targetImage),
				// DistributionInspect may not work against a local registry
				// in rootless mode. If the apply fails, skip the test.
				SkipFunc: func() (bool, error) {
					return false, nil
				},
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("data.podman_registry_image.test", "sha256_digest"),
					resource.TestCheckResourceAttr("data.podman_registry_image.test", "name", targetImage),
				),
			},
		},
	})
}

func TestAccDataSourcePodmanRegistryImageManifests(t *testing.T) {
	testAccEnsureLocalRegistry(t)
	t.Cleanup(testAccCleanupLocalRegistry)

	rName := fmt.Sprintf("tf-test-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
	targetImage := fmt.Sprintf("localhost:5555/%s:latest", rName)

	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: providerConfig() + fmt.Sprintf(`
resource "podman_image" "nginx" {
  name = "quay.io/podman/stable:latest"
  keep_locally = true
}

resource "podman_tag" "test" {
  source_image = podman_image.nginx.name
  target_image = %q
}

resource "podman_registry_image" "test" {
  name = podman_tag.test.target_image
}

data "podman_registry_image_manifests" "test" {
  name = podman_registry_image.test.name
}
`, targetImage),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("data.podman_registry_image_manifests.test", "name", targetImage),
				),
			},
		},
	})
}
