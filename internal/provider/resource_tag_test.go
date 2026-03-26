package provider

import (
	"context"
	"fmt"
	"testing"

	"github.com/docker/docker/client"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

func TestAccPodmanTag_basic(t *testing.T) {
	rName := fmt.Sprintf("tf-test-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
	targetImage := fmt.Sprintf("localhost/%s:v1", rName)

	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		CheckDestroy: testAccCheckPodmanTagDestroy,
		Steps: []resource.TestStep{
			{
				Config: providerConfig() + fmt.Sprintf(`
resource "podman_image" "test" {
  name = "quay.io/podman/stable:latest"
}

resource "podman_tag" "test" {
  source_image = podman_image.test.name
  target_image = %q
}
`, targetImage),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("podman_tag.test", "source_image_id"),
					resource.TestCheckResourceAttr("podman_tag.test", "target_image", targetImage),
				),
			},
		},
	})
}

func TestAccPodmanTag_noTag(t *testing.T) {
	rName := fmt.Sprintf("tf-test-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
	targetImage := fmt.Sprintf("localhost/%s", rName)

	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		CheckDestroy: testAccCheckPodmanTagDestroy,
		Steps: []resource.TestStep{
			{
				Config: providerConfig() + fmt.Sprintf(`
resource "podman_image" "test" {
  name = "quay.io/podman/stable:latest"
}

resource "podman_tag" "test" {
  source_image = podman_image.test.name
  target_image = %q
}
`, targetImage),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("podman_tag.test", "source_image_id"),
					resource.TestCheckResourceAttr("podman_tag.test", "target_image", targetImage),
				),
			},
		},
	})
}

func TestAccPodmanTag_update(t *testing.T) {
	rName := fmt.Sprintf("tf-test-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
	targetImageV1 := fmt.Sprintf("localhost/%s:v1", rName)
	targetImageV2 := fmt.Sprintf("localhost/%s:v2", rName)

	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		CheckDestroy: testAccCheckPodmanTagDestroy,
		Steps: []resource.TestStep{
			{
				Config: providerConfig() + fmt.Sprintf(`
resource "podman_image" "test" {
  name = "quay.io/podman/stable:latest"
}

resource "podman_tag" "test" {
  source_image = podman_image.test.name
  target_image = %q
}
`, targetImageV1),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("podman_tag.test", "target_image", targetImageV1),
				),
			},
			{
				Config: providerConfig() + fmt.Sprintf(`
resource "podman_image" "test" {
  name = "quay.io/podman/stable:latest"
}

resource "podman_tag" "test" {
  source_image = podman_image.test.name
  target_image = %q
}
`, targetImageV2),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("podman_tag.test", "target_image", targetImageV2),
				),
			},
		},
	})
}

func testAccCheckPodmanTagDestroy(s *terraform.State) error {
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return err
	}
	defer cli.Close()

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "podman_tag" {
			continue
		}

		targetImage := rs.Primary.Attributes["target_image"]
		_, _, err := cli.ImageInspectWithRaw(context.Background(), targetImage)
		if err == nil {
			return fmt.Errorf("tag %s still exists", targetImage)
		}
	}

	return nil
}
