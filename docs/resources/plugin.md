---
page_title: "podman_plugin Resource - terraform-provider-podman"
subcategory: ""
description: |-
  Manages the lifecycle of a Podman plugin.
---

# podman_plugin (Resource)

Manages the lifecycle of a Podman plugin.

## Example Usage

```terraform
resource "podman_plugin" "sample" {
  name                  = "docker.io/library/sample-plugin:latest"
  alias                 = "sample"
  enabled               = true
  grant_all_permissions = true

  env = [
    "DEBUG=1",
  ]
}
```

## Argument Reference

The following arguments are supported:

- `name` - (Required, String) The plugin name or reference to install. Changing this forces recreation of the resource.
- `alias` - (Optional, Computed, String) An alias for the plugin. Changing this forces recreation of the resource.
- `enable_timeout` - (Optional, Number) The timeout in seconds for the enable operation. Default: `60`.
- `enabled` - (Optional, Boolean) Whether the plugin is enabled. Default: `true`.
- `env` - (Optional, Computed, Set of String) A set of environment variables to pass to the plugin, in the format `KEY=VALUE`.
- `force_destroy` - (Optional, Boolean) Whether to force removal of the plugin when destroying the resource. Default: `false`.
- `force_disable` - (Optional, Boolean) Whether to force disable the plugin when disabling. Default: `false`.
- `grant_all_permissions` - (Optional, Boolean) Whether to automatically grant all permissions requested by the plugin. Default: `false`.
- `grant_permissions` - (Optional, Block Set) A set of specific permissions to grant to the plugin. See [grant_permissions](#grant_permissions) below for details.

### grant_permissions

- `name` - (Required, String) The name of the permission.
- `value` - (Required, Set of String) The values for the permission.

## Attribute Reference

In addition to all arguments above, the following attributes are exported:

- `id` - (String) The ID of the plugin.
- `plugin_reference` - (String) The full reference of the installed plugin.

## Import

Podman plugins can be imported using the plugin ID:

```shell
terraform import podman_plugin.sample PLUGIN_ID
```
