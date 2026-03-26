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

func testAccCheckPodmanContainerDestroy(s *terraform.State) error {
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return err
	}
	defer cli.Close()

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "podman_container" {
			continue
		}

		_, err := cli.ContainerInspect(context.Background(), rs.Primary.ID)
		if err == nil {
			return fmt.Errorf("container %s still exists", rs.Primary.ID)
		}
	}

	return nil
}

// TestAccPodmanContainer_basic creates a basic alpine container and verifies name and image.
func TestAccPodmanContainer_basic(t *testing.T) {
	name := "tf-test-" + acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckPodmanContainerDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccPodmanContainerConfig_basic(name),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("podman_container.test", "name", name),
					resource.TestCheckResourceAttrSet("podman_container.test", "image"),
				),
			},
		},
	})
}

func testAccPodmanContainerConfig_basic(name string) string {
	return providerConfig() + fmt.Sprintf(`
resource "podman_image" "test" {
  name = "quay.io/podman/stable:latest"
  keep_locally = true
}

resource "podman_container" "test" {
  name    = "%s"
  image   = podman_image.test.image_id
  command = ["sleep", "300"]
}
`, name)
}

// TestAccPodmanContainer_withPorts creates an nginx container with port mappings.
func TestAccPodmanContainer_withPorts(t *testing.T) {
	name := "tf-test-" + acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckPodmanContainerDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccPodmanContainerConfig_withPorts(name),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("podman_container.test", "name", name),
					resource.TestCheckResourceAttr("podman_container.test", "ports.0.internal", "80"),
					resource.TestCheckResourceAttr("podman_container.test", "ports.0.external", "8888"),
				),
			},
		},
	})
}

func testAccPodmanContainerConfig_withPorts(name string) string {
	return providerConfig() + fmt.Sprintf(`
resource "podman_image" "test" {
  name = "quay.io/podman/stable:latest"
  keep_locally = true
}

resource "podman_container" "test" {
  name  = "%s"
  image = podman_image.test.image_id

  ports {
    internal = 80
    external = 8888
  }
}
`, name)
}

// TestAccPodmanContainer_withVolume creates a volume and binds it to a container.
func TestAccPodmanContainer_withVolume(t *testing.T) {
	containerName := "tf-test-" + acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)
	volumeName := "tf-test-" + acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckPodmanContainerDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccPodmanContainerConfig_withVolume(containerName, volumeName),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("podman_container.test", "name", containerName),
					resource.TestCheckResourceAttr("podman_volume.test", "name", volumeName),
					resource.TestCheckTypeSetElemNestedAttrs("podman_container.test", "volumes.*", map[string]string{
						"volume_name":    volumeName,
						"container_path": "/data",
					}),
				),
			},
		},
	})
}

func testAccPodmanContainerConfig_withVolume(containerName, volumeName string) string {
	return providerConfig() + fmt.Sprintf(`
resource "podman_image" "test" {
  name = "quay.io/podman/stable:latest"
  keep_locally = true
}

resource "podman_volume" "test" {
  name = "%s"
}

resource "podman_container" "test" {
  name    = "%s"
  image   = podman_image.test.image_id
  command = ["sleep", "300"]

  volumes {
    volume_name    = podman_volume.test.name
    container_path = "/data"
  }
}
`, volumeName, containerName)
}

// TestAccPodmanContainer_withEnv creates a container with environment variables.
func TestAccPodmanContainer_withEnv(t *testing.T) {
	name := "tf-test-" + acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckPodmanContainerDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccPodmanContainerConfig_withEnv(name),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("podman_container.test", "name", name),
					resource.TestCheckTypeSetElemAttr("podman_container.test", "env.*", "FOO=bar"),
					resource.TestCheckTypeSetElemAttr("podman_container.test", "env.*", "BAZ=qux"),
				),
			},
		},
	})
}

func testAccPodmanContainerConfig_withEnv(name string) string {
	return providerConfig() + fmt.Sprintf(`
resource "podman_image" "test" {
  name = "quay.io/podman/stable:latest"
  keep_locally = true
}

resource "podman_container" "test" {
  name    = "%s"
  image   = podman_image.test.image_id
  command = ["sleep", "300"]
  env     = ["FOO=bar", "BAZ=qux"]
}
`, name)
}

// TestAccPodmanContainer_withLabels creates a container with labels.
func TestAccPodmanContainer_withLabels(t *testing.T) {
	name := "tf-test-" + acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckPodmanContainerDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccPodmanContainerConfig_withLabels(name),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("podman_container.test", "name", name),
					resource.TestCheckTypeSetElemNestedAttrs("podman_container.test", "labels.*", map[string]string{
						"label": "app",
						"value": "test",
					}),
				),
			},
		},
	})
}

