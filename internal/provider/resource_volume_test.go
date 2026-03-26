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

func TestAccPodmanVolume_basic(t *testing.T) {
	rName := fmt.Sprintf("tf-test-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))

	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		CheckDestroy: testAccCheckPodmanVolumeDestroy,
		Steps: []resource.TestStep{
			{
				Config: providerConfig() + fmt.Sprintf(`
resource "podman_volume" "test" {
  name = %q
}
`, rName),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("podman_volume.test", "name", rName),
					resource.TestCheckResourceAttrSet("podman_volume.test", "mountpoint"),
				),
			},
		},
	})
}

func TestAccPodmanVolume_withLabels(t *testing.T) {
	rName := fmt.Sprintf("tf-test-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))

	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		CheckDestroy: testAccCheckPodmanVolumeDestroy,
		Steps: []resource.TestStep{
			{
				Config: providerConfig() + fmt.Sprintf(`
resource "podman_volume" "test" {
  name = %q

  labels {
    label = "env"
    value = "test"
  }

  labels {
    label = "project"
    value = "terraform"
  }
}
`, rName),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("podman_volume.test", "name", rName),
					resource.TestCheckResourceAttr("podman_volume.test", "labels.#", "2"),
				),
			},
		},
	})
}

func TestAccPodmanVolume_autoName(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		CheckDestroy: testAccCheckPodmanVolumeDestroy,
		Steps: []resource.TestStep{
			{
				Config: providerConfig() + `
resource "podman_volume" "test" {
}
`,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("podman_volume.test", "name"),
					resource.TestCheckResourceAttrSet("podman_volume.test", "mountpoint"),
				),
			},
		},
	})
}

func testAccCheckPodmanVolumeDestroy(s *terraform.State) error {
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return err
	}
	defer cli.Close()

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "podman_volume" {
			continue
		}

		_, err := cli.VolumeInspect(context.Background(), rs.Primary.ID)
		if err == nil {
			return fmt.Errorf("volume %s still exists", rs.Primary.ID)
		}
	}

	return nil
}
