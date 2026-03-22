# Look up a plugin by alias
data "podman_plugin" "by_alias" {
  alias = "my-volume-plugin"
}

# Look up a plugin by ID
data "podman_plugin" "by_id" {
  id = "abc123def456"
}

output "plugin_enabled" {
  value = data.podman_plugin.by_alias.enabled
}
