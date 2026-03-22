resource "podman_image" "nginx" {
  name = "docker.io/library/nginx:latest"
}

resource "podman_volume" "data" {
  name = "app_data"
}

resource "podman_container" "example" {
  name  = "my-nginx"
  image = podman_image.nginx.image_id

  ports {
    internal = 80
    external = 8080
    protocol = "tcp"
  }

  ports {
    internal = 443
    external = 8443
  }

  env = [
    "NGINX_HOST=example.com",
    "NGINX_PORT=80",
  ]

  volumes {
    volume_name    = podman_volume.data.name
    container_path = "/usr/share/nginx/html"
    read_only      = false
  }

  volumes {
    host_path      = "/var/log/nginx"
    container_path = "/var/log/nginx"
    read_only      = false
  }

  labels {
    label = "app"
    value = "nginx"
  }

  labels {
    label = "environment"
    value = "production"
  }

  restart = "unless-stopped"
}
