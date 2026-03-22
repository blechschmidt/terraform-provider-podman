# Look up an existing network by name
data "podman_network" "my_network" {
  name = "my-bridge-network"
}

# Use network attributes
output "network_driver" {
  value = data.podman_network.my_network.driver
}

output "network_subnet" {
  value = data.podman_network.my_network.ipam_config[0].subnet
}
