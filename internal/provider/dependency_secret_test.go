package provider

import (
	"fmt"
	"testing"
)

// podman_secret uses a fresh random podman ID per creation, so any ForceNew
// change (name, data) is observable as an ID change. Containers cannot mount
// podman secrets natively, so the dependency edge is expressed by referencing
// the secret's name/id from a container attribute.

func hclSecret(name, data string) string {
	return fmt.Sprintf(`resource "podman_secret" "s" {
  name = %q
  data = %q
}`, name, data)
}

// --- Secret resource recreation ---

func TestAccDepSecretNameChangeRecreatesSecret(t *testing.T) {
	runDepRecreateTest(t, depRecreateTest{
		resourceName: "podman_secret.s", label: "secret name", recreate: true,
		configA: hclSecret(randDepName("sec-a"), "topsecret"),
		configB: hclSecret(randDepName("sec-b"), "topsecret"),
	})
}

func TestAccDepSecretDataChangeRecreatesSecret(t *testing.T) {
	n := randDepName("secdata")
	runDepRecreateTest(t, depRecreateTest{
		resourceName: "podman_secret.s", label: "secret data", recreate: true,
		configA: hclSecret(n, "value-one"),
		configB: hclSecret(n, "value-two"),
	})
}

func TestAccDepSecretAddLabelRecreatesSecret(t *testing.T) {
	n := randDepName("seclbl")
	runDepRecreateTest(t, depRecreateTest{
		resourceName: "podman_secret.s", label: "secret add label", recreate: true,
		configA: hclSecret(n, "data"),
		configB: fmt.Sprintf("resource \"podman_secret\" \"s\" {\n  name = %q\n  data = \"data\"\n  labels {\n    label = \"tier\"\n    value = \"db\"\n  }\n}", n),
	})
}

// --- Secret identity feeding a ForceNew container attribute (command) ---

func hclSecretCmdContainer(secName, secData, cname string, cmd string) string {
	return fmt.Sprintf(`
%s

resource "podman_container" "test" {
  name    = %q
  image   = %q
  start   = false
  command = %s
}
`, hclSecret(secName, secData), cname, depImage, cmd)
}

func TestAccDepSecretDataChangeRecreatesDependentContainer(t *testing.T) {
	sec := randDepName("scmd")
	cname := randDepName("scmdc")
	cfg := func(data string) string {
		return hclSecretCmdContainer(sec, data, cname, `["echo", podman_secret.s.id]`)
	}
	runDepRecreateTest(t, depRecreateTest{
		resourceName: "podman_container.test", label: "container on secret data via command", recreate: true,
		configA: cfg("d1"), configB: cfg("d2"),
	})
}

func TestAccDepSecretNameChangeRecreatesDependentContainer(t *testing.T) {
	cname := randDepName("snmc")
	cfg := func(secName string) string {
		return hclSecretCmdContainer(secName, "data", cname, `["echo", podman_secret.s.name]`)
	}
	runDepRecreateTest(t, depRecreateTest{
		resourceName: "podman_container.test", label: "container on secret name via command", recreate: true,
		configA: cfg(randDepName("snm-a")), configB: cfg(randDepName("snm-b")),
	})
}

// --- Secret identity feeding an in-place container attribute (env/label) ---

func TestAccDepSecretNameChangeContainerEnvInPlace(t *testing.T) {
	cname := randDepName("senv")
	cfg := func(secName string) string {
		return fmt.Sprintf(`
%s

resource "podman_container" "test" {
  name  = %q
  image = %q
  start = false
  env   = ["SECRET_NAME=${podman_secret.s.name}"]
}
`, hclSecret(secName, "data"), cname, depImage)
	}
	runDepRecreateTest(t, depRecreateTest{
		resourceName: "podman_container.test", label: "container env on secret name", recreate: false,
		configA: cfg(randDepName("senv-a")), configB: cfg(randDepName("senv-b")),
	})
}

func TestAccDepSecretDataChangeContainerLabelInPlace(t *testing.T) {
	sec := randDepName("slbl")
	cname := randDepName("slblc")
	cfg := func(data string) string {
		return fmt.Sprintf(`
%s

resource "podman_container" "test" {
  name  = %q
  image = %q
  start = false
  labels {
    label = "secret-id"
    value = podman_secret.s.id
  }
}
`, hclSecret(sec, data), cname, depImage)
	}
	runDepRecreateTest(t, depRecreateTest{
		resourceName: "podman_container.test", label: "container label on secret id", recreate: false,
		configA: cfg("d1"), configB: cfg("d2"),
	})
}

// --- One secret consumed by two containers ---

func TestAccDepSecretSharedByTwoContainers(t *testing.T) {
	sec := randDepName("sshare")
	c1 := randDepName("sshare1")
	c2 := randDepName("sshare2")
	cfg := func(data string) string {
		return fmt.Sprintf(`
%s

resource "podman_container" "test" {
  name    = %q
  image   = %q
  start   = false
  command = ["echo", podman_secret.s.id]
}

resource "podman_container" "other" {
  name    = %q
  image   = %q
  start   = false
  command = ["echo", podman_secret.s.id]
}
`, hclSecret(sec, data), c1, depImage, c2, depImage)
	}
	runDepRecreateTest(t, depRecreateTest{
		resourceName: "podman_container.test", label: "shared secret consumer", recreate: true,
		configA: cfg("d1"), configB: cfg("d2"),
	})
}
