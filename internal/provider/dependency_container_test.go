package provider

import (
	"fmt"
	"testing"
)

// hclC builds a single not-started container using the workhorse image, with
// the supplied extra body lines. Used by the container-attribute tests below.
func hclC(name, body string) string {
	return fmt.Sprintf(`
resource "podman_container" "test" {
  name  = %q
  image = %q
  start = false
  %s
}
`, name, depImage, body)
}

// hclCS is like hclC but starts the container, so runtime-derived fields such as
// published ports and extra hosts are fully realized and round-trip through Read.
func hclCS(name, body string) string {
	return fmt.Sprintf(`
resource "podman_container" "test" {
  name    = %q
  image   = %q
  start   = true
  command = ["sleep", "300"]
  %s
}
`, name, depImage, body)
}

// --- ForceNew attribute changes: container must be recreated ---

func TestAccDepContainerCommandChangeRecreates(t *testing.T) {
	n := randDepName("cmd")
	runDepRecreateTest(t, depRecreateTest{
		resourceName: "podman_container.test", label: "command", recreate: true,
		configA: hclC(n, `command = ["sleep", "infinity"]`),
		configB: hclC(n, `command = ["sleep", "120"]`),
	})
}

func TestAccDepContainerEntrypointChangeRecreates(t *testing.T) {
	n := randDepName("entry")
	runDepRecreateTest(t, depRecreateTest{
		resourceName: "podman_container.test", label: "entrypoint", recreate: true,
		configA: hclC(n, `entrypoint = ["/bin/sh", "-c", "sleep infinity"]`),
		configB: hclC(n, `entrypoint = ["/bin/sh", "-c", "sleep 100"]`),
	})
}

func TestAccDepContainerHostnameChangeRecreates(t *testing.T) {
	n := randDepName("host")
	runDepRecreateTest(t, depRecreateTest{
		resourceName: "podman_container.test", label: "hostname", recreate: true,
		configA: hclC(n, `hostname = "alpha"`),
		configB: hclC(n, `hostname = "beta"`),
	})
}

func TestAccDepContainerUserChangeRecreates(t *testing.T) {
	n := randDepName("user")
	runDepRecreateTest(t, depRecreateTest{
		resourceName: "podman_container.test", label: "user", recreate: true,
		configA: hclC(n, `user = "405"`),
		configB: hclC(n, `user = "100"`),
	})
}

func TestAccDepContainerDomainnameChangeRecreates(t *testing.T) {
	n := randDepName("domain")
	runDepRecreateTest(t, depRecreateTest{
		resourceName: "podman_container.test", label: "domainname", recreate: true,
		configA: hclC(n, `domainname = "a.example.com"`),
		configB: hclC(n, `domainname = "b.example.com"`),
	})
}

func TestAccDepContainerPrivilegedChangeRecreates(t *testing.T) {
	n := randDepName("priv")
	runDepRecreateTest(t, depRecreateTest{
		resourceName: "podman_container.test", label: "privileged", recreate: true,
		configA: hclC(n, `privileged = false`),
		configB: hclC(n, `privileged = true`),
	})
}

func TestAccDepContainerWorkingDirChangeRecreates(t *testing.T) {
	n := randDepName("workdir")
	runDepRecreateTest(t, depRecreateTest{
		resourceName: "podman_container.test", label: "working_dir", recreate: true,
		configA: hclC(n, `working_dir = "/tmp"`),
		configB: hclC(n, `working_dir = "/root"`),
	})
}

func TestAccDepContainerPortChangeRecreates(t *testing.T) {
	n := randDepName("port")
	runDepRecreateTest(t, depRecreateTest{
		resourceName: "podman_container.test", label: "ports", recreate: true,
		configA: hclCS(n, "ports {\n    internal = 80\n  }"),
		configB: hclCS(n, "ports {\n    internal = 81\n  }"),
	})
}

func TestAccDepContainerVolumeHostPathChangeRecreates(t *testing.T) {
	n := randDepName("vol")
	runDepRecreateTest(t, depRecreateTest{
		resourceName: "podman_container.test", label: "volumes", recreate: true,
		configA: hclCS(n, "volumes {\n    host_path      = \"/tmp/tf-dep-a\"\n    container_path = \"/data\"\n  }"),
		configB: hclCS(n, "volumes {\n    host_path      = \"/tmp/tf-dep-b\"\n    container_path = \"/data\"\n  }"),
	})
}

