package provider

import (
	"fmt"
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

// TestAccPodmanContainer_waitMode tests the wait code path with wait = true.
func TestAccPodmanContainer_waitMode(t *testing.T) {
	name := "tf-test-" + acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckPodmanContainerDestroy,
		Steps: []resource.TestStep{
			{
				Config: providerConfig() + fmt.Sprintf(`
resource "podman_image" "test" {
  name = "quay.io/podman/stable:latest"
  keep_locally = true
}

resource "podman_container" "test" {
  name         = "%s"
  image        = podman_image.test.image_id
  command      = ["sh", "-c", "echo done"]
  wait         = true
  wait_timeout = 30
  must_run     = false
}
`, name),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("podman_container.test", "name", name),
					resource.TestCheckResourceAttr("podman_container.test", "wait", "true"),
					resource.TestCheckResourceAttr("podman_container.test", "wait_timeout", "30"),
					resource.TestCheckResourceAttrSet("podman_container.test", "exit_code"),
				),
			},
		},
	})
}

// TestAccPodmanContainer_attachMode tests the attach code path with attach = true.
func TestAccPodmanContainer_attachMode(t *testing.T) {
	name := "tf-test-" + acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckPodmanContainerDestroy,
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
  command  = ["sh", "-c", "echo attached"]
  attach   = true
  logs     = true
  must_run = false
}
`, name),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("podman_container.test", "name", name),
					resource.TestCheckResourceAttr("podman_container.test", "attach", "true"),
					resource.TestCheckResourceAttr("podman_container.test", "logs", "true"),
					resource.TestCheckResourceAttrSet("podman_container.test", "container_logs"),
					resource.TestCheckResourceAttrSet("podman_container.test", "exit_code"),
				),
			},
		},
	})
}

// TestAccPodmanContainer_withTmpfs tests the tmpfs code path.
func TestAccPodmanContainer_withTmpfs(t *testing.T) {
	name := "tf-test-" + acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckPodmanContainerDestroy,
		Steps: []resource.TestStep{
			{
				Config: providerConfig() + fmt.Sprintf(`
resource "podman_image" "test" {
  name = "quay.io/podman/stable:latest"
  keep_locally = true
}

resource "podman_container" "test" {
  name    = "%s"
  image   = podman_image.test.image_id
  command = ["sleep", "300"]

  tmpfs = {
    "/run" = "rw,noexec,nosuid,size=65536k"
  }
}
`, name),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("podman_container.test", "name", name),
					resource.TestCheckResourceAttrSet("podman_container.test", "tmpfs./run"),
				),
			},
		},
	})
}

// TestAccPodmanContainer_withStorageOpts tests the storage_opts code path.
func TestAccPodmanContainer_withStorageOpts(t *testing.T) {
	t.Skip("storage_opts not supported in rootless Podman")
	name := "tf-test-" + acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckPodmanContainerDestroy,
		Steps: []resource.TestStep{
			{
				Config: providerConfig() + fmt.Sprintf(`
resource "podman_image" "test" {
  name = "quay.io/podman/stable:latest"
  keep_locally = true
}

resource "podman_container" "test" {
  name    = "%s"
  image   = podman_image.test.image_id
  command = ["sleep", "300"]

  storage_opts = {
    "size" = "10G"
  }
}
`, name),
				ExpectNonEmptyPlan: true,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("podman_container.test", "name", name),
				),
			},
		},
	})
}

// TestAccPodmanContainer_withWorkingDir tests the working_dir code path.
func TestAccPodmanContainer_withWorkingDir(t *testing.T) {
	name := "tf-test-" + acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckPodmanContainerDestroy,
		Steps: []resource.TestStep{
			{
				Config: providerConfig() + fmt.Sprintf(`
resource "podman_image" "test" {
  name = "quay.io/podman/stable:latest"
  keep_locally = true
}

