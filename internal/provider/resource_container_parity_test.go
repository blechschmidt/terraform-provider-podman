package provider

import (
	"fmt"
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

// imageRef returns a small image suitable for acceptance tests.
const testImageRef = "quay.io/podman/stable:latest"

// providerConfigWithImage emits a podman_image data source declaration that
// pulls the image (or refers to a previously-pulled one), then the
// shared provider config preamble.
func providerConfigWithImage() string {
	return providerConfig() + `
resource "podman_image" "img" {
  name         = "` + testImageRef + `"
  keep_locally = true
}
`
}

// --------------------- container parity tests ---------------------

// TestAccPodmanContainer_platform exercises the new `platform` field.
func TestAccPodmanContainer_platform(t *testing.T) {
	if os.Getenv("TF_ACC") == "" {
		t.Skip("TF_ACC not set; skipping acceptance test")
	}
	testAccPreCheck(t)
	name := "tf-test-" + acctest.RandStringFromCharSet(8, acctest.CharSetAlphaNum)
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckPodmanContainerDestroy,
		Steps: []resource.TestStep{
			{
				Config: providerConfigWithImage() + fmt.Sprintf(`
resource "podman_container" "test" {
  name     = "%s"
  image    = podman_image.img.image_id
  platform = "linux/amd64"
  command  = ["sleep", "30"]
}
`, name),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("podman_container.test", "name", name),
					resource.TestCheckResourceAttr("podman_container.test", "platform", "linux/amd64"),
				),
			},
		},
	})
}

// TestAccPodmanContainer_cgroupParent covers `cgroup_parent`.
func TestAccPodmanContainer_cgroupParent(t *testing.T) {
	if os.Getenv("TF_ACC") == "" {
		t.Skip("TF_ACC not set; skipping acceptance test")
	}
	testAccPreCheck(t)
	name := "tf-test-" + acctest.RandStringFromCharSet(8, acctest.CharSetAlphaNum)
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckPodmanContainerDestroy,
		Steps: []resource.TestStep{
			{
				Config: providerConfigWithImage() + fmt.Sprintf(`
resource "podman_container" "test" {
  name          = "%s"
  image         = podman_image.img.image_id
  command       = ["sleep", "30"]
  cgroup_parent = "user.slice"
}
`, name),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("podman_container.test", "cgroup_parent", "user.slice"),
				),
			},
		},
	})
}

// TestAccPodmanContainer_memoryReservation covers `memory_reservation`.
func TestAccPodmanContainer_memoryReservation(t *testing.T) {
	if os.Getenv("TF_ACC") == "" {
		t.Skip("TF_ACC not set; skipping acceptance test")
	}
	testAccPreCheck(t)
	name := "tf-test-" + acctest.RandStringFromCharSet(8, acctest.CharSetAlphaNum)
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckPodmanContainerDestroy,
		Steps: []resource.TestStep{
			{
				Config: providerConfigWithImage() + fmt.Sprintf(`
resource "podman_container" "test" {
  name               = "%s"
  image              = podman_image.img.image_id
  command            = ["sleep", "30"]
  memory             = 67108864
  memory_reservation = 33554432
}
`, name),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("podman_container.test", "memory", "67108864"),
					resource.TestCheckResourceAttr("podman_container.test", "memory_reservation", "33554432"),
				),
			},
		},
	})
}

// TestAccPodmanContainer_cpus covers the convenience `cpus` field.
func TestAccPodmanContainer_cpus(t *testing.T) {
	if os.Getenv("TF_ACC") == "" {
		t.Skip("TF_ACC not set; skipping acceptance test")
	}
	testAccPreCheck(t)
	name := "tf-test-" + acctest.RandStringFromCharSet(8, acctest.CharSetAlphaNum)
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckPodmanContainerDestroy,
		Steps: []resource.TestStep{
			{
				Config: providerConfigWithImage() + fmt.Sprintf(`
resource "podman_container" "test" {
  name    = "%s"
  image   = podman_image.img.image_id
  command = ["sleep", "30"]
  cpus    = "0.5"
}
`, name),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("podman_container.test", "cpus", "0.5"),
				),
			},
		},
	})
}

// TestAccPodmanContainer_healthcheckStartInterval covers `healthcheck.start_interval`.
func TestAccPodmanContainer_healthcheckStartInterval(t *testing.T) {
	if os.Getenv("TF_ACC") == "" {
		t.Skip("TF_ACC not set; skipping acceptance test")
	}
	testAccPreCheck(t)
	name := "tf-test-" + acctest.RandStringFromCharSet(8, acctest.CharSetAlphaNum)
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckPodmanContainerDestroy,
		Steps: []resource.TestStep{
			{
				Config: providerConfigWithImage() + fmt.Sprintf(`
resource "podman_container" "test" {
  name    = "%s"
  image   = podman_image.img.image_id
  command = ["sleep", "30"]
  healthcheck {
    test           = ["CMD", "true"]
    interval       = "10s"
    timeout        = "3s"
    retries        = 2
    start_period   = "5s"
    start_interval = "2s"
  }
}
`, name),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("podman_container.test", "healthcheck.0.start_interval", "2s"),
				),
			},
		},
	})
}