func TestAccDepContainerDnsChangeRecreates(t *testing.T) {
	n := randDepName("dns")
	runDepRecreateTest(t, depRecreateTest{
		resourceName: "podman_container.test", label: "dns", recreate: true,
		configA: hclC(n, `dns = ["1.1.1.1"]`),
		configB: hclC(n, `dns = ["8.8.8.8"]`),
	})
}

func TestAccDepContainerDnsSearchChangeRecreates(t *testing.T) {
	n := randDepName("dnss")
	runDepRecreateTest(t, depRecreateTest{
		resourceName: "podman_container.test", label: "dns_search", recreate: true,
		configA: hclC(n, `dns_search = ["a.local"]`),
		configB: hclC(n, `dns_search = ["b.local"]`),
	})
}

func TestAccDepContainerHostEntryChangeRecreates(t *testing.T) {
	n := randDepName("hosts")
	runDepRecreateTest(t, depRecreateTest{
		resourceName: "podman_container.test", label: "host", recreate: true,
		configA: hclCS(n, "host {\n    host = \"svc.local\"\n    ip   = \"10.0.0.1\"\n  }"),
		configB: hclCS(n, "host {\n    host = \"svc.local\"\n    ip   = \"10.0.0.2\"\n  }"),
	})
}

func TestAccDepContainerCapabilitiesChangeRecreates(t *testing.T) {
	n := randDepName("cap")
	runDepRecreateTest(t, depRecreateTest{
		resourceName: "podman_container.test", label: "capabilities", recreate: true,
		configA: hclCS(n, "capabilities {\n    add = [\"NET_ADMIN\"]\n  }"),
		configB: hclCS(n, "capabilities {\n    add = [\"SYS_TIME\"]\n  }"),
	})
}

func TestAccDepContainerNameChangeRecreates(t *testing.T) {
	a := randDepName("rename-a")
	b := randDepName("rename-b")
	runDepRecreateTest(t, depRecreateTest{
		resourceName: "podman_container.test", label: "name", recreate: true,
		configA: hclC(a, ""),
		configB: hclC(b, ""),
	})
}

// --- In-place updates: container must NOT be recreated ---

func TestAccDepContainerEnvChangeInPlace2(t *testing.T) {
	n := randDepName("env2")
	runDepRecreateTest(t, depRecreateTest{
		resourceName: "podman_container.test", label: "env", recreate: false,
		configA: hclC(n, `env = ["FOO=a"]`),
		configB: hclC(n, `env = ["FOO=b"]`),
	})
}

func TestAccDepContainerLabelChangeInPlace(t *testing.T) {
	n := randDepName("label")
	runDepRecreateTest(t, depRecreateTest{
		resourceName: "podman_container.test", label: "labels", recreate: false,
		configA: hclC(n, "labels {\n    label = \"role\"\n    value = \"a\"\n  }"),
		configB: hclC(n, "labels {\n    label = \"role\"\n    value = \"b\"\n  }"),
	})
}

func TestAccDepContainerRestartChangeInPlace(t *testing.T) {
	n := randDepName("restart")
	runDepRecreateTest(t, depRecreateTest{
		resourceName: "podman_container.test", label: "restart", recreate: false,
		configA: hclC(n, `restart = "no"`),
		configB: hclC(n, `restart = "on-failure"`),
	})
}

func TestAccDepContainerMemoryChangeInPlace(t *testing.T) {
	n := randDepName("mem")
	runDepRecreateTest(t, depRecreateTest{
		resourceName: "podman_container.test", label: "memory", recreate: false,
		configA: hclC(n, `memory = 67108864`),
		configB: hclC(n, `memory = 134217728`),
	})
}

func TestAccDepContainerCpuSharesChangeInPlace(t *testing.T) {
	n := randDepName("cpu")
	runDepRecreateTest(t, depRecreateTest{
		resourceName: "podman_container.test", label: "cpu_shares", recreate: false,
		configA: hclC(n, `cpu_shares = 512`),
		configB: hclC(n, `cpu_shares = 1024`),
	})
}