resource "podman_container" "test" {
  name        = "%s"
  image       = podman_image.test.image_id
  command     = ["sleep", "300"]
  working_dir = "/tmp"
}
`, name),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("podman_container.test", "name", name),
					resource.TestCheckResourceAttr("podman_container.test", "working_dir", "/tmp"),
				),
			},
		},
	})
}

// TestAccPodmanContainer_withUser tests the user code path.
func TestAccPodmanContainer_withUser(t *testing.T) {
	name := "tf-test-" + acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckPodmanContainerDestroy,
		Steps: []resource.TestStep{
			{
				Config: providerConfig() + fmt.Sprintf(`
resource "podman_image" "test" {
  name = "quay.io/podman/stable:latest"
  keep_locally = true
}

resource "podman_container" "test" {
  name    = "%s"
  image   = podman_image.test.image_id
  command = ["sleep", "300"]
  user    = "nobody"
}
`, name),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("podman_container.test", "name", name),
					resource.TestCheckResourceAttr("podman_container.test", "user", "nobody"),
				),
			},
		},
	})
}

// TestAccPodmanContainer_withGroupAdd tests the group_add code path.
func TestAccPodmanContainer_withGroupAdd(t *testing.T) {
	name := "tf-test-" + acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckPodmanContainerDestroy,
		Steps: []resource.TestStep{
			{
				Config: providerConfig() + fmt.Sprintf(`
resource "podman_image" "test" {
  name = "quay.io/podman/stable:latest"
  keep_locally = true
}

resource "podman_container" "test" {
  name      = "%s"
  image     = podman_image.test.image_id
  command   = ["sleep", "300"]
  group_add = ["audio"]
}
`, name),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("podman_container.test", "name", name),
					resource.TestCheckResourceAttr("podman_container.test", "group_add.#", "1"),
				),
			},
		},
	})
}

// TestAccPodmanContainer_withInit tests the init code path.
func TestAccPodmanContainer_withInit(t *testing.T) {
	name := "tf-test-" + acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckPodmanContainerDestroy,
		Steps: []resource.TestStep{
			{
				Config: providerConfig() + fmt.Sprintf(`
resource "podman_image" "test" {
  name = "quay.io/podman/stable:latest"
  keep_locally = true
}

resource "podman_container" "test" {
  name    = "%s"
  image   = podman_image.test.image_id
  command = ["sleep", "300"]
  init    = true
}
`, name),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("podman_container.test", "name", name),
					resource.TestCheckResourceAttr("podman_container.test", "init", "true"),
				),
			},
		},
	})
}

// TestAccPodmanContainer_withLogDriver tests the log_driver code path.
func TestAccPodmanContainer_withLogDriver(t *testing.T) {
	name := "tf-test-" + acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckPodmanContainerDestroy,
		Steps: []resource.TestStep{
			{
				Config: providerConfig() + fmt.Sprintf(`
resource "podman_image" "test" {
  name = "quay.io/podman/stable:latest"
  keep_locally = true
}

resource "podman_container" "test" {
  name       = "%s"
  image      = podman_image.test.image_id
  command    = ["sleep", "300"]
  log_driver = "journald"
}
`, name),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("podman_container.test", "name", name),
					resource.TestCheckResourceAttr("podman_container.test", "log_driver", "journald"),
				),
			},
		},
	})
}

// TestAccPodmanContainer_withRestartPolicy tests the restart policy code path.
func TestAccPodmanContainer_withRestartPolicy(t *testing.T) {
	name := "tf-test-" + acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckPodmanContainerDestroy,
		Steps: []resource.TestStep{
			{
				Config: providerConfig() + fmt.Sprintf(`
resource "podman_image" "test" {
  name = "quay.io/podman/stable:latest"
  keep_locally = true
}

resource "podman_container" "test" {
  name            = "%s"
  image           = podman_image.test.image_id
  command         = ["sleep", "300"]
  restart         = "on-failure"
  max_retry_count = 3
}
`, name),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("podman_container.test", "name", name),
					resource.TestCheckResourceAttr("podman_container.test", "restart", "on-failure"),
					resource.TestCheckResourceAttr("podman_container.test", "max_retry_count", "3"),
				),
			},
		},
	})
}

// TestAccPodmanContainer_withHostname tests the hostname code path.
func TestAccPodmanContainer_withHostname(t *testing.T) {
	name := "tf-test-" + acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckPodmanContainerDestroy,
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
  command  = ["sleep", "300"]
  hostname = "testhost"
}
`, name),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("podman_container.test", "name", name),
					resource.TestCheckResourceAttr("podman_container.test", "hostname", "testhost"),
				),
			},
		},
	})
}

