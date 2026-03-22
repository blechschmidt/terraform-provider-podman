# Basic named volume
resource "podman_volume" "data" {
  name = "app_data"
}

# Volume with driver options
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