func testAccPodmanContainerConfig_withLabels(name string) string {
	return providerConfig() + fmt.Sprintf(`
resource "podman_image" "test" {
  name = "quay.io/podman/stable:latest"
  keep_locally = true
}

resource "podman_container" "test" {
  name    = "%s"
  image   = podman_image.test.image_id
  command = ["sleep", "300"]

  labels {
    label = "app"
    value = "test"
  }
}
`, name)
}

// TestAccPodmanContainer_withHealthcheck creates a container with a healthcheck.
func TestAccPodmanContainer_withHealthcheck(t *testing.T) {
	name := "tf-test-" + acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckPodmanContainerDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccPodmanContainerConfig_withHealthcheck(name),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("podman_container.test", "name", name),
					resource.TestCheckResourceAttr("podman_container.test", "healthcheck.0.test.0", "CMD-SHELL"),
					resource.TestCheckResourceAttr("podman_container.test", "healthcheck.0.test.1", "true"),
					resource.TestCheckResourceAttr("podman_container.test", "healthcheck.0.interval", "5s"),
					resource.TestCheckResourceAttr("podman_container.test", "healthcheck.0.timeout", "3s"),
					resource.TestCheckResourceAttr("podman_container.test", "healthcheck.0.retries", "3"),
				),
			},
		},
	})
}

func testAccPodmanContainerConfig_withHealthcheck(name string) string {
	return providerConfig() + fmt.Sprintf(`
resource "podman_image" "test" {
  name = "quay.io/podman/stable:latest"
  keep_locally = true
}

resource "podman_container" "test" {
  name    = "%s"
  image   = podman_image.test.image_id
  command = ["sleep", "300"]

  healthcheck {
    test     = ["CMD-SHELL", "true"]
    interval = "5s"
    timeout  = "3s"
    retries  = 3
  }
}
`, name)
}

// TestAccPodmanContainer_withUpload creates a container that uploads a file.
func TestAccPodmanContainer_withUpload(t *testing.T) {
	name := "tf-test-" + acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckPodmanContainerDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccPodmanContainerConfig_withUpload(name),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("podman_container.test", "name", name),
					resource.TestCheckTypeSetElemNestedAttrs("podman_container.test", "upload.*", map[string]string{
						"file":    "/tmp/hello.txt",
						"content": "hello world",
					}),
				),
			},
		},
	})
}

func testAccPodmanContainerConfig_withUpload(name string) string {
	return providerConfig() + fmt.Sprintf(`
resource "podman_image" "test" {
  name = "quay.io/podman/stable:latest"
  keep_locally = true
}

resource "podman_container" "test" {
  name    = "%s"
  image   = podman_image.test.image_id
  command = ["sleep", "300"]

  upload {
    file    = "/tmp/hello.txt"
    content = "hello world"
  }
}
`, name)
}

// TestAccPodmanContainer_withNetworkAdvanced creates a network and a container with a static IP.
func TestAccPodmanContainer_withNetworkAdvanced(t *testing.T) {
	containerName := "tf-test-" + acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)
	networkName := "tf-test-" + acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckPodmanContainerDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccPodmanContainerConfig_withNetworkAdvanced(containerName, networkName),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("podman_container.test", "name", containerName),
					resource.TestCheckTypeSetElemNestedAttrs("podman_container.test", "networks_advanced.*", map[string]string{
						"name":         networkName,
						"ipv4_address": "172.20.0.10",
					}),
				),
			},
		},
	})
}

func testAccPodmanContainerConfig_withNetworkAdvanced(containerName, networkName string) string {
	return providerConfig() + fmt.Sprintf(`
resource "podman_image" "test" {
  name = "quay.io/podman/stable:latest"
  keep_locally = true
}

resource "podman_network" "test" {
  name   = "%s"
  driver = "bridge"

  ipam_config {
    subnet  = "172.20.0.0/16"
    gateway = "172.20.0.1"
  }
}

resource "podman_container" "test" {
  name    = "%s"
  image   = podman_image.test.image_id
  command = ["sleep", "300"]

  networks_advanced {
    name         = podman_network.test.name
    ipv4_address = "172.20.0.10"
  }
}
`, networkName, containerName)
}

// TestAccPodmanContainer_withCapabilities creates a container with added capabilities.
func TestAccPodmanContainer_withCapabilities(t *testing.T) {
	name := "tf-test-" + acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckPodmanContainerDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccPodmanContainerConfig_withCapabilities(name),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("podman_container.test", "name", name),
					resource.TestCheckResourceAttr("podman_container.test", "capabilities.0.add.#", "1"),
					resource.TestCheckTypeSetElemAttr("podman_container.test", "capabilities.0.add.*", "NET_ADMIN"),
				),
			},
		},
	})
}

