# Basic bridge network
resource "podman_network" "bridge" {
  name = "my-bridge-network"
}

# Network with IPAM configuration
resource "podman_network" "ipam" {
  name   = "my-ipam-network"
  driver = "bridge"

  ipam_config {
    subnet  = "172.20.0.0/16"
    gateway = "172.20.0.1"
  }

  labels {
    label = "environment"
    value = "production"
  }
}

# Internal network with IPv6
resource "podman_network" "internal_ipv6" {
  name     = "my-internal-network"
  internal = true
  ipv6     = true

  ipam_config {
    subnet  = "fd00::/64"
    gateway = "fd00::1"
  }
}
