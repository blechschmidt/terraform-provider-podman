terraform {
  required_providers {
    podman = {
      source = "hashicorp/podman"
    }
  }
}

provider "podman" {
  host = "unix:///run/podman/podman.sock"
}

# Pull an image
resource "podman_image" "nginx" {
  name = "docker.io/library/nginx:latest"
}

# Create a network
resource "podman_network" "my_network" {
  name   = "my-network"
  driver = "bridge"

  ipam_config {
    subnet  = "172.20.0.0/16"
    gateway = "172.20.0.1"
  }
}

# Create a volume
resource "podman_volume" "data" {
  name = "my-data"
}

# Run a container
resource "podman_container" "nginx" {
  name  = "my-nginx"
  image = podman_image.nginx.image_id

  ports {
    internal = 80
    external = 8080
  }

  volumes {
    volume_name    = podman_volume.data.name
    container_path = "/usr/share/nginx/html"
    read_only      = false
  }

  networks_advanced {
    name         = podman_network.my_network.name
    ipv4_address = "172.20.0.10"
  }

  env = [
    "NGINX_HOST=localhost",
    "NGINX_PORT=80",
  ]

  labels {
    label = "maintainer"
    value = "terraform"
  }
}

# Data sources
data "podman_image" "nginx" {
  name = "docker.io/library/nginx:latest"

  depends_on = [podman_image.nginx]
}

data "podman_network" "my_network" {
  name = podman_network.my_network.name
}

output "container_id" {
  value = podman_container.nginx.id
}

output "image_digest" {
  value = data.podman_image.nginx.repo_digest
}
