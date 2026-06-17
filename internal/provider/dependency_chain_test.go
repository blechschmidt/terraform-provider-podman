package provider

import (
	"fmt"
	"testing"
)

const depImageAlt = "docker.io/library/alpine:3.19"

// fullStack renders a named volume, a secret, and a container that depends on
// all of them: the image, a volume mount, the secret (referenced from the
// container's command), a network mode, and an env var. Each chain test varies
// exactly one input between configA and configB.
func fullStack(cname, image, volPath, secData, netMode, env string) string {
	return fmt.Sprintf(`
resource "podman_volume" "v" {
  name = "%[1]s-vol"
}

resource "podman_secret" "s" {
  name = "%[1]s-sec"
  data = %[2]q
}

resource "podman_container" "test" {
  name         = %[1]q
  image        = %[3]q
  start        = false
  network_mode = %[4]q
  env          = [%[5]q]
  command      = ["echo", podman_secret.s.id]
  volumes {
    volume_name    = podman_volume.v.name
    container_path = %[6]q
  }
}
`, cname, secData, image, netMode, env, volPath)
}

func TestAccDepChainFullStackImageChange(t *testing.T) {
	c := randDepName("fs-img")
	runDepRecreateTest(t, depRecreateTest{
		resourceName: "podman_container.test", label: "full stack image", recreate: true,
		pulls:   []string{depImage, depImageAlt},
		configA: fullStack(c, depImage, "/data", "d", "none", "A=1"),
		configB: fullStack(c, depImageAlt, "/data", "d", "none", "A=1"),
	})
}

func TestAccDepChainFullStackVolumePathChange(t *testing.T) {
	c := randDepName("fs-vol")
	runDepRecreateTest(t, depRecreateTest{
		resourceName: "podman_container.test", label: "full stack volume path", recreate: true,
		configA: fullStack(c, depImage, "/data", "d", "none", "A=1"),
		configB: fullStack(c, depImage, "/data2", "d", "none", "A=1"),
	})
}

func TestAccDepChainFullStackSecretDataChange(t *testing.T) {
	c := randDepName("fs-sec")
	runDepRecreateTest(t, depRecreateTest{
		resourceName: "podman_container.test", label: "full stack secret data", recreate: true,
		configA: fullStack(c, depImage, "/data", "d1", "none", "A=1"),
		configB: fullStack(c, depImage, "/data", "d2", "none", "A=1"),
	})
}

func TestAccDepChainFullStackNetworkModeChange(t *testing.T) {
	c := randDepName("fs-net")
	runDepRecreateTest(t, depRecreateTest{
		resourceName: "podman_container.test", label: "full stack network_mode", recreate: true,
		configA: fullStack(c, depImage, "/data", "d", "none", "A=1"),
		configB: fullStack(c, depImage, "/data", "d", "host", "A=1"),
	})
}

func TestAccDepChainFullStackEnvChangeInPlace(t *testing.T) {
	c := randDepName("fs-env")
	runDepRecreateTest(t, depRecreateTest{
		resourceName: "podman_container.test", label: "full stack env", recreate: false,
		configA: fullStack(c, depImage, "/data", "d", "none", "A=1"),
		configB: fullStack(c, depImage, "/data", "d", "none", "A=2"),
	})
}

// --- Diamond: one image resource consumed by two containers ---

func diamondImage(imageName, c1, c2 string) string {
	return fmt.Sprintf(`
resource "podman_image" "base" {
  name         = %[1]q
  keep_locally = true
}

resource "podman_container" "test" {
  name  = %[2]q
  image = podman_image.base.image_id
  start = false
}

resource "podman_container" "other" {
  name  = %[3]q
  image = podman_image.base.image_id
  start = false
}
`, imageName, c1, c2)
}

func TestAccDepChainDiamondImageChangeConsumer1(t *testing.T) {
	c1, c2 := randDepName("dia1a"), randDepName("dia1b")
	runDepRecreateTest(t, depRecreateTest{
		resourceName: "podman_container.test", label: "diamond consumer 1", recreate: true,
		pulls:   []string{depImage, depImageAlt},
		configA: diamondImage(depImage, c1, c2),
		configB: diamondImage(depImageAlt, c1, c2),
	})
}

