---
page_title: "podman_logs Data Source - Podman"
subcategory: ""
description: |-
  Reads log output from a Podman container.
---

# podman_logs (Data Source)

Reads log output from a Podman container. This data source can be used to inspect the stdout and stderr output of a running or stopped container.

## Example Usage

```hcl
# Read the last 50 lines of logs from a container
data "podman_logs" "app" {
  name                     = "my-app-container"
  tail                     = "50"
  timestamps               = true
  logs_list_string_enabled = true
}

output "app_logs" {
  value = data.podman_logs.app.logs_list_string
}

# Read only stderr since a specific time
data "podman_logs" "errors" {
  name        = "my-app-container"
  show_stdout = false
  show_stderr = true
  since       = "2024-01-01T00:00:00Z"
}
```

## Argument Reference

- `name` (String, Required) - The name or ID of the container.
- `details` (Boolean, Optional) - Show extra details provided to logs. Defaults to `false`.
- `discard_headers` (Boolean, Optional) - Strip Docker-style multiplex headers from the log output. Defaults to `false`.
- `follow` (Boolean, Optional) - Follow log output as it is produced. Defaults to `false`.
- `logs_list_string_enabled` (Boolean, Optional) - Enable the `logs_list_string` attribute in the output. Defaults to `false`.
- `show_stderr` (Boolean, Optional) - Include stderr in the log output. Defaults to `true`.
- `show_stdout` (Boolean, Optional) - Include stdout in the log output. Defaults to `true`.
- `since` (String, Optional) - Show logs since a given timestamp (e.g., `2023-01-01T00:00:00Z`) or relative time (e.g., `42m` for 42 minutes).
- `tail` (String, Optional) - Number of lines to show from the end of the logs. Set to `all` to show the complete log. Defaults to `all`.
- `timestamps` (Boolean, Optional) - Prepend a timestamp to each log line. Defaults to `false`.
- `until` (String, Optional) - Show logs before a given timestamp or relative time.

## Attribute Reference

In addition to all arguments above, the following attributes are exported:

- `id` (String) - The container name or ID.
- `logs_list_string` (List of String) - The log output as a list of strings, one entry per log line. Only populated when `logs_list_string_enabled` is set to `true`.
