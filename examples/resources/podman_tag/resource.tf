resource "podman_image" "nginx" {
  name = "docker.io/library/nginx:latest"
}

resource "podman_tag" "custom" {
  source_image = podman_image.nginx.name
  target_image = "my-registry.example.com/nginx:v1"
}
