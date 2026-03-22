# Basic provider configuration using rootless Podman socket
provider "podman" {
  host = "unix:///run/user/1000/podman/podman.sock"
}

# Provider configuration with registry authentication
provider "podman" {
  host = "unix:///run/user/1000/podman/podman.sock"

  registry_auth {
    address  = "registry.example.com"
    username = "user"
    password = "pass"
  }
}
