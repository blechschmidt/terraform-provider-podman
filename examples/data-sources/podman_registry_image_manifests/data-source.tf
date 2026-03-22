# Read manifest list for a multi-platform image
data "podman_registry_image_manifests" "nginx" {
  name = "docker.io/library/nginx:latest"
}

output "nginx_platforms" {
  value = data.podman_registry_image_manifests.nginx.manifests
}

# With explicit authentication for a private registry
data "podman_registry_image_manifests" "private_app" {
  name                 = "registry.example.com/myapp:v2.0"
  insecure_skip_verify = false

  auth_config {
    address  = "registry.example.com"
    username = var.registry_username
    password = var.registry_password
  }
}