// TestAccPodmanContainer_withDomainname tests the domainname code path.
func TestAccPodmanContainer_withDomainname(t *testing.T) {
	name := "tf-test-" + acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckPodmanContainerDestroy,
		Steps: []resource.TestStep{
			{
				Config: providerConfig() + fmt.Sprintf(`
resource "podman_image" "test" {
  name = "quay.io/podman/stable:latest"
  keep_locally = true
}

resource "podman_container" "test" {
  name       = "%s"
  image      = podman_image.test.image_id
  command    = ["sleep", "300"]
  domainname = "test.local"
}
`, name),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("podman_container.test", "name", name),
				),
			},
		},
	})
}

// TestAccPodmanContainer_withStopSignal tests the stop_signal code path.
func TestAccPodmanContainer_withStopSignal(t *testing.T) {
	name := "tf-test-" + acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckPodmanContainerDestroy,
		Steps: []resource.TestStep{
			{
				Config: providerConfig() + fmt.Sprintf(`
resource "podman_image" "test" {
  name = "quay.io/podman/stable:latest"
  keep_locally = true
}

resource "podman_container" "test" {
  name        = "%s"
  image       = podman_image.test.image_id
  command     = ["sleep", "300"]
  stop_signal = "SIGTERM"
}
`, name),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("podman_container.test", "name", name),
				),
			},
		},
	})
}

// TestAccPodmanContainer_readOnly tests the read_only code path.
func TestAccPodmanContainer_readOnly(t *testing.T) {
	name := "tf-test-" + acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckPodmanContainerDestroy,
		Steps: []resource.TestStep{
			{
				Config: providerConfig() + fmt.Sprintf(`
resource "podman_image" "test" {
  name = "quay.io/podman/stable:latest"
  keep_locally = true
}

resource "podman_container" "test" {
  name      = "%s"
  image     = podman_image.test.image_id
  command   = ["sleep", "300"]
  read_only = true
}
`, name),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("podman_container.test", "name", name),
					resource.TestCheckResourceAttr("podman_container.test", "read_only", "true"),
				),
			},
		},
	})
}

// TestAccPodmanContainer_publishAllPorts tests the publish_all_ports code path.
func TestAccPodmanContainer_publishAllPorts(t *testing.T) {
	name := "tf-test-" + acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckPodmanContainerDestroy,
		Steps: []resource.TestStep{
			{
				Config: providerConfig() + fmt.Sprintf(`
resource "podman_image" "test" {
  name = "quay.io/podman/stable:latest"
  keep_locally = true
}

resource "podman_container" "test" {
  name              = "%s"
  image             = podman_image.test.image_id
  command           = ["sleep", "300"]
  publish_all_ports = true
}
`, name),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("podman_container.test", "name", name),
					resource.TestCheckResourceAttr("podman_container.test", "publish_all_ports", "true"),
				),
			},
		},
	})
}

// TestAccPodmanContainer_destroyGrace tests the destroy_grace_seconds code path.
func TestAccPodmanContainer_destroyGrace(t *testing.T) {
	name := "tf-test-" + acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckPodmanContainerDestroy,
		Steps: []resource.TestStep{
			{
				Config: providerConfig() + fmt.Sprintf(`
resource "podman_image" "test" {
  name = "quay.io/podman/stable:latest"
  keep_locally = true
}

resource "podman_container" "test" {
  name                  = "%s"
  image                 = podman_image.test.image_id
  command               = ["sleep", "300"]
  destroy_grace_seconds = 5
}
`, name),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("podman_container.test", "name", name),
					resource.TestCheckResourceAttr("podman_container.test", "destroy_grace_seconds", "5"),
				),
			},
		},
	})
}

