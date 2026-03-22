resource "podman_image" "app" {
  name = "my-app:latest"

  build {
    context = "${path.module}/app"
  }
}

resource "podman_tag" "app" {
  source_image = podman_image.app.name
  target_image = "registry.example.com/my-app:latest"
}

resource "podman_registry_image" "app" {
  name = "registry.example.com/my-app:latest"

  triggers = {
    image_id = podman_image.app.image_id
  }

  depends_on = [podman_tag.app]
}