func testAccPodmanContainerConfig_withCapabilities(name string) string {
	return providerConfig() + fmt.Sprintf(`
resource "podman_image" "test" {
  name = "quay.io/podman/stable:latest"
  keep_locally = true
}

resource "podman_container" "test" {
  name    = "%s"
  image   = podman_image.test.image_id
  command = ["sleep", "300"]

  capabilities {
    add = ["NET_ADMIN"]
  }
}
`, name)
}

// TestAccPodmanContainer_withDns creates a container with custom DNS servers.
func TestAccPodmanContainer_withDns(t *testing.T) {
	name := "tf-test-" + acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckPodmanContainerDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccPodmanContainerConfig_withDns(name),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("podman_container.test", "name", name),
					resource.TestCheckResourceAttr("podman_container.test", "dns.#", "1"),
					resource.TestCheckTypeSetElemAttr("podman_container.test", "dns.*", "8.8.8.8"),
				),
			},
		},
	})
}

func testAccPodmanContainerConfig_withDns(name string) string {
	return providerConfig() + fmt.Sprintf(`
resource "podman_image" "test" {
  name = "quay.io/podman/stable:latest"
  keep_locally = true
}

resource "podman_container" "test" {
  name    = "%s"
  image   = podman_image.test.image_id
  command = ["sleep", "300"]
  dns     = ["8.8.8.8"]
}
`, name)
}

// TestAccPodmanContainer_withHostEntry creates a container with a custom host entry.
func TestAccPodmanContainer_withHostEntry(t *testing.T) {
	name := "tf-test-" + acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckPodmanContainerDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccPodmanContainerConfig_withHostEntry(name),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("podman_container.test", "name", name),
					resource.TestCheckTypeSetElemNestedAttrs("podman_container.test", "host.*", map[string]string{
						"host": "myhost",
						"ip":   "10.0.0.1",
					}),
				),
			},
		},
	})
}

func testAccPodmanContainerConfig_withHostEntry(name string) string {
	return providerConfig() + fmt.Sprintf(`
resource "podman_image" "test" {
  name = "quay.io/podman/stable:latest"
  keep_locally = true
}

resource "podman_container" "test" {
  name    = "%s"
  image   = podman_image.test.image_id
  command = ["sleep", "300"]

  host {
    host = "myhost"
    ip   = "10.0.0.1"
  }
}
`, name)
}

// TestAccPodmanContainer_withUlimit creates a container with ulimit settings.
func TestAccPodmanContainer_withUlimit(t *testing.T) {
	name := "tf-test-" + acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckPodmanContainerDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccPodmanContainerConfig_withUlimit(name),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("podman_container.test", "name", name),
					resource.TestCheckTypeSetElemNestedAttrs("podman_container.test", "ulimit.*", map[string]string{
						"name": "nofile",
						"soft": "1024",
						"hard": "2048",
					}),
				),
			},
		},
	})
}

func testAccPodmanContainerConfig_withUlimit(name string) string {
	return providerConfig() + fmt.Sprintf(`
resource "podman_image" "test" {
  name = "quay.io/podman/stable:latest"
  keep_locally = true
}

resource "podman_container" "test" {
  name    = "%s"
  image   = podman_image.test.image_id
  command = ["sleep", "300"]

  ulimit {
    name = "nofile"
    soft = 1024
    hard = 2048
  }
}
`, name)
}

// TestAccPodmanContainer_withMounts creates a container with a tmpfs mount.
func TestAccPodmanContainer_withMounts(t *testing.T) {
	name := "tf-test-" + acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckPodmanContainerDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccPodmanContainerConfig_withMounts(name),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("podman_container.test", "name", name),
					resource.TestCheckTypeSetElemNestedAttrs("podman_container.test", "mounts.*", map[string]string{
						"target": "/mnt/tmpfs",
						"type":   "tmpfs",
					}),
				),
			},
		},
	})
}

func testAccPodmanContainerConfig_withMounts(name string) string {
	return providerConfig() + fmt.Sprintf(`
resource "podman_image" "test" {
  name = "quay.io/podman/stable:latest"
  keep_locally = true
}

resource "podman_container" "test" {
  name    = "%s"
  image   = podman_image.test.image_id
  command = ["sleep", "300"]

  mounts {
    target = "/mnt/tmpfs"
    type   = "tmpfs"
  }
}
`, name)
}