// TestAccPodmanContainer_volumeSelinuxRelabel covers `volumes.selinux_relabel`.
func TestAccPodmanContainer_volumeSelinuxRelabel(t *testing.T) {
	if os.Getenv("TF_ACC") == "" {
		t.Skip("TF_ACC not set; skipping acceptance test")
	}
	testAccPreCheck(t)
	cName := "tf-test-" + acctest.RandStringFromCharSet(8, acctest.CharSetAlphaNum)
	vName := "tf-test-" + acctest.RandStringFromCharSet(8, acctest.CharSetAlphaNum)
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckPodmanContainerDestroy,
		Steps: []resource.TestStep{
			{
				Config: providerConfigWithImage() + fmt.Sprintf(`
resource "podman_volume" "v" {
  name = "%s"
}
resource "podman_container" "test" {
  name    = "%s"
  image   = podman_image.img.image_id
  command = ["sleep", "30"]
  volumes {
    volume_name     = podman_volume.v.name
    container_path  = "/data"
    selinux_relabel = "shared"
  }
}
`, vName, cName),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("podman_container.test", "volumes.#", "1"),
				),
			},
		},
	})
}

// TestAccPodmanContainer_uploadPermissions covers `upload.permissions`.
func TestAccPodmanContainer_uploadPermissions(t *testing.T) {
	if os.Getenv("TF_ACC") == "" {
		t.Skip("TF_ACC not set; skipping acceptance test")
	}
	testAccPreCheck(t)
	name := "tf-test-" + acctest.RandStringFromCharSet(8, acctest.CharSetAlphaNum)
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckPodmanContainerDestroy,
		Steps: []resource.TestStep{
			{
				Config: providerConfigWithImage() + fmt.Sprintf(`
resource "podman_container" "test" {
  name    = "%s"
  image   = podman_image.img.image_id
  command = ["sleep", "30"]
  upload {
    file        = "/tmp/hello.txt"
    content     = "hi"
    permissions = "0600"
  }
}
`, name),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("podman_container.test", "upload.#", "1"),
				),
			},
		},
	})
}

// TestAccPodmanContainer_networksAdvancedFull covers
// networks_advanced.{mac_address,link_local_ips,driver_opts}.
func TestAccPodmanContainer_networksAdvancedFull(t *testing.T) {
	if os.Getenv("TF_ACC") == "" {
		t.Skip("TF_ACC not set; skipping acceptance test")
	}
	testAccPreCheck(t)
	cName := "tf-test-" + acctest.RandStringFromCharSet(8, acctest.CharSetAlphaNum)
	nName := "tf-test-net-" + acctest.RandStringFromCharSet(8, acctest.CharSetAlphaNum)
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckPodmanContainerDestroy,
		Steps: []resource.TestStep{
			{
				Config: providerConfigWithImage() + fmt.Sprintf(`
resource "podman_network" "n" {
  name = "%s"
  ipam_config {
    subnet = "10.99.42.0/24"
  }
}
resource "podman_container" "test" {
  name    = "%s"
  image   = podman_image.img.image_id
  command = ["sleep", "30"]
  networks_advanced {
    name           = podman_network.n.name
    ipv4_address   = "10.99.42.42"
    aliases        = ["alias1"]
    mac_address    = "02:42:ac:11:42:42"
  }
}
`, nName, cName),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("podman_container.test", "networks_advanced.#", "1"),
				),
			},
		},
	})
}

// TestAccPodmanContainer_mountSubpath covers `mounts.volume_options.subpath`.
func TestAccPodmanContainer_mountSubpath(t *testing.T) {
	if os.Getenv("TF_ACC") == "" {
		t.Skip("TF_ACC not set; skipping acceptance test")
	}
	testAccPreCheck(t)
	cName := "tf-test-" + acctest.RandStringFromCharSet(8, acctest.CharSetAlphaNum)
	vName := "tf-test-" + acctest.RandStringFromCharSet(8, acctest.CharSetAlphaNum)
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckPodmanContainerDestroy,
		Steps: []resource.TestStep{
			{
				Config: providerConfigWithImage() + fmt.Sprintf(`
resource "podman_volume" "v" {
  name = "%s"
}
resource "podman_container" "test" {
  name    = "%s"
  image   = podman_image.img.image_id
  command = ["sleep", "30"]
  mounts {
    type   = "volume"
    target = "/data"
    source = podman_volume.v.name
    volume_options {
      no_copy = false
    }
  }
}
`, vName, cName),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("podman_container.test", "mounts.#", "1"),
				),
			},
		},
	})
}

