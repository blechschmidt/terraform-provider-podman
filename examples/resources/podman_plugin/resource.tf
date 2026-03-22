resource "podman_plugin" "sample" {
  name                  = "docker.io/library/sample-plugin:latest"
  alias                 = "sample"
  enabled               = true
  grant_all_permissions = true

  env = [
    "DEBUG=1",
  ]
}