// TestAccPodmanContainer_withSysctls creates a container with sysctl settings.
func TestAccPodmanContainer_withSysctls(t *testing.T) {
	name := "tf-test-" + acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckPodmanContainerDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccPodmanContainerConfig_withSysctls(name),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("podman_container.test", "name", name),
					resource.TestCheckResourceAttr("podman_container.test", "sysctls.net.ipv4.ip_forward", "1"),
				),
			},
		},
	})
}

func testAccPodmanContainerConfig_withSysctls(name string) string {
	return providerConfig() + fmt.Sprintf(`
resource "podman_image" "test" {
  name = "quay.io/podman/stable:latest"
  keep_locally = true
}

resource "podman_container" "test" {
  name    = "%s"
  image   = podman_image.test.image_id
  command = ["sleep", "300"]

  sysctls = {
    "net.ipv4.ip_forward" = "1"
  }
}
`, name)
}

// TestAccPodmanContainer_withResourceLimits creates a container with memory and CPU constraints.
func TestAccPodmanContainer_withResourceLimits(t *testing.T) {
	name := "tf-test-" + acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckPodmanContainerDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccPodmanContainerConfig_withResourceLimits(name),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("podman_container.test", "name", name),
					resource.TestCheckResourceAttr("podman_container.test", "memory", "67108864"),
					resource.TestCheckResourceAttr("podman_container.test", "cpu_shares", "512"),
				),
			},
		},
	})
}

func testAccPodmanContainerConfig_withResourceLimits(name string) string {
	return providerConfig() + fmt.Sprintf(`
resource "podman_image" "test" {
  name = "quay.io/podman/stable:latest"
  keep_locally = true
}

resource "podman_container" "test" {
  name       = "%s"
  image      = podman_image.test.image_id
  command    = ["sleep", "300"]
  memory     = 67108864
  cpu_shares = 512
}
`, name)
}

// TestAccPodmanContainer_withLogs creates a container with logs enabled and verifies output.
func TestAccPodmanContainer_withLogs(t *testing.T) {
	name := "tf-test-" + acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckPodmanContainerDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccPodmanContainerConfig_withLogs(name),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("podman_container.test", "name", name),
					resource.TestCheckResourceAttr("podman_container.test", "logs", "true"),
					resource.TestCheckResourceAttrSet("podman_container.test", "container_logs"),
				),
			},
		},
	})
}

func testAccPodmanContainerConfig_withLogs(name string) string {
	return providerConfig() + fmt.Sprintf(`
resource "podman_image" "test" {
  name = "quay.io/podman/stable:latest"
  keep_locally = true
}

resource "podman_container" "test" {
  name    = "%s"
  image   = podman_image.test.image_id
  command = ["echo", "hello"]
  attach  = true
  logs    = true
  must_run = false
}
`, name)
}

// TestAccPodmanContainer_notStarted creates a container that is not started.
func TestAccPodmanContainer_notStarted(t *testing.T) {
	name := "tf-test-" + acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckPodmanContainerDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccPodmanContainerConfig_notStarted(name),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("podman_container.test", "name", name),
					resource.TestCheckResourceAttr("podman_container.test", "start", "false"),
				),
			},
		},
	})
}

func testAccPodmanContainerConfig_notStarted(name string) string {
	return providerConfig() + fmt.Sprintf(`
resource "podman_image" "test" {
  name = "quay.io/podman/stable:latest"
  keep_locally = true
}

resource "podman_container" "test" {
  name     = "%s"
  image    = podman_image.test.image_id
  command  = ["sleep", "300"]
  start    = false
  must_run = false
}
`, name)
}

// TestAccPodmanContainer_update creates a container then updates memory from 64MB to 128MB.
func TestAccPodmanContainer_update(t *testing.T) {
	name := "tf-test-" + acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckPodmanContainerDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccPodmanContainerConfig_update(name, 67108864),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("podman_container.test", "name", name),
					resource.TestCheckResourceAttr("podman_container.test", "memory", "67108864"),
				),
			},
			{
				Config: testAccPodmanContainerConfig_update(name, 134217728),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("podman_container.test", "name", name),
					resource.TestCheckResourceAttr("podman_container.test", "memory", "134217728"),
				),
			},
		},
	})
}

func testAccPodmanContainerConfig_update(name string, memory int) string {
	return providerConfig() + fmt.Sprintf(`
resource "podman_image" "test" {
  name = "quay.io/podman/stable:latest"
  keep_locally = true
}

resource "podman_container" "test" {
  name    = "%s"
  image   = podman_image.test.image_id
  command = ["sleep", "300"]
  memory  = %d
}
`, name, memory)
}
