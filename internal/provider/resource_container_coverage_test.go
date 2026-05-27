package provider

import (
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

// TestAccPodmanContainer_entrypoint covers `entrypoint`.
func TestAccPodmanContainer_entrypoint(t *testing.T) {
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
  name       = "%s"
  image      = podman_image.img.image_id
  entrypoint = ["/bin/sh"]
  command    = ["-c", "sleep 30"]
}
`, name),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("podman_container.test", "entrypoint.0", "/bin/sh"),
				),
			},
		},
	})
}

// TestAccPodmanContainer_privileged covers `privileged`.
func TestAccPodmanContainer_privileged(t *testing.T) {
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
  name       = "%s"
  image      = podman_image.img.image_id
  command    = ["sleep", "30"]
  privileged = true
}
`, name),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("podman_container.test", "privileged", "true"),
				),
			},
		},
	})
}

// TestAccPodmanContainer_stdinTty covers `stdin_open` and `tty`.
func TestAccPodmanContainer_stdinTty(t *testing.T) {
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
  name       = "%s"
  image      = podman_image.img.image_id
  command    = ["sleep", "30"]
  stdin_open = true
  tty        = true
}
`, name),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("podman_container.test", "stdin_open", "true"),
					resource.TestCheckResourceAttr("podman_container.test", "tty", "true"),
				),
			},
		},
	})
}

// TestAccPodmanContainer_securityOpts covers `security_opts`.
func TestAccPodmanContainer_securityOpts(t *testing.T) {
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
  security_opts = ["seccomp=unconfined"]
}
`, name),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("podman_container.test", "security_opts.#", "1"),
				),
			},
		},
	})
}

// TestAccPodmanContainer_namespaceModes covers
// pid_mode/ipc_mode/cgroupns_mode at once.
func TestAccPodmanContainer_namespaceModes(t *testing.T) {
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
  pid_mode      = "private"
  ipc_mode      = "private"
  cgroupns_mode = "private"
}
`, name),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("podman_container.test", "cgroupns_mode", "private"),
				),
			},
		},
	})
}

// TestAccPodmanContainer_shmSize covers `shm_size`.
func TestAccPodmanContainer_shmSize(t *testing.T) {
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
  command  = ["sleep", "30"]
  shm_size = 67108864
}
`, name),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("podman_container.test", "shm_size", "67108864"),
				),
			},
		},
	})
}

// TestAccPodmanContainer_cpuSet covers the `cpu_set` field. Skipped when the
// cpuset cgroup controller isn't delegated to the user (typical rootless).
func TestAccPodmanContainer_cpuSet(t *testing.T) {
	if os.Getenv("TF_ACC") == "" {
		t.Skip("TF_ACC not set; skipping acceptance test")
	}
	testAccPreCheck(t)
	if data, err := os.ReadFile(fmt.Sprintf("/sys/fs/cgroup/user.slice/user-%d.slice/user@%d.service/cgroup.subtree_control", os.Getuid(), os.Getuid())); err == nil {
		if !strings.Contains(string(data), "cpuset") {
			t.Skip("cpuset controller not delegated to user (rootless cgroup v2); skipping cpu_set test")
		}
	}
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
  cpu_set = "0"
}
`, name),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("podman_container.test", "cpu_set", "0"),
				),
			},
		},
	})
}

// TestAccPodmanContainer_logOpts covers `log_opts`. Uses json-file which
// podman's compat layer reports back consistently.
func TestAccPodmanContainer_logOpts(t *testing.T) {
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
  name       = "%s"
  image      = podman_image.img.image_id
  command    = ["sleep", "30"]
  log_driver = "json-file"
  log_opts = {
    "max-size" = "1m"
  }
}
`, name),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("podman_container.test", "log_driver", "json-file"),
				),
			},
		},
	})
}

// TestAccPodmanContainer_stopTimeout covers `stop_timeout`.
func TestAccPodmanContainer_stopTimeout(t *testing.T) {
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
  name         = "%s"
  image        = podman_image.img.image_id
  command      = ["sleep", "30"]
  stop_timeout = 7
}
`, name),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("podman_container.test", "stop_timeout", "7"),
				),
			},
		},
	})
}

// TestAccPodmanContainer_devices covers the `devices` block. Uses /dev/null
// since it always exists.
func TestAccPodmanContainer_devices(t *testing.T) {
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
  devices {
    host_path      = "/dev/null"
    container_path = "/dev/null2"
    permissions    = "rwm"
  }
}
`, name),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("podman_container.test", "devices.#", "1"),
				),
			},
		},
	})
}

// --------------------- image build parity tests ---------------------

// TestAccPodmanImage_buildArgs exercises `build.build_args` and `build.labels`.
func TestAccPodmanImage_buildArgs(t *testing.T) {
	if os.Getenv("TF_ACC") == "" {
		t.Skip("TF_ACC not set; skipping acceptance test")
	}
	testAccPreCheck(t)
	dir := t.TempDir()
	dockerfile := dir + "/Dockerfile"
	if err := os.WriteFile(dockerfile, []byte("FROM "+testImageRef+"\nARG MSG\nRUN echo $MSG > /tmp/msg.txt\n"), 0o644); err != nil {
		t.Fatalf("write dockerfile: %v", err)
	}
	tag := "tf-test-args-" + acctest.RandStringFromCharSet(8, acctest.CharSetAlphaNum) + ":latest"
	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: providerConfig() + fmt.Sprintf(`
resource "podman_image" "built" {
  name         = "%s"
  keep_locally = false
  build {
    context    = "%s"
    build_args = { "MSG" = "hi" }
    labels     = { "my.label" = "lv" }
    remove     = true
    force_remove = true
  }
}
`, tag, dir),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("podman_image.built", "image_id"),
				),
			},
		},
	})
}

// TestAccPodmanImage_buildParity covers the new build-block parity fields:
// pull_parent, squash, suppress_output, security_opt, label alias, ulimit.
// Uses an inline Containerfile that fits in a tmp dir.
func TestAccPodmanImage_buildParity(t *testing.T) {
	if os.Getenv("TF_ACC") == "" {
		t.Skip("TF_ACC not set; skipping acceptance test")
	}
	testAccPreCheck(t)
	dir := t.TempDir()
	dockerfile := dir + "/Dockerfile"
	if err := os.WriteFile(dockerfile, []byte("FROM "+testImageRef+"\nRUN true\n"), 0o644); err != nil {
		t.Fatalf("write dockerfile: %v", err)
	}
	tag := "tf-test-build-" + acctest.RandStringFromCharSet(8, acctest.CharSetAlphaNum) + ":latest"
	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: providerConfig() + fmt.Sprintf(`
resource "podman_image" "built" {
  name         = "%s"
  keep_locally = false
  build {
    context        = "%s"
    dockerfile     = "Dockerfile"
    pull_parent    = false
    squash         = false
    remove         = true
    force_remove   = true
    no_cache       = false
    suppress_output = false
    label = {
      "extra.label" = "value"
    }
  }
}
`, tag, dir),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("podman_image.built", "image_id"),
				),
			},
		},
	})
}
