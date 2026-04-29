package provider

import (
	"archive/tar"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/client"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

// prepareRootfs pulls a small image, exports its filesystem, and untars
// the result to a fresh temporary directory. The returned path is the
// rootfs the test points podman_container.rootfs at; the cleanup func
// removes the directory and the helper container.
func prepareRootfs(t *testing.T) (string, func()) {
	t.Helper()
	ctx := context.Background()

	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		t.Fatalf("failed to create podman client: %v", err)
	}

	const ref = "docker.io/library/alpine:3.20"

	pullResp, err := cli.ImagePull(ctx, ref, image.PullOptions{})
	if err != nil {
		t.Fatalf("failed to pull %s: %v", ref, err)
	}
	if _, err := io.Copy(io.Discard, pullResp); err != nil {
		t.Fatalf("failed to drain image pull response: %v", err)
	}
	pullResp.Close()

	createResp, err := cli.ContainerCreate(ctx,
		&container.Config{Image: ref, Cmd: []string{"/bin/true"}},
		&container.HostConfig{},
		nil, nil, "")
	if err != nil {
		t.Fatalf("failed to create helper container: %v", err)
	}
	helperID := createResp.ID

	exportRC, err := cli.ContainerExport(ctx, helperID)
	if err != nil {
		_ = cli.ContainerRemove(ctx, helperID, container.RemoveOptions{Force: true})
		t.Fatalf("failed to export container: %v", err)
	}
	defer exportRC.Close()

	rootfs, err := os.MkdirTemp("", "tf-podman-rootfs-")
	if err != nil {
		_ = cli.ContainerRemove(ctx, helperID, container.RemoveOptions{Force: true})
		t.Fatalf("failed to create rootfs tempdir: %v", err)
	}

	if err := untar(exportRC, rootfs); err != nil {
		_ = cli.ContainerRemove(ctx, helperID, container.RemoveOptions{Force: true})
		_ = os.RemoveAll(rootfs)
		t.Fatalf("failed to extract rootfs: %v", err)
	}

	if err := cli.ContainerRemove(ctx, helperID, container.RemoveOptions{Force: true}); err != nil {
		t.Logf("warning: failed to remove helper container %s: %v", helperID, err)
	}

	cleanup := func() {
		_ = os.RemoveAll(rootfs)
	}
	return rootfs, cleanup
}

// untar extracts a tar stream into dest. Handles regular files, dirs, and
// symlinks — enough for an exported container filesystem.
func untar(r io.Reader, dest string) error {
	tr := tar.NewReader(r)
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return err
		}
		// Reject path traversal
		clean := filepath.Clean(hdr.Name)
		if strings.HasPrefix(clean, "..") || strings.Contains(clean, "/../") {
			continue
		}
		target := filepath.Join(dest, clean)

		switch hdr.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(target, os.FileMode(hdr.Mode)&0o7777); err != nil {
				return err
			}
		case tar.TypeReg, tar.TypeRegA:
			if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
				return err
			}
			f, err := os.OpenFile(target, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, os.FileMode(hdr.Mode)&0o7777)
			if err != nil {
				return err
			}
			if _, err := io.Copy(f, tr); err != nil {
				f.Close()
				return err
			}
			f.Close()
		case tar.TypeSymlink:
			if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
				return err
			}
			_ = os.Remove(target)
			if err := os.Symlink(hdr.Linkname, target); err != nil {
				return err
			}
		case tar.TypeLink:
			if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
				return err
			}
			linkSrc := filepath.Join(dest, filepath.Clean(hdr.Linkname))
			_ = os.Remove(target)
			if err := os.Link(linkSrc, target); err != nil {
				// Fall back to copying if hard-link fails (e.g. cross-device)
				in, openErr := os.Open(linkSrc)
				if openErr != nil {
					return err
				}
				out, createErr := os.OpenFile(target, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o644)
				if createErr != nil {
					in.Close()
					return err
				}
				_, copyErr := io.Copy(out, in)
				in.Close()
				out.Close()
				if copyErr != nil {
					return copyErr
				}
			}
		default:
			// Skip device nodes, fifos etc — alpine's userland doesn't need them
			// for /bin/sleep to run.
		}
	}
}

// TestAccPodmanContainer_rootfs creates a container from an exploded
// rootfs (no image), exercising the libpod create path.
func TestAccPodmanContainer_rootfs(t *testing.T) {
	if os.Getenv("TF_ACC") == "" {
		t.Skip("TF_ACC not set; skipping acceptance test")
	}
	testAccPreCheck(t)

	rootfs, cleanup := prepareRootfs(t)
	defer cleanup()

	name := "tf-test-rootfs-" + acctest.RandStringFromCharSet(8, acctest.CharSetAlphaNum)

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckPodmanContainerDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccPodmanContainerConfig_rootfs(name, rootfs),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("podman_container.test", "name", name),
					resource.TestCheckResourceAttr("podman_container.test", "rootfs", rootfs),
					resource.TestCheckResourceAttr("podman_container.test", "exit_code", "0"),
				),
			},
		},
	})
}

func testAccPodmanContainerConfig_rootfs(name, rootfs string) string {
	return providerConfig() + fmt.Sprintf(`
resource "podman_container" "test" {
  name    = "%s"
  rootfs  = "%s"
  command = ["/bin/sh", "-c", "echo hello && exit 0"]

  start        = true
  wait         = true
  wait_timeout = 30
  must_run     = false
}
`, name, rootfs)
}

// TestAccPodmanContainer_rootfsOverlay verifies the rootfs_overlay flag
// reaches podman: writes inside the container should not survive
// because the rootfs is mounted overlayed.
func TestAccPodmanContainer_rootfsOverlay(t *testing.T) {
	if os.Getenv("TF_ACC") == "" {
		t.Skip("TF_ACC not set; skipping acceptance test")
	}
	testAccPreCheck(t)

	rootfs, cleanup := prepareRootfs(t)
	defer cleanup()

	name := "tf-test-rootfs-" + acctest.RandStringFromCharSet(8, acctest.CharSetAlphaNum)

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckPodmanContainerDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccPodmanContainerConfig_rootfsOverlay(name, rootfs),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("podman_container.test", "name", name),
					resource.TestCheckResourceAttr("podman_container.test", "rootfs", rootfs),
					resource.TestCheckResourceAttr("podman_container.test", "rootfs_overlay", "true"),
					resource.TestCheckResourceAttr("podman_container.test", "exit_code", "0"),
					func(_ *terraform.State) error {
						// The container wrote /overlay-marker; with overlay
						// mode the marker must not appear on the host rootfs.
						markerPath := filepath.Join(rootfs, "overlay-marker")
						if _, err := os.Stat(markerPath); !os.IsNotExist(err) {
							return fmt.Errorf("overlay-marker leaked to host rootfs at %s (err=%v)", markerPath, err)
						}
						return nil
					},
				),
			},
		},
	})
}

func testAccPodmanContainerConfig_rootfsOverlay(name, rootfs string) string {
	return providerConfig() + fmt.Sprintf(`
resource "podman_container" "test" {
  name           = "%s"
  rootfs         = "%s"
  rootfs_overlay = true
  command        = ["/bin/sh", "-c", "touch /overlay-marker && exit 0"]

  start        = true
  wait         = true
  wait_timeout = 30
  must_run     = false
}
`, name, rootfs)
}