func TestAccDepChainDiamondImageChangeConsumer2(t *testing.T) {
	c1, c2 := randDepName("dia2a"), randDepName("dia2b")
	runDepRecreateTest(t, depRecreateTest{
		resourceName: "podman_container.other", label: "diamond consumer 2", recreate: true,
		pulls:   []string{depImage, depImageAlt},
		configA: diamondImage(depImage, c1, c2),
		configB: diamondImage(depImageAlt, c1, c2),
	})
}

// --- Container depending on another container via volumes_from ---

func volumesFromPair(aName, bName, aImage string) string {
	return fmt.Sprintf(`
resource "podman_container" "a" {
  name  = %q
  image = %q
  start = false
}

resource "podman_container" "test" {
  name  = %q
  image = %q
  start = false
  volumes {
    from_container = podman_container.a.name
  }
}
`, aName, aImage, bName, depImage)
}

func TestAccDepChainVolumesFromNameChangeRecreatesConsumer(t *testing.T) {
	bName := randDepName("vf-b")
	runDepRecreateTest(t, depRecreateTest{
		resourceName: "podman_container.test", label: "volumes_from consumer on producer name", recreate: true,
		configA: volumesFromPair(randDepName("vf-a1"), bName, depImage),
		configB: volumesFromPair(randDepName("vf-a2"), bName, depImage),
	})
}

func TestAccDepChainVolumesFromNameChangeRecreatesProducer(t *testing.T) {
	bName := randDepName("vfp-b")
	runDepRecreateTest(t, depRecreateTest{
		resourceName: "podman_container.a", label: "volumes_from producer on name change", recreate: true,
		configA: volumesFromPair(randDepName("vfp-a1"), bName, depImage),
		configB: volumesFromPair(randDepName("vfp-a2"), bName, depImage),
	})
}

func TestAccDepChainVolumesFromProducerImageChangeRecreatesProducer(t *testing.T) {
	aName, bName := randDepName("vfi-a"), randDepName("vfi-b")
	runDepRecreateTest(t, depRecreateTest{
		resourceName: "podman_container.a", label: "volumes_from producer on image change", recreate: true,
		pulls:   []string{depImage, depImageAlt},
		configA: volumesFromPair(aName, bName, depImage),
		configB: volumesFromPair(aName, bName, depImageAlt),
	})
}

func TestAccDepChainVolumesFromProducerImageChangeKeepsConsumer(t *testing.T) {
	aName, bName := randDepName("vfk-a"), randDepName("vfk-b")
	runDepRecreateTest(t, depRecreateTest{
		resourceName: "podman_container.test", label: "volumes_from consumer stable on producer image", recreate: false,
		pulls:   []string{depImage, depImageAlt},
		configA: volumesFromPair(aName, bName, depImage),
		configB: volumesFromPair(aName, bName, depImageAlt),
	})
}

// --- Two upstream resources feeding one container ---

