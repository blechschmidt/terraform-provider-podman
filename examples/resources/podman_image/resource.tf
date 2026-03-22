# Pull an image from a registry
resource "podman_image" "nginx" {
  name = "docker.io/library/nginx:latest"
}

# Build an image from a Dockerfile
resource "podman_image" "app" {
  name = "my-app:latest"

  build {
    context    = "${path.module}/app"
    dockerfile = "Dockerfile"

    build_args = {
      GO_VERSION = "1.21"
    }

    labels = {
      maintainer = "team@example.com"
    }

    tag = ["my-app:v1.0.0"]
  }
}

# Pull a specific platform image
resource "podman_image" "nginx_arm" {
  name     = "docker.io/library/nginx:latest"
  platform = "linux/arm64"
}
