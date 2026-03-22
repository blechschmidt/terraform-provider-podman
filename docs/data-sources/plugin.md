---
page_title: "podman_plugin Data Source - Podman"
subcategory: ""
description: |-
  Reads information about an existing Podman plugin.
---

# podman_plugin (Data Source)

Reads information about an existing Podman plugin. You can look up a plugin by its `alias` or `id`. At least one of these arguments must be specified.

## Example Usage

{{tffile "examples/data-sources/podman_plugin/data-source.tf"}}

## Argument Reference

- `alias` (String, Optional) - The alias of the plugin. At least one of `alias` or `id` must be specified.
- `id` (String, Optional) - The ID of the plugin. At least one of `alias` or `id` must be specified.

## Attribute Reference

In addition to all arguments above, the following attributes are exported:

- `enabled` (Boolean) - Whether the plugin is currently enabled.
- `env` (Set of String) - The environment variables configured for the plugin.
- `grant_all_permissions` (Boolean) - Whether all permissions have been granted to the plugin.
- `name` (String) - The fully qualified name of the plugin.
- `plugin_reference` (String) - The reference of the plugin from the registry.
