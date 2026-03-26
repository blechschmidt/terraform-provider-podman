package provider

import (
	"fmt"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestAccDataSourcePodmanImage(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: providerConfig() + `
resource "podman_image" "test" {
  name = "quay.io/podman/stable:latest"
  keep_locally = true
}

data "podman_image" "test" {
  name = podman_image.test.name
}
`,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("data.podman_image.test", "repo_digest"),
					resource.TestMatchResourceAttr("data.podman_image.test", "repo_digest", regexp.MustCompile(`.+`)),
				),
			},
		},
	})
}

func TestAccDataSourcePodmanNetwork(t *testing.T) {
	networkName := fmt.Sprintf("tf-test-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))

	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: providerConfig() + fmt.Sprintf(`
resource "podman_network" "test" {
  name   = "%s"
  driver = "bridge"

  ipam_config {
    subnet  = "172.29.0.0/16"
    gateway = "172.29.0.1"
  }
}

data "podman_network" "test" {
  name = podman_network.test.name
}
`, networkName),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("data.podman_network.test", "driver", "bridge"),
					resource.TestCheckResourceAttr("data.podman_network.test", "ipam_config.0.subnet", "172.29.0.0/16"),
				),
			},
		},
	})
}

func TestAccDataSourcePodmanLogs_basic(t *testing.T) {
	containerName := fmt.Sprintf("tf-test-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))

	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: providerConfig() + fmt.Sprintf(`
resource "podman_image" "test" {
  name = "quay.io/podman/stable:latest"
  keep_locally = true
}

resource "podman_container" "test" {
  name     = "%s"
  image    = podman_image.test.image_id
  command  = ["sh", "-c", "echo hello_from_test"]
  must_run = false
  rm       = false
  start    = true
  logs     = true
}

data "podman_logs" "test" {
  name                     = podman_container.test.name
  show_stdout              = true
  logs_list_string_enabled = true
  tail                     = "10"
  depends_on               = [podman_container.test]
}
`, containerName),
				Check: resource.ComposeTestCheckFunc(
					resource.TestMatchResourceAttr("data.podman_logs.test", "logs_list_string.#", regexp.MustCompile(`[1-9][0-9]*`)),
				),
			},
		},
	})
}

func TestAccDataSourcePodmanLogs_withTimestamps(t *testing.T) {
	containerName := fmt.Sprintf("tf-test-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))

	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: providerConfig() + fmt.Sprintf(`
resource "podman_image" "test" {
  name = "quay.io/podman/stable:latest"
  keep_locally = true
}

resource "podman_container" "test" {
  name     = "%s"
  image    = podman_image.test.image_id
  command  = ["sh", "-c", "echo hello_from_test"]
  must_run = false
  rm       = false
  start    = true
  logs     = true
}

data "podman_logs" "test" {
  name                     = podman_container.test.name
  show_stdout              = true
  logs_list_string_enabled = true
  timestamps               = true
  tail                     = "10"
  depends_on               = [podman_container.test]
}
`, containerName),
				Check: resource.ComposeTestCheckFunc(
					resource.TestMatchResourceAttr("data.podman_logs.test", "logs_list_string.#", regexp.MustCompile(`[1-9][0-9]*`)),
					resource.TestMatchResourceAttr("data.podman_logs.test", "logs_list_string.0", regexp.MustCompile(`\d{4}-\d{2}-\d{2}`)),
				),
			},
		},
	})
}

func TestAccDataSourcePodmanLogs_discardHeaders(t *testing.T) {
	containerName := fmt.Sprintf("tf-test-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))

	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: providerConfig() + fmt.Sprintf(`
resource "podman_image" "test" {
  name = "quay.io/podman/stable:latest"
  keep_locally = true
}

resource "podman_container" "test" {
  name     = "%s"
  image    = podman_image.test.image_id
  command  = ["sh", "-c", "echo hello_from_test"]
  must_run = false
  rm       = false
  start    = true
  logs     = true
}

data "podman_logs" "test" {
  name                     = podman_container.test.name
  show_stdout              = true
  logs_list_string_enabled = true
  discard_headers          = true
  tail                     = "10"
  depends_on               = [podman_container.test]
}
`, containerName),
				Check: resource.ComposeTestCheckFunc(
					resource.TestMatchResourceAttr("data.podman_logs.test", "logs_list_string.#", regexp.MustCompile(`[1-9][0-9]*`)),
				),
			},
		},
	})
}
