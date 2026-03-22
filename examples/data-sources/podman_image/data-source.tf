# Look up an existing image by name
data "podman_image" "nginx" {
  name = "docker.io/library/nginx:latest"
}

# Use the image digest in another resource
output "nginx_digest" {
  value = data.podman_image.nginx.repo_digest
}
