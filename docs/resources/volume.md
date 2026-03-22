---
page_title: "podman_volume Resource - terraform-provider-podman"
subcategory: ""
description: |-
  Manages the lifecycle of a Podman volume.
---

# podman_volume (Resource)

Manages the lifecycle of a Podman volume.

## Example Usage

### Basic Named Volume

```terraform
resource "podman_volume" "data" {
  name = "app_data"
}
```

### Volume with Driver Options

```terraform
resource "podman_volume" "custom" {
  name   = "custom_volume"
  driver = "local"

  driver_opts = {
    type   = "tmpfs"
    device = "tmpfs"
    o      = "size=100m"
  }

  labels {
    label = "environment"
    value = "development"
  }
}
```

## Argument Reference

The following arguments are supported:

- `name` - (Optional, Computed, String) The name of the volume. If omitted, a name is automatically generated. Changing this forces recreation of the resource.
- `driver` - (Optional, Computed, String) The driver to use for the volume. Changing this forces recreation of the resource.
- `driver_opts` - (Optional, Map of String) Driver-specific options. Changing this forces recreation of the resource.
- `labels` - (Optional, Block Set) A set of labels to apply to the volume. Changing this forces recreation of the resource. See [labels](#labels) below for details.

### labels

- `label` - (Required, String) The name of the label.
- `value` - (Required, String) The value of the label.

## Attribute Reference

In addition to all arguments above, the following attributes are exported:

- `id` - (String) The ID of the volume.
- `mountpoint` - (String) The filesystem path where the volume is mounted.

## Import

Podman volumes can be imported using the volume name:

```shell
terraform import podman_volume.data app_data
```
