package provider

import (
	"fmt"
	"testing"
)

// podman_tag creates a new local tag (target_image) for a source image. A
// container that consumes tag.target_image therefore depends on the tag, and a
// change to the target tag recreates the container (image is ForceNew).

// hclTagContainer renders a tag of the workhorse image plus a container that
// uses the tag's target image. containerBody adds extra container lines.
func hclTagContainer(target, cname, containerBody string) string {
	return fmt.Sprintf(`
resource "podman_tag" "t" {
  source_image = %q
  target_image = %q
}

resource "podman_container" "test" {
  name  = %q
  image = podman_tag.t.target_image
  start = false
  %s
}
`, depImage, target, cname, containerBody)
}

func TestAccDepTagTargetTagChangeRecreatesContainer(t *testing.T) {
	repo := "localhost/" + randDepName("tagapp")
	cname := randDepName("tagc")
	runDepRecreateTest(t, depRecreateTest{
		resourceName: "podman_container.test", label: "container on tag target tag", recreate: true,
		configA: hclTagContainer(repo+":v1", cname, ""),
		configB: hclTagContainer(repo+":v2", cname, ""),
	})
}

func TestAccDepTagTargetRepoChangeRecreatesContainer(t *testing.T) {
	cname := randDepName("tagrepoc")
	runDepRecreateTest(t, depRecreateTest{
		resourceName: "podman_container.test", label: "container on tag target repo", recreate: true,
		configA: hclTagContainer("localhost/"+randDepName("repo1")+":latest", cname, ""),
		configB: hclTagContainer("localhost/"+randDepName("repo2")+":latest", cname, ""),
	})
}

func TestAccDepTagTargetChangeContainerCommandStillRecreates(t *testing.T) {
	repo := "localhost/" + randDepName("tagcmd")
	cname := randDepName("tagcmdc")
	runDepRecreateTest(t, depRecreateTest{
		resourceName: "podman_container.test", label: "tag target with command", recreate: true,
		configA: hclTagContainer(repo+":a", cname, `command = ["sleep", "infinity"]`),
		configB: hclTagContainer(repo+":b", cname, `command = ["sleep", "infinity"]`),
	})
}

// Image -> Tag -> Container chain: the tag's source is a podman_image resource.
func TestAccDepImageTagContainerChainTargetChange(t *testing.T) {
	repo := "localhost/" + randDepName("chainapp")
	cname := randDepName("chainc")
	cfg := func(target string) string {
		return fmt.Sprintf(`
resource "podman_image" "base" {
  name         = %q
  keep_locally = true
}

resource "podman_tag" "t" {
  source_image = podman_image.base.name
  target_image = %q
}

resource "podman_container" "test" {
  name  = %q
  image = podman_tag.t.target_image
  start = false
}
`, depImage, target, cname)
	}
	runDepRecreateTest(t, depRecreateTest{
		resourceName: "podman_container.test", label: "image->tag->container", recreate: true,
		pulls:   []string{depImage},
		configA: cfg(repo + ":1"),
		configB: cfg(repo + ":2"),
	})
}

// A tag's target consumed by two containers.
func TestAccDepTagSharedByTwoContainers(t *testing.T) {
	repo := "localhost/" + randDepName("tshare")
	c1 := randDepName("tshare1")
	c2 := randDepName("tshare2")
	cfg := func(tag string) string {
		return fmt.Sprintf(`
resource "podman_tag" "t" {
  source_image = %q
  target_image = %q
}

resource "podman_container" "test" {
  name  = %q
  image = podman_tag.t.target_image
  start = false
}

resource "podman_container" "other" {
  name  = %q
  image = podman_tag.t.target_image
  start = false
}
`, depImage, repo+":"+tag, c1, c2)
	}
	runDepRecreateTest(t, depRecreateTest{
		resourceName: "podman_container.test", label: "shared tag consumer", recreate: true,
		configA: cfg("one"), configB: cfg("two"),
	})
}

// Tag target consumed via an in-place container attribute (label) must not
// recreate the container.
func TestAccDepTagTargetChangeContainerLabelInPlace(t *testing.T) {
	repo := "localhost/" + randDepName("tlbl")
	cname := randDepName("tlblc")
	cfg := func(tag string) string {
		return fmt.Sprintf(`
resource "podman_tag" "t" {
  source_image = %q
  target_image = %q
}

resource "podman_container" "test" {
  name  = %q
  image = %q
  start = false
  labels {
    label = "image-ref"
    value = podman_tag.t.target_image
  }
}
`, depImage, repo+":"+tag, cname, depImage)
	}
	runDepRecreateTest(t, depRecreateTest{
		resourceName: "podman_container.test", label: "tag target via label", recreate: false,
		configA: cfg("x"), configB: cfg("y"),
	})
}