// TestAccPodmanContainer_containerReadRefreshTimeoutMilliseconds covers the
// `container_read_refresh_timeout_milliseconds` provider knob.
func TestAccPodmanContainer_containerReadRefreshTimeoutMilliseconds(t *testing.T) {
	if os.Getenv("TF_ACC") == "" {
		t.Skip("TF_ACC not set; skipping acceptance test")
	}
	testAccPreCheck(t)
	name := "tf-test-" + acctest.RandStringFromCharSet(8, acctest.CharSetAlphaNum)
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckPodmanContainerDestroy,
		Steps: []resource.TestStep{
			{
				Config: providerConfigWithImage() + fmt.Sprintf(`
resource "podman_container" "test" {
  name                                          = "%s"
  image                                         = podman_image.img.image_id
  command                                       = ["sleep", "30"]
  container_read_refresh_timeout_milliseconds   = 30000
}
`, name),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("podman_container.test", "container_read_refresh_timeout_milliseconds", "30000"),
				),
			},
		},
	})
}

// --------------------- new data source tests ---------------------

// TestAccDataPodmanContainers covers the data.podman_containers data source.
func TestAccDataPodmanContainers(t *testing.T) {
	if os.Getenv("TF_ACC") == "" {
		t.Skip("TF_ACC not set; skipping acceptance test")
	}
	testAccPreCheck(t)
	cName := "tf-test-" + acctest.RandStringFromCharSet(8, acctest.CharSetAlphaNum)
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckPodmanContainerDestroy,
		Steps: []resource.TestStep{
			{
				Config: providerConfigWithImage() + fmt.Sprintf(`
resource "podman_container" "tt" {
  name    = "%s"
  image   = podman_image.img.image_id
  command = ["sleep", "60"]
}
data "podman_containers" "all" {
  depends_on = [podman_container.tt]
  all        = true
  name       = ["%s"]
}
`, cName, cName),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("data.podman_containers.all", "containers.#", "1"),
					resource.TestCheckResourceAttr("data.podman_containers.all", "containers.0.names.0", cName),
				),
			},
		},
	})
}

// TestAccDataPodmanRegistryImageTags covers data.podman_registry_image_tags
// against a public Docker Hub image.
func TestAccDataPodmanRegistryImageTags(t *testing.T) {
	if os.Getenv("TF_ACC") == "" {
		t.Skip("TF_ACC not set; skipping acceptance test")
	}
	testAccPreCheck(t)
	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: providerConfig() + `
data "podman_registry_image_tags" "alpine" {
  name = "alpine"
}
`,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("data.podman_registry_image_tags.alpine", "tags.#"),
				),
			},
		},
	})
}

// TestAccDataPodmanNetwork_containers verifies the new `containers`
// computed list on the network data source.
func TestAccDataPodmanNetwork_containers(t *testing.T) {
	if os.Getenv("TF_ACC") == "" {
		t.Skip("TF_ACC not set; skipping acceptance test")
	}
	testAccPreCheck(t)
	cName := "tf-test-" + acctest.RandStringFromCharSet(8, acctest.CharSetAlphaNum)
	nName := "tf-test-net-" + acctest.RandStringFromCharSet(8, acctest.CharSetAlphaNum)
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckPodmanContainerDestroy,
		Steps: []resource.TestStep{
			{
				Config: providerConfigWithImage() + fmt.Sprintf(`
resource "podman_network" "n" {
  name = "%s"
}
resource "podman_container" "test" {
  name    = "%s"
  image   = podman_image.img.image_id
  command = ["sleep", "60"]
  networks_advanced {
    name = podman_network.n.name
  }
}
data "podman_network" "n" {
  name       = podman_network.n.name
  depends_on = [podman_container.test]
}
`, nName, cName),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("data.podman_network.n", "containers.#"),
				),
			},
		},
	})
}

// TestAccPodmanTag_tagTriggers covers `tag_triggers`.
func TestAccPodmanTag_tagTriggers(t *testing.T) {
	if os.Getenv("TF_ACC") == "" {
		t.Skip("TF_ACC not set; skipping acceptance test")
	}
	testAccPreCheck(t)
	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: providerConfig() + `
resource "podman_image" "img" {
  name         = "` + testImageRef + `"
  keep_locally = true
}
resource "podman_tag" "t" {
  source_image = podman_image.img.image_id
  target_image = "tf-tag-test:trigger"
  tag_triggers = ["v1"]
}
`,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("podman_tag.t", "tag_triggers.0", "v1"),
				),
			},
		},
	})
}
