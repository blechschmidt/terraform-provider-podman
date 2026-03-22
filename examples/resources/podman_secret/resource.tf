resource "podman_secret" "db_credentials" {
  name = "db_password"
  data = var.db_password

  labels {
    label = "application"
    value = "my-app"
  }
}