func imageVolumeContainer(cname, image, volName, volPath string) string {
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
`, volName, cname, image, volPath)
}

func TestAccDepChainImageVolumeImageChange(t *testing.T) {
	c, v := randDepName("iv-c"), randDepName("iv-v")
	runDepRecreateTest(t, depRecreateTest{
		resourceName: "podman_container.test", label: "image+volume image change", recreate: true,
		pulls:   []string{depImage, depImageAlt},
		configA: imageVolumeContainer(c, depImage, v, "/data"),
		configB: imageVolumeContainer(c, depImageAlt, v, "/data"),
	})
}

func TestAccDepChainImageVolumeVolumePathChange(t *testing.T) {
	c, v := randDepName("ivp-c"), randDepName("ivp-v")
	runDepRecreateTest(t, depRecreateTest{
		resourceName: "podman_container.test", label: "image+volume path change", recreate: true,
		configA: imageVolumeContainer(c, depImage, v, "/data"),
		configB: imageVolumeContainer(c, depImage, v, "/data2"),
	})
}

func TestAccDepChainImageVolumeVolumeNameChange(t *testing.T) {
	c := randDepName("ivn-c")
	runDepRecreateTest(t, depRecreateTest{
		resourceName: "podman_container.test", label: "image+volume name change", recreate: true,
		configA: imageVolumeContainer(c, depImage, randDepName("ivn-v1"), "/data"),
		configB: imageVolumeContainer(c, depImage, randDepName("ivn-v2"), "/data"),
	})
}

// --- Volume + secret feeding one container ---

func volumeSecretContainer(cname, volName, secData, volPath string) string {
	return fmt.Sprintf(`
resource "podman_volume" "v" {
  name = %q
}

resource "podman_secret" "s" {
  name = "%s-sec"
  data = %q
}

resource "podman_container" "test" {
  name    = %q
  image   = %q
  start   = false
  command = ["echo", podman_secret.s.id]
  volumes {
    volume_name    = podman_volume.v.name
    container_path = %q
  }
}
`, volName, cname, secData, cname, depImage, volPath)
}

func TestAccDepChainVolumeSecretVolumePathChange(t *testing.T) {
	c, v := randDepName("vs-c"), randDepName("vs-v")
	runDepRecreateTest(t, depRecreateTest{
		resourceName: "podman_container.test", label: "volume+secret volume change", recreate: true,
		configA: volumeSecretContainer(c, v, "d", "/data"),
		configB: volumeSecretContainer(c, v, "d", "/data2"),
	})
}

func TestAccDepChainVolumeSecretSecretChange(t *testing.T) {
	c, v := randDepName("vss-c"), randDepName("vss-v")
	runDepRecreateTest(t, depRecreateTest{
		resourceName: "podman_container.test", label: "volume+secret secret change", recreate: true,
		configA: volumeSecretContainer(c, v, "d1", "/data"),
		configB: volumeSecretContainer(c, v, "d2", "/data"),
	})
}

// --- Tag + volume feeding one container ---

func tagVolumeContainer(cname, target, volName, volPath string) string {
	return fmt.Sprintf(`
resource "podman_tag" "t" {
  source_image = %q
  target_image = %q
}

resource "podman_volume" "v" {
  name = %q
}

resource "podman_container" "test" {
  name  = %q
  image = podman_tag.t.target_image
  start = false
  volumes {
    volume_name    = podman_volume.v.name
    container_path = %q
  }
}
`, depImage, target, volName, cname, volPath)
}

func TestAccDepChainTagVolumeTagChange(t *testing.T) {
	c, v := randDepName("tv-c"), randDepName("tv-v")
	repo := "localhost/" + randDepName("tv-app")
	runDepRecreateTest(t, depRecreateTest{
		resourceName: "podman_container.test", label: "tag+volume tag change", recreate: true,
		configA: tagVolumeContainer(c, repo+":1", v, "/data"),
		configB: tagVolumeContainer(c, repo+":2", v, "/data"),
	})
}

func TestAccDepChainTagVolumeVolumePathChange(t *testing.T) {
	c, v := randDepName("tvp-c"), randDepName("tvp-v")
	repo := "localhost/" + randDepName("tvp-app")
	runDepRecreateTest(t, depRecreateTest{
		resourceName: "podman_container.test", label: "tag+volume path change", recreate: true,
		configA: tagVolumeContainer(c, repo+":1", v, "/data"),
		configB: tagVolumeContainer(c, repo+":1", v, "/data2"),
	})
}

// --- Multiple volumes on one container ---

func twoVolumeContainer(cname, v1, v2, p1, p2 string) string {
	return fmt.Sprintf(`
resource "podman_volume" "v1" { name = %q }
resource "podman_volume" "v2" { name = %q }

resource "podman_container" "test" {
  name  = %q
  image = %q
  start = false
  volumes {
    volume_name    = podman_volume.v1.name
    container_path = %q
  }
  volumes {
    volume_name    = podman_volume.v2.name
    container_path = %q
  }
}
`, v1, v2, cname, depImage, p1, p2)
}

func TestAccDepChainTwoVolumesOnePathChange(t *testing.T) {
	c := randDepName("2v-c")
	v1, v2 := randDepName("2v-v1"), randDepName("2v-v2")
	runDepRecreateTest(t, depRecreateTest{
		resourceName: "podman_container.test", label: "two volumes one path change", recreate: true,
		configA: twoVolumeContainer(c, v1, v2, "/a", "/b"),
		configB: twoVolumeContainer(c, v1, v2, "/a", "/b2"),
	})
}

func TestAccDepChainTwoVolumesNameChange(t *testing.T) {
	c := randDepName("2vn-c")
	v1 := randDepName("2vn-v1")
	runDepRecreateTest(t, depRecreateTest{
		resourceName: "podman_container.test", label: "two volumes name change", recreate: true,
		configA: twoVolumeContainer(c, v1, randDepName("2vn-v2a"), "/a", "/b"),
		configB: twoVolumeContainer(c, v1, randDepName("2vn-v2b"), "/a", "/b"),
	})
}

// --- Two secrets referenced in one container's command ---

func twoSecretContainer(cname, d1, d2 string) string {
	return fmt.Sprintf(`
resource "podman_secret" "s1" {
  name = "%[1]s-s1"
  data = %[2]q
}

resource "podman_secret" "s2" {
  name = "%[1]s-s2"
  data = %[3]q
}

resource "podman_container" "test" {
  name    = %[1]q
  image   = %[4]q
  start   = false
  command = ["echo", podman_secret.s1.id, podman_secret.s2.id]
}
`, cname, d1, d2, depImage)
}

func TestAccDepChainTwoSecretsOneChange(t *testing.T) {
	c := randDepName("2s-c")
	runDepRecreateTest(t, depRecreateTest{
		resourceName: "podman_container.test", label: "two secrets one change", recreate: true,
		configA: twoSecretContainer(c, "a", "x"),
		configB: twoSecretContainer(c, "a", "y"),
	})
}

func TestAccDepChainTwoSecretsBothStableInPlaceEnv(t *testing.T) {
	c := randDepName("2se-c")
	cfg := func(env string) string {
		return fmt.Sprintf(`
resource "podman_secret" "s1" {
  name = "%[1]s-s1"
  data = "a"
}

resource "podman_secret" "s2" {
  name = "%[1]s-s2"
  data = "b"
}

resource "podman_container" "test" {
  name    = %[1]q
  image   = %[2]q
  start   = false
  command = ["echo", podman_secret.s1.id, podman_secret.s2.id]
  env     = [%[3]q]
}
`, c, depImage, env)
	}
	runDepRecreateTest(t, depRecreateTest{
		resourceName: "podman_container.test", label: "two secrets env in place", recreate: false,
		configA: cfg("K=1"), configB: cfg("K=2"),
	})
}

// --- Image -> Tag -> Container plus a secret (three-resource chain) ---

func imageTagSecretContainer(cname, target, secData string) string {
	return fmt.Sprintf(`
resource "podman_image" "base" {
  name         = %[1]q
  keep_locally = true
}

resource "podman_tag" "t" {
  source_image = podman_image.base.name
  target_image = %[2]q
}

resource "podman_secret" "s" {
  name = "%[3]s-sec"
  data = %[4]q
}

resource "podman_container" "test" {
  name    = %[3]q
  image   = podman_tag.t.target_image
  start   = false
  command = ["echo", podman_secret.s.id]
}
`, depImage, target, cname, secData)
}

func TestAccDepChainImageTagSecretTagChange(t *testing.T) {
	c := randDepName("its-t")
	repo := "localhost/" + randDepName("its-app")
	runDepRecreateTest(t, depRecreateTest{
		resourceName: "podman_container.test", label: "image-tag-secret tag change", recreate: true,
		pulls:   []string{depImage},
		configA: imageTagSecretContainer(c, repo+":1", "d"),
		configB: imageTagSecretContainer(c, repo+":2", "d"),
	})
}

func TestAccDepChainImageTagSecretSecretChange(t *testing.T) {
	c := randDepName("itss-t")
	repo := "localhost/" + randDepName("itss-app")
	runDepRecreateTest(t, depRecreateTest{
		resourceName: "podman_container.test", label: "image-tag-secret secret change", recreate: true,
		pulls:   []string{depImage},
		configA: imageTagSecretContainer(c, repo+":1", "d1"),
		configB: imageTagSecretContainer(c, repo+":1", "d2"),
	})
}

// --- Network + volume on one container ---

func networkVolumeContainer(cname, netName, volName, netMode string) string {
	return fmt.Sprintf(`
resource "podman_network" "n" {
  name = %q
}

resource "podman_volume" "v" {
  name = %q
}

resource "podman_container" "test" {
  name         = %q
  image        = %q
  start        = false
  network_mode = %q
  volumes {
    volume_name    = podman_volume.v.name
    container_path = "/data"
  }
}
`, netName, volName, cname, depImage, netMode)
}

func TestAccDepChainNetworkVolumeNetworkModeChange(t *testing.T) {
	c := randDepName("nv-c")
	n, v := randDepName("nv-n"), randDepName("nv-v")
	runDepRecreateTest(t, depRecreateTest{
		resourceName: "podman_container.test", label: "network+volume network_mode", recreate: true,
		configA: networkVolumeContainer(c, n, v, "none"),
		configB: networkVolumeContainer(c, n, v, "host"),
	})
}

// --- Container joined to two networks; changing one alias is in place ---

func twoNetworkContainer(cname, n1, n2, alias2 string) string {
	return fmt.Sprintf(`
resource "podman_network" "n1" { name = %q }
resource "podman_network" "n2" { name = %q }

resource "podman_container" "test" {
  name  = %q
  image = %q
  start = false
  networks_advanced {
    name = podman_network.n1.name
  }
  networks_advanced {
    name    = podman_network.n2.name
    aliases = [%q]
  }
}
`, n1, n2, cname, depImage, alias2)
}

func TestAccDepChainTwoNetworksAliasChangeInPlace(t *testing.T) {
	c := randDepName("2n-c")
	n1, n2 := randDepName("2n-n1"), randDepName("2n-n2")
	runDepRecreateTest(t, depRecreateTest{
		resourceName: "podman_container.test", label: "two networks alias in place", recreate: false,
		configA: twoNetworkContainer(c, n1, n2, "alpha"),
		configB: twoNetworkContainer(c, n1, n2, "beta"),
	})
}

// --- Full stack: volume swap and label-in-place variants ---

func TestAccDepChainFullStackLabelChangeInPlace(t *testing.T) {
	c := randDepName("fsl")
	cfg := func(val string) string {
		return fmt.Sprintf(`
resource "podman_volume" "v" { name = "%[1]s-vol" }

resource "podman_container" "test" {
  name  = %[1]q
  image = %[2]q
  start = false
  labels {
    label = "tier"
    value = %[3]q
  }
  volumes {
    volume_name    = podman_volume.v.name
    container_path = "/data"
  }
}
`, c, depImage, val)
	}
	runDepRecreateTest(t, depRecreateTest{
		resourceName: "podman_container.test", label: "full stack label in place", recreate: false,
		configA: cfg("a"), configB: cfg("b"),
	})
}

// --- Three-container volumes_from chain: A <- B <- C ---

func TestAccDepChainThreeContainerVolumesFrom(t *testing.T) {
	b, c := randDepName("3c-b"), randDepName("3c-c")
	cfg := func(aName string) string {
		return fmt.Sprintf(`
resource "podman_container" "a" {
  name  = %q
  image = %q
  start = false
}

resource "podman_container" "b" {
  name  = %q
  image = %q
  start = false
  volumes {
    from_container = podman_container.a.name
  }
}

resource "podman_container" "test" {
  name  = %q
  image = %q
  start = false
  volumes {
    from_container = podman_container.b.name
  }
}
`, aName, depImage, b, depImage, c, depImage)
	}
	runDepRecreateTest(t, depRecreateTest{
		resourceName: "podman_container.b", label: "three-container volumes_from", recreate: true,
		configA: cfg(randDepName("3c-a1")),
		configB: cfg(randDepName("3c-a2")),
	})
}

// --- Network + volume + image, image change ---

func TestAccDepChainNetworkVolumeImageChange(t *testing.T) {
	c := randDepName("nvi-c")
	n, v := randDepName("nvi-n"), randDepName("nvi-v")
	cfg := func(image string) string {
		return fmt.Sprintf(`
resource "podman_network" "n" { name = %q }
resource "podman_volume" "v" { name = %q }

resource "podman_container" "test" {
  name  = %q
  image = %q
  start = false
  networks_advanced {
    name = podman_network.n.name
  }
  volumes {
    volume_name    = podman_volume.v.name
    container_path = "/data"
  }
}
`, n, v, c, image)
	}
	runDepRecreateTest(t, depRecreateTest{
		resourceName: "podman_container.test", label: "network+volume+image image change", recreate: true,
		pulls:   []string{depImage, depImageAlt},
		configA: cfg(depImage),
		configB: cfg(depImageAlt),
	})
}

func TestAccDepChainNetworkVolumeNetworkModeWithImage(t *testing.T) {
	c := randDepName("nvm-c")
	n, v := randDepName("nvm-n"), randDepName("nvm-v")
	cfg := func(netMode string) string {
		return fmt.Sprintf(`
resource "podman_network" "n" { name = %q }
resource "podman_volume" "v" { name = %q }

resource "podman_container" "test" {
  name         = %q
  image        = %q
  start        = false
  network_mode = %q
  volumes {
    volume_name    = podman_volume.v.name
    container_path = "/data"
  }
}
`, n, v, c, depImage, netMode)
	}
	runDepRecreateTest(t, depRecreateTest{
		resourceName: "podman_container.test", label: "network+volume network_mode (image)", recreate: true,
		configA: cfg("none"),
		configB: cfg("host"),
	})
}

// --- Secret + network on one container, secret-driven command change ---

func TestAccDepChainSecretNetworkSecretChange(t *testing.T) {
	c := randDepName("sn-c")
	n := randDepName("sn-n")
	cfg := func(data string) string {
		return fmt.Sprintf(`
resource "podman_network" "n" { name = %q }

resource "podman_secret" "s" {
  name = "%s-sec"
  data = %q
}

resource "podman_container" "test" {
  name    = %q
  image   = %q
  start   = false
  command = ["echo", podman_secret.s.id]
  networks_advanced {
    name = podman_network.n.name
  }
}
`, n, c, data, c, depImage)
	}
	runDepRecreateTest(t, depRecreateTest{
		resourceName: "podman_container.test", label: "secret+network secret change", recreate: true,
		configA: cfg("d1"), configB: cfg("d2"),
	})
}

// --- Kitchen-sink volume swap between two named volumes ---

func TestAccDepChainFullStackVolumeSwap(t *testing.T) {
	c := randDepName("fsswap")
	cfg := func(which string) string {
		return fmt.Sprintf(`
resource "podman_volume" "v1" { name = "%[1]s-v1" }
resource "podman_volume" "v2" { name = "%[1]s-v2" }

resource "podman_secret" "s" {
  name = "%[1]s-sec"
  data = "d"
}

resource "podman_container" "test" {
  name    = %[1]q
  image   = %[2]q
  start   = false
  command = ["echo", podman_secret.s.id]
  volumes {
    volume_name    = podman_volume.%[3]s.name
    container_path = "/data"
  }
}
`, c, depImage, which)
	}
	runDepRecreateTest(t, depRecreateTest{
		resourceName: "podman_container.test", label: "full stack volume swap", recreate: true,
		configA: cfg("v1"), configB: cfg("v2"),
	})
}
