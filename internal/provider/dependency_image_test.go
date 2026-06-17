package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

// TestAccDepImageNameChangeRecreatesContainer is the seed dependency test and a
// regression test for the image-name DiffSuppress bug: a podman_image whose name
// changes from "docker.io/library/alpine" to "docker.io/library/alpine:3.20"
// (the old name is a string prefix of the new one) must recreate a container
// that references the image's name.
func TestAccDepImageNameChangeRecreatesContainer(t *testing.T) {
	cname := randDepName("imgname")
	var before, after string

	cfg := func(imageName string) string {
		return fmt.Sprintf(`
resource "podman_image" "base" {
  name         = %q
  keep_locally = true
}

resource "podman_container" "test" {
  name  = %q
  image = podman_image.base.name
  start = false
}
`, imageName, cname)
	}

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
			testAccDepPullImages(t, depImageLatest, depImage)
		},
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckPodmanContainerDestroy,
		Steps: []resource.TestStep{
			{
				Config: cfg(depImageLatest),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckPodmanContainerExists("podman_container.test"),
					captureID("podman_container.test", &before),
				),
			},
			{
				Config: cfg(depImage),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckPodmanContainerExists("podman_container.test"),
					captureID("podman_container.test", &after),
					requireRecreated("container on image name change", &before, &after),
				),
			},
		},
	})
}

// TestAccDepImageIDChangeRecreatesContainer wires the container to the image's
// computed image_id. Switching the image's name to a different image changes the
// image_id, which must recreate the container.
func TestAccDepImageIDChangeRecreatesContainer(t *testing.T) {
	cname := randDepName("imgid")
	var before, after string

	cfg := func(imageName string) string {
		return fmt.Sprintf(`
resource "podman_image" "base" {
  name         = %q
  keep_locally = true
}

resource "podman_container" "test" {
  name  = %q
  image = podman_image.base.image_id
  start = false
}
`, imageName, cname)
	}

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
			testAccDepPullImages(t, depImage, "docker.io/library/alpine:3.19")
		},
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckPodmanContainerDestroy,
		Steps: []resource.TestStep{
			{
				Config: cfg(depImage),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckPodmanContainerExists("podman_container.test"),
					captureID("podman_container.test", &before),
				),
			},
			{
				Config: cfg("docker.io/library/alpine:3.19"),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckPodmanContainerExists("podman_container.test"),
					captureID("podman_container.test", &after),
					requireRecreated("container on image_id change", &before, &after),
				),
			},
		},
	})
}

// TestAccDepContainerImageTagChangeRecreates uses a literal image reference (no
// image resource) and changes the tag, which must recreate the container.
func TestAccDepContainerImageTagChangeRecreates(t *testing.T) {
	cname := randDepName("imgtag")
	var before, after string

	cfg := func(imageRef string) string {
		return fmt.Sprintf(`
resource "podman_container" "test" {
  name  = %q
  image = %q
  start = false
}
`, cname, imageRef)
	}

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
			testAccDepPullImages(t, depImage, "docker.io/library/alpine:3.19")
		},
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckPodmanContainerDestroy,
		Steps: []resource.TestStep{
			{
				Config: cfg(depImage),
				Check: resource.ComposeTestCheckFunc(
					captureID("podman_container.test", &before),
				),
			},
			{
				Config: cfg("docker.io/library/alpine:3.19"),
				Check: resource.ComposeTestCheckFunc(
					captureID("podman_container.test", &after),
					requireRecreated("container on image tag change", &before, &after),
				),
			},
		},
	})
}

// TestAccDepContainerEnvChangeInPlace verifies the negative case: changing an
// in-place-updatable attribute (env) must NOT recreate the container.
func TestAccDepContainerEnvChangeInPlace(t *testing.T) {
	cname := randDepName("env")
	var before, after string

	cfg := func(val string) string {
		return fmt.Sprintf(`
resource "podman_container" "test" {
  name  = %q
  image = %q
  start = false
  env   = ["FOO=%s"]
}
`, cname, depImage, val)
	}

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckPodmanContainerDestroy,
		Steps: []resource.TestStep{
			{
				Config: cfg("one"),
				Check:  captureID("podman_container.test", &before),
			},
			{
				Config: cfg("two"),
				Check: resource.ComposeTestCheckFunc(
					captureID("podman_container.test", &after),
					requireNotRecreated("container on env change", &before, &after),
				),
			},
		},
	})
}
