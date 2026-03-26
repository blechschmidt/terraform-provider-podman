package provider

import (
	"context"
	"fmt"
	"testing"

	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/client"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

func TestAccPodmanNetwork_basic(t *testing.T) {
	rName := fmt.Sprintf("tf-test-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))

	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		CheckDestroy: testAccCheckPodmanNetworkDestroy,
		Steps: []resource.TestStep{
			{
				Config: providerConfig() + fmt.Sprintf(`
resource "podman_network" "test" {
  name = %q
}
`, rName),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("podman_network.test", "name", rName),
					resource.TestCheckResourceAttr("podman_network.test", "driver", "bridge"),
				),
			},
		},
	})
}

func TestAccPodmanNetwork_withIpam(t *testing.T) {
	rName := fmt.Sprintf("tf-test-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))

	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		CheckDestroy: testAccCheckPodmanNetworkDestroy,
		Steps: []resource.TestStep{
			{
				Config: providerConfig() + fmt.Sprintf(`
resource "podman_network" "test" {
  name = %q

  ipam_config {
    subnet  = "172.28.0.0/16"
    gateway = "172.28.0.1"
  }
}
`, rName),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("podman_network.test", "name", rName),
					resource.TestCheckResourceAttr("podman_network.test", "ipam_config.0.subnet", "172.28.0.0/16"),
					resource.TestCheckResourceAttr("podman_network.test", "ipam_config.0.gateway", "172.28.0.1"),
				),
			},
		},
	})
}

func TestAccPodmanNetwork_internal(t *testing.T) {
	rName := fmt.Sprintf("tf-test-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))

	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		CheckDestroy: testAccCheckPodmanNetworkDestroy,
		Steps: []resource.TestStep{
			{
				Config: providerConfig() + fmt.Sprintf(`
resource "podman_network" "test" {
  name     = %q
  internal = true
}
`, rName),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("podman_network.test", "name", rName),
					resource.TestCheckResourceAttr("podman_network.test", "internal", "true"),
				),
			},
		},
	})
}

func TestAccPodmanNetwork_withLabels(t *testing.T) {
	rName := fmt.Sprintf("tf-test-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))

	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		CheckDestroy: testAccCheckPodmanNetworkDestroy,
		Steps: []resource.TestStep{
			{
				Config: providerConfig() + fmt.Sprintf(`
resource "podman_network" "test" {
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
					resource.TestCheckResourceAttr("podman_network.test", "name", rName),
					resource.TestCheckResourceAttr("podman_network.test", "labels.#", "2"),
				),
			},
		},
	})
}

func TestAccPodmanNetwork_ipv6(t *testing.T) {
	rName := fmt.Sprintf("tf-test-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))

	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		CheckDestroy: testAccCheckPodmanNetworkDestroy,
		Steps: []resource.TestStep{
			{
				Config: providerConfig() + fmt.Sprintf(`
resource "podman_network" "test" {
  name = %q
  ipv6 = true
}
`, rName),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("podman_network.test", "name", rName),
					resource.TestCheckResourceAttr("podman_network.test", "ipv6", "true"),
				),
			},
		},
	})
}

func testAccCheckPodmanNetworkDestroy(s *terraform.State) error {
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return err
	}
	defer cli.Close()

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "podman_network" {
			continue
		}

		_, err := cli.NetworkInspect(context.Background(), rs.Primary.ID, network.InspectOptions{})
		if err == nil {
			return fmt.Errorf("network %s still exists", rs.Primary.ID)
		}
	}

	return nil
}
