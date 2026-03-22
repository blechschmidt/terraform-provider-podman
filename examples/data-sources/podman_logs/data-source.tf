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
