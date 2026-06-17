package provider

import (
	"fmt"
	"testing"
)

// podman_network uses a fresh random podman ID per creation, so a ForceNew
// change (name, internal, ...) is observable as an ID change. Containers attach
// to a network through the in-place networks_advanced block or the ForceNew
// network_mode scalar.

func hclNetwork(res, name string, body string) string {
	return fmt.Sprintf(`resource "podman_network" %q {
  name = %q
  %s
}`, res, name, body)
}

// --- Network resource recreation ---

func TestAccDepNetworkNameChangeRecreatesNetwork(t *testing.T) {
	runDepRecreateTest(t, depRecreateTest{
		resourceName: "podman_network.n", label: "network name", recreate: true,
		configA: hclNetwork("n", randDepName("net-a"), ""),
		configB: hclNetwork("n", randDepName("net-b"), ""),
	})
}

func TestAccDepNetworkInternalChangeRecreatesNetwork(t *testing.T) {
	n := randDepName("netint")
	runDepRecreateTest(t, depRecreateTest{
		resourceName: "podman_network.n", label: "network internal", recreate: true,
		configA: hclNetwork("n", n, "internal = false"),
		configB: hclNetwork("n", n, "internal = true"),
	})
}

func TestAccDepNetworkDriverBridgeNameChangeRecreatesNetwork(t *testing.T) {
	runDepRecreateTest(t, depRecreateTest{
		resourceName: "podman_network.n", label: "bridge network name", recreate: true,
		configA: hclNetwork("n", randDepName("netbr-a"), `driver = "bridge"`),
		configB: hclNetwork("n", randDepName("netbr-b"), `driver = "bridge"`),
	})
}

func TestAccDepNetworkInternalTrueNameChangeRecreatesNetwork(t *testing.T) {
	runDepRecreateTest(t, depRecreateTest{
		resourceName: "podman_network.n", label: "internal network name", recreate: true,
		configA: hclNetwork("n", randDepName("neti-a"), "internal = true"),
		configB: hclNetwork("n", randDepName("neti-b"), "internal = true"),
	})
}

// --- Container network_mode (ForceNew scalar) ---

func TestAccDepContainerNetworkModeNoneRecreates(t *testing.T) {
	n := randDepName("nmode")
	runDepRecreateTest(t, depRecreateTest{
		resourceName: "podman_container.test", label: "network_mode", recreate: true,
		configA: hclC(n, `network_mode = "bridge"`),
		configB: hclC(n, `network_mode = "none"`),
	})
}

func TestAccDepContainerNetworkModeHostRecreates(t *testing.T) {
	n := randDepName("nmodeh")
	runDepRecreateTest(t, depRecreateTest{
		resourceName: "podman_container.test", label: "network_mode host", recreate: true,
		configA: hclC(n, `network_mode = "none"`),
		configB: hclC(n, `network_mode = "host"`),
	})
}

// --- Container attaching to a network via networks_advanced (in-place) ---

func hclNetContainer(net, netName, cname, naBody string) string {
	return fmt.Sprintf(`
resource "podman_network" "n" {
  name = %q
}

resource "podman_container" "test" {
  name  = %q
  image = %q
  start = false
  networks_advanced {
    name = podman_network.n.name
    %s
  }
}
`, netName, cname, depImage, naBody)
}

func TestAccDepContainerNetworkAliasChangeInPlace(t *testing.T) {
	net := randDepName("nalias")
	cname := randDepName("naliasc")
	runDepRecreateTest(t, depRecreateTest{
		resourceName: "podman_container.test", label: "network alias", recreate: false,
		configA: hclNetContainer(net, net, cname, `aliases = ["web1"]`),
		configB: hclNetContainer(net, net, cname, `aliases = ["web2"]`),
	})
}

func TestAccDepContainerJoinNetworkInPlace(t *testing.T) {
	net := randDepName("njoin")
	cname := randDepName("njoinc")
	without := fmt.Sprintf(`
resource "podman_network" "n" {
  name = %q
}

resource "podman_container" "test" {
  name  = %q
  image = %q
  start = false
}
`, net, cname, depImage)
	runDepRecreateTest(t, depRecreateTest{
		resourceName: "podman_container.test", label: "join network", recreate: false,
		configA: without,
		configB: hclNetContainer(net, net, cname, `aliases = ["app"]`),
	})
}

// --- Network identity change with a dependent container attached ---

func TestAccDepNetworkNameChangeRecreatesNetworkWithConsumer(t *testing.T) {
	cname := randDepName("nconsc")
	cfg := func(netName string) string { return hclNetContainer(netName, netName, cname, `aliases = ["svc"]`) }
	runDepRecreateTest(t, depRecreateTest{
		resourceName: "podman_network.n", label: "network with consumer", recreate: true,
		configA: cfg(randDepName("ncons-a")),
		configB: cfg(randDepName("ncons-b")),
	})
}

// --- Two containers attached to one network ---

func TestAccDepNetworkSharedByTwoContainers(t *testing.T) {
	c1 := randDepName("nshare1")
	c2 := randDepName("nshare2")
	cfg := func(netName string) string {
		return fmt.Sprintf(`
resource "podman_network" "n" {
  name = %q
}

resource "podman_container" "test" {
  name  = %q
  image = %q
  start = false
  networks_advanced {
    name = podman_network.n.name
  }
}

resource "podman_container" "other" {
  name  = %q
  image = %q
  start = false
  networks_advanced {
    name = podman_network.n.name
  }
}
`, netName, c1, depImage, c2, depImage)
	}
	runDepRecreateTest(t, depRecreateTest{
		resourceName: "podman_network.n", label: "network shared", recreate: true,
		configA: cfg(randDepName("nshare-a")),
		configB: cfg(randDepName("nshare-b")),
	})
}
