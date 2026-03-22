---
page_title: "podman_network Resource - terraform-provider-podman"
subcategory: ""
description: |-
  Manages the lifecycle of a Podman network.
---

# podman_network (Resource)

Manages the lifecycle of a Podman network.

## Example Usage

### Basic Bridge Network

```terraform
resource "podman_network" "bridge" {
  name = "my-bridge-network"
}
```

### Network with IPAM Configuration

```terraform
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
```

### Internal Network with IPv6

```terraform
resource "podman_network" "internal_ipv6" {
  name     = "my-internal-network"
  internal = true
  ipv6     = true

  ipam_config {
    subnet  = "fd00::/64"
    gateway = "fd00::1"
  }
}
```

## Argument Reference

The following arguments are supported:

- `name` - (Required, String) The name of the network.
- `attachable` - (Optional, Boolean) Whether the network is manually attachable.
- `driver` - (Optional, String) The network driver to use. Default: `bridge`. Changing this forces recreation of the resource.
- `ingress` - (Optional, Boolean) Whether this is an ingress network. Changing this forces recreation of the resource.
- `internal` - (Optional, Boolean) Whether the network is internal-only (no external connectivity). Changing this forces recreation of the resource.
- `ipam_config` - (Optional, Block List) IPAM configuration blocks. See [ipam_config](#ipam_config) below for details.
- `ipam_driver` - (Optional, String) The IPAM driver to use. Default: `default`. Changing this forces recreation of the resource.
- `ipam_options` - (Optional, Map of String) Driver-specific options for the IPAM driver.
- `ipv6` - (Optional, Boolean) Whether to enable IPv6 networking. Changing this forces recreation of the resource.
- `labels` - (Optional, Block Set) A set of labels to apply to the network. See [labels](#labels) below for details.
- `options` - (Optional, Computed, Map of String) Driver-specific options.

### ipam_config

- `aux_address` - (Optional, Map of String) Auxiliary IPv4 or IPv6 addresses used by the network driver.
- `gateway` - (Optional, String) The gateway address for the subnet.
- `ip_range` - (Optional, String) The range of IPs from which to allocate container IPs.
- `subnet` - (Optional, String) The subnet in CIDR format that represents the network segment.

### labels

- `label` - (Required, String) The name of the label.
- `value` - (Required, String) The value of the label.

## Attribute Reference

In addition to all arguments above, the following attributes are exported:

- `id` - (String) The ID of the network.
- `scope` - (String) The scope of the network.

## Import

Podman networks can be imported using the network name:

```shell
terraform import podman_network.bridge my-bridge-network
```
