package provider

import (
	"fmt"
	"testing"
)

// hclVolC renders a named volume plus a container that mounts it through the
// (ForceNew) volumes block. body overrides extra lines of the volumes block.
func hclVolC(vol, cname, volBody string) string {
	return fmt.Sprintf(`
resource "podman_volume" "v" {
  name = %q
}

resource "podman_container" "test" {
  name  = %q
  image = %q
  start = false
  volumes {
    volume_name    = podman_volume.v.name
    %s
  }
}
`, vol, cname, depImage, volBody)
}

// --- Volume resource recreation (tracked: the volume) ---

func TestAccDepVolumeNameChangeRecreatesVolume(t *testing.T) {
	runDepRecreateTest(t, depRecreateTest{
		resourceName: "podman_volume.v", label: "volume name", recreate: true,
		configA: fmt.Sprintf(`resource "podman_volume" "v" { name = %q }`, randDepName("vol-a")),
		configB: fmt.Sprintf(`resource "podman_volume" "v" { name = %q }`, randDepName("vol-b")),
	})
}

// Note: podman_volume uses its name as the Terraform ID, so recreation is only
// observable when the name changes. Non-name ForceNew changes recreate the
// volume too, but with the same ID, so we assert those through a dependent
// container instead (whose ID is a fresh container ID on every replacement).

func TestAccDepVolumeNameChangeWithExplicitDriverRecreatesVolume(t *testing.T) {
	cfg := func(name string) string {
		return fmt.Sprintf(`resource "podman_volume" "v" {
  name   = %q
  driver = "local"
}`, name)
	}
	runDepRecreateTest(t, depRecreateTest{
		resourceName: "podman_volume.v", label: "volume name (local driver)", recreate: true,
		configA: cfg(randDepName("vdrv-a")), configB: cfg(randDepName("vdrv-b")),
	})
}

func TestAccDepVolumeSwapRecreatesContainer(t *testing.T) {
	v1 := randDepName("vswap1")
	v2 := randDepName("vswap2")
	cname := randDepName("vswapc")
	cfg := func(which string) string {
		return fmt.Sprintf(`
resource "podman_volume" "v1" { name = %q }
resource "podman_volume" "v2" { name = %q }

resource "podman_container" "test" {
  name  = %q
  image = %q
  start = false
  volumes {
    volume_name    = podman_volume.%s.name
    container_path = "/data"
  }
}
`, v1, v2, cname, depImage, which)
	}
	runDepRecreateTest(t, depRecreateTest{
		resourceName: "podman_container.test", label: "container on volume swap", recreate: true,
		configA: cfg("v1"), configB: cfg("v2"),
	})
}

// --- Container that depends on a named volume; container-side ForceNew leaf
// changes recreate the container while the volume is left intact ---

func TestAccDepVolumeMountContainerPathChangeRecreatesContainer(t *testing.T) {
	vol := randDepName("vmnt")
	cname := randDepName("vmntc")
	runDepRecreateTest(t, depRecreateTest{
		resourceName: "podman_container.test", label: "volume mount path", recreate: true,
		configA: hclVolC(vol, cname, `container_path = "/data"`),
		configB: hclVolC(vol, cname, `container_path = "/data2"`),
	})
}

func TestAccDepVolumeMountReadOnlyChangeRecreatesContainer(t *testing.T) {
	vol := randDepName("vro")
	cname := randDepName("vroc")
	runDepRecreateTest(t, depRecreateTest{
		resourceName: "podman_container.test", label: "volume mount read_only", recreate: true,
		configA: hclVolC(vol, cname, "container_path = \"/data\"\n    read_only      = false"),
		configB: hclVolC(vol, cname, "container_path = \"/data\"\n    read_only      = true"),
	})
}

// --- Container that depends on a named volume via the mounts block ---

func hclVolMountsC(vol, cname, mountBody string) string {
	return fmt.Sprintf(`
resource "podman_volume" "v" {
  name = %q
}

resource "podman_container" "test" {
  name  = %q
  image = %q
  start = false
  mounts {
    type   = "volume"
    source = podman_volume.v.name
    %s
  }
}
`, vol, cname, depImage, mountBody)
}

func TestAccDepVolumeMountsTargetChangeRecreatesContainer(t *testing.T) {
	vol := randDepName("mtgt")
	cname := randDepName("mtgtc")
	runDepRecreateTest(t, depRecreateTest{
		resourceName: "podman_container.test", label: "mounts target", recreate: true,
		configA: hclVolMountsC(vol, cname, `target = "/data"`),
		configB: hclVolMountsC(vol, cname, `target = "/data2"`),
	})
}

func TestAccDepVolumeMountsReadOnlyChangeRecreatesContainer(t *testing.T) {
	vol := randDepName("mro")
	cname := randDepName("mroc")
	runDepRecreateTest(t, depRecreateTest{
		resourceName: "podman_container.test", label: "mounts read_only", recreate: true,
		configA: hclVolMountsC(vol, cname, "target    = \"/data\"\n    read_only = false"),
		configB: hclVolMountsC(vol, cname, "target    = \"/data\"\n    read_only = true"),
	})
}

// --- Volume identity change propagates to the dependent container ---

func TestAccDepVolumeNameChangeRecreatesDependentContainer(t *testing.T) {
	cname := randDepName("vdepc")
	cfg := func(vol string) string { return hclVolC(vol, cname, `container_path = "/data"`) }
	runDepRecreateTest(t, depRecreateTest{
		resourceName: "podman_container.test", label: "container on volume name change", recreate: true,
		configA: cfg(randDepName("vdep-a")),
		configB: cfg(randDepName("vdep-b")),
	})
}

// --- Anonymous volume (no volume resource) ---

func TestAccDepContainerAnonymousVolumePathChangeRecreates(t *testing.T) {
	cname := randDepName("anonv")
	cfg := func(path string) string {
		return fmt.Sprintf(`
resource "podman_container" "test" {
  name  = %q
  image = %q
  start = false
  volumes {
    container_path = %q
  }
}
`, cname, depImage, path)
	}
	runDepRecreateTest(t, depRecreateTest{
		resourceName: "podman_container.test", label: "anonymous volume", recreate: true,
		configA: cfg("/data"), configB: cfg("/data2"),
	})
}

// --- Two containers sharing one named volume ---

func TestAccDepVolumeSharedByTwoContainers(t *testing.T) {
	vol := randDepName("shvol")
	c1 := randDepName("shc1")
	c2 := randDepName("shc2")
	cfg := func(path string) string {
		return fmt.Sprintf(`
resource "podman_volume" "v" {
  name = %q
}

resource "podman_container" "test" {
  name  = %q
  image = %q
  start = false
  volumes {
    volume_name    = podman_volume.v.name
    container_path = %q
  }
}

resource "podman_container" "other" {
  name  = %q
  image = %q
  start = false
  volumes {
    volume_name    = podman_volume.v.name
    container_path = "/shared"
  }
}
`, vol, c1, depImage, path, c2, depImage)
	}
	runDepRecreateTest(t, depRecreateTest{
		resourceName: "podman_container.test", label: "shared volume consumer", recreate: true,
		configA: cfg("/data"), configB: cfg("/data2"),
	})
}