// TestAccPodmanContainer_rmMode tests the rm (auto-remove) code path.
func TestAccPodmanContainer_rmMode(t *testing.T) {
	name := "tf-test-" + acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckPodmanContainerDestroy,
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
  command  = ["true"]
  rm       = true
  must_run = false
}
`, name),
				ExpectNonEmptyPlan: true, // Container auto-removes, so refresh shows it as gone
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("podman_container.test", "name", name),
				),
			},
		},
	})
}

// TestAccPodmanContainer_uploadBase64 tests the base64 upload code path.
func TestAccPodmanContainer_uploadBase64(t *testing.T) {
	name := "tf-test-" + acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckPodmanContainerDestroy,
		Steps: []resource.TestStep{
			{
				Config: providerConfig() + fmt.Sprintf(`
resource "podman_image" "test" {
  name = "quay.io/podman/stable:latest"
  keep_locally = true
}

resource "podman_container" "test" {
  name    = "%s"
  image   = podman_image.test.image_id
  command = ["sleep", "300"]

  upload {
    file           = "/tmp/test.bin"
    content_base64 = "aGVsbG8="
  }
}
`, name),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("podman_container.test", "name", name),
					resource.TestCheckTypeSetElemNestedAttrs("podman_container.test", "upload.*", map[string]string{
						"file":           "/tmp/test.bin",
						"content_base64": "aGVsbG8=",
					}),
				),
			},
		},
	})
}

// TestAccPodmanContainer_uploadSource tests the source file upload code path.
func TestAccPodmanContainer_uploadSource(t *testing.T) {
	name := "tf-test-" + acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)

	tmpFile, err := os.CreateTemp("", "tf-test-upload-*.txt")
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())

	if _, err := tmpFile.WriteString("hello from source file"); err != nil {
		t.Fatalf("failed to write temp file: %v", err)
	}
	tmpFile.Close()

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckPodmanContainerDestroy,
		Steps: []resource.TestStep{
			{
				Config: providerConfig() + fmt.Sprintf(`
resource "podman_image" "test" {
  name = "quay.io/podman/stable:latest"
  keep_locally = true
}

resource "podman_container" "test" {
  name    = "%s"
  image   = podman_image.test.image_id
  command = ["sleep", "300"]

  upload {
    file   = "/tmp/test.txt"
    source = "%s"
  }
}
`, name, tmpFile.Name()),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("podman_container.test", "name", name),
					resource.TestCheckTypeSetElemNestedAttrs("podman_container.test", "upload.*", map[string]string{
						"file":   "/tmp/test.txt",
						"source": tmpFile.Name(),
					}),
				),
			},
		},
	})
}

// TestAccPodmanContainer_multipleNetworks tests the multi-network connect code path.
func TestAccPodmanContainer_multipleNetworks(t *testing.T) {
	containerName := "tf-test-" + acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)
	networkName1 := "tf-test-" + acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)
	networkName2 := "tf-test-" + acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckPodmanContainerDestroy,
		Steps: []resource.TestStep{
			{
				Config: providerConfig() + fmt.Sprintf(`
resource "podman_image" "test" {
  name = "quay.io/podman/stable:latest"
  keep_locally = true
}

resource "podman_network" "net1" {
  name   = "%s"
  driver = "bridge"

  ipam_config {
    subnet  = "172.21.0.0/16"
    gateway = "172.21.0.1"
  }
}

resource "podman_network" "net2" {
  name   = "%s"
  driver = "bridge"

  ipam_config {
    subnet  = "172.22.0.0/16"
    gateway = "172.22.0.1"
  }
}

resource "podman_container" "test" {
  name    = "%s"
  image   = podman_image.test.image_id
  command = ["sleep", "300"]

  networks_advanced {
    name         = podman_network.net1.name
    ipv4_address = "172.21.0.10"
  }

  networks_advanced {
    name         = podman_network.net2.name
    ipv4_address = "172.22.0.10"
  }
}
`, networkName1, networkName2, containerName),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("podman_container.test", "name", containerName),
					resource.TestCheckResourceAttr("podman_container.test", "networks_advanced.#", "2"),
				),
			},
		},
	})
}
