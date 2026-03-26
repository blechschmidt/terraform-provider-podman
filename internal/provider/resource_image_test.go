package provider

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/client"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

func TestAccPodmanImage_pull(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		CheckDestroy: testAccCheckPodmanImageDestroy,
		Steps: []resource.TestStep{
			{
				Config: providerConfig() + `
resource "podman_image" "test" {
  name = "quay.io/podman/stable:latest"
}
`,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("podman_image.test", "image_id"),
					resource.TestCheckResourceAttrSet("podman_image.test", "repo_digest"),
				),
			},
		},
	})
}

func TestAccPodmanImage_pullWithPlatform(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		CheckDestroy: testAccCheckPodmanImageDestroy,
		Steps: []resource.TestStep{
			{
				Config: providerConfig() + `
resource "podman_image" "test" {
  name     = "quay.io/podman/stable:latest"
  platform = "linux/amd64"
}
`,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("podman_image.test", "image_id"),
					resource.TestCheckResourceAttr("podman_image.test", "platform", "linux/amd64"),
				),
			},
		},
	})
}

func TestAccPodmanImage_keepLocally(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		CheckDestroy: func(s *terraform.State) error {
			cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
			if err != nil {
				return err
			}
			defer cli.Close()

			// With keep_locally = true, the image should still exist after destroy.
			_, _, err = cli.ImageInspectWithRaw(context.Background(), "quay.io/podman/stable:latest")
			if err != nil {
				return fmt.Errorf("expected image nginx:latest to still exist after destroy with keep_locally=true, but got: %s", err)
			}

			// Clean up: remove the image after the test.
			_, _ = cli.ImageRemove(context.Background(), "quay.io/podman/stable:latest", image.RemoveOptions{Force: true})
			return nil
		},
		Steps: []resource.TestStep{
			{
				Config: providerConfig() + `
resource "podman_image" "test" {
  name         = "quay.io/podman/stable:latest"
  keep_locally = true
}
`,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("podman_image.test", "image_id"),
				),
			},
		},
	})
}

func TestAccPodmanImage_forceRemove(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		CheckDestroy: testAccCheckPodmanImageDestroy,
		Steps: []resource.TestStep{
			{
				Config: providerConfig() + `
resource "podman_image" "test" {
  name         = "quay.io/podman/stable:latest"
  force_remove = true
}
`,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("podman_image.test", "image_id"),
					resource.TestCheckResourceAttr("podman_image.test", "force_remove", "true"),
				),
			},
		},
	})
}

func TestAccPodmanImage_build(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "tf-test-build-")
	if err != nil {
		t.Fatalf("failed to create temp dir: %s", err)
	}
	defer os.RemoveAll(tmpDir)

	dockerfileContent := "FROM quay.io/podman/stable:latest\nRUN echo hello > /hello.txt\n"
	err = os.WriteFile(filepath.Join(tmpDir, "Dockerfile"), []byte(dockerfileContent), 0644)
	if err != nil {
		t.Fatalf("failed to write Dockerfile: %s", err)
	}

	rName := fmt.Sprintf("tf-test-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
	imageName := fmt.Sprintf("localhost/%s:latest", rName)

	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		CheckDestroy: testAccCheckPodmanImageDestroy,
		Steps: []resource.TestStep{
			{
				Config: providerConfig() + fmt.Sprintf(`
resource "podman_image" "test" {
  name = %q

  build {
    context    = %q
    dockerfile = "Dockerfile"
  }
}
`, imageName, tmpDir),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("podman_image.test", "image_id"),
				),
			},
		},
	})
}

func testAccCheckPodmanImageDestroy(s *terraform.State) error {
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return err
	}
	defer cli.Close()

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "podman_image" {
			continue
		}

		_, _, err := cli.ImageInspectWithRaw(context.Background(), rs.Primary.Attributes["name"])
		if err == nil {
			return fmt.Errorf("image %s still exists", rs.Primary.Attributes["name"])
		}
	}

	return nil
}
