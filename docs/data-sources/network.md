---
page_title: "podman_network Data Source - Podman"
subcategory: ""
description: |-
  Reads information about an existing Podman network.
---

# podman_network (Data Source)

Reads information about an existing Podman network. This data source can be used to retrieve details such as the driver, IPAM configuration, and scope of a network managed by Podman.

## Example Usage

```hcl
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
```

## Argument Reference

- `name` (String, Required) - The name of the network.

## Attribute Reference

In addition to all arguments above, the following attributes are exported:

- `id` (String) - The network ID.
- `driver` (String) - The network driver (e.g., `bridge`).
- `internal` (Boolean) - Whether the network is internal (no external connectivity).
- `scope` (String) - The scope of the network (e.g., `local`).
- `options` (Map of String) - Driver-specific options for the network.
- `ipam_config` (List of Object) - IP Address Management configuration for the network. Each object contains the following attributes:
  - `aux_address` (Map of String) - Auxiliary IPv4 or IPv6 addresses used by the network driver.
  - `gateway` (String) - The gateway address for the subnet.
  - `ip_range` (String) - The range of IP addresses from which container IPs are allocated.
  - `subnet` (String) - The subnet in CIDR format.
