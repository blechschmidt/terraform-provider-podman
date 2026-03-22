# Read the digest of a remote image
data "podman_registry_image" "nginx" {
  name = "docker.io/library/nginx:latest"
}

# Use the digest to pull or track changes
output "nginx_sha256" {
  value = data.podman_registry_image.nginx.sha256_digest
}

# Example with a private registry (TLS verification disabled)
data "podman_registry_image" "internal_app" {
  name                 = "registry.internal.example.com/myapp:v1.0"
  insecure_skip_verify = true
}
