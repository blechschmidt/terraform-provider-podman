terraform {
  required_providers {
    podman = {
      source = "blechschmidt/podman"
    }
  }
}

provider "podman" {
  host = "unix:///run/user/1000/podman/podman.sock"
}

# --- Images ---

resource "podman_image" "alpine" {
  name = "docker.io/library/alpine:latest"
}

resource "podman_image" "nginx" {
  name = "docker.io/library/nginx:alpine"
}

# --- Network ---

resource "podman_network" "app" {
  name   = "tf-app-net"
  driver = "bridge"

  ipam_config {
    subnet  = "172.30.0.0/24"
    gateway = "172.30.0.1"
  }

  labels {
    label = "managed-by"
    value = "terraform"
  }
}

# --- Volumes ---

resource "podman_volume" "html" {
  name = "tf-nginx-html"
}

resource "podman_volume" "data" {
  name = "tf-app-data"
}

# --- Containers ---

# Nginx reverse proxy with port mapping and custom network
resource "podman_container" "nginx" {
  name  = "tf-nginx"
  image = podman_image.nginx.image_id

  ports {
    internal = 80
    external = 8080
  }

  volumes {
    volume_name    = podman_volume.html.name
    container_path = "/usr/share/nginx/html"
    read_only      = true
  }

  networks_advanced {
    name         = podman_network.app.name
    ipv4_address = "172.30.0.10"
  }

  env = [
    "NGINX_HOST=localhost",
  ]

  labels {
    label = "app"
    value = "webserver"
  }

  restart = "unless-stopped"
}

# Alpine worker with volume, healthcheck, and resource limits
resource "podman_container" "worker" {
  name  = "tf-worker"
  image = podman_image.alpine.image_id

  command = ["sh", "-c", "while true; do date >> /data/log.txt; sleep 5; done"]

  volumes {
    volume_name    = podman_volume.data.name
    container_path = "/data"
  }

  networks_advanced {
    name         = podman_network.app.name
    ipv4_address = "172.30.0.20"
  }

  healthcheck {
    test     = ["CMD-SHELL", "test -f /data/log.txt"]
    interval = "10s"
    timeout  = "3s"
    retries  = 3
  }

  memory     = 67108864 # 64 MB
  cpu_shares = 512

  labels {
    label = "app"
    value = "worker"
  }

  restart = "on-failure"
}

# Alpine container with file upload
resource "podman_container" "init" {
  name  = "tf-init"
  image = podman_image.alpine.image_id

  command = ["cat", "/etc/app/config.json"]

  upload {
    file    = "/etc/app/config.json"
    content = jsonencode({
      name    = "terraform-podman-test"
      version = "0.1.1"
      debug   = true
    })
  }

  must_run = false
  start    = true
  logs     = true
  rm       = false
}

# --- Tag ---

resource "podman_tag" "alpine_custom" {
  source_image = podman_image.alpine.image_id
  target_image = "localhost/my-alpine:test"
}

# --- Data sources ---

data "podman_image" "alpine" {
  name       = podman_image.alpine.name
  depends_on = [podman_image.alpine]
}

data "podman_network" "app" {
  name       = podman_network.app.name
  depends_on = [podman_network.app]
}

data "podman_logs" "init" {
  name                     = podman_container.init.name
  show_stdout              = true
  show_stderr              = true
  logs_list_string_enabled = true
  depends_on               = [podman_container.init]
}

# --- Outputs ---

output "nginx_url" {
  value = "http://localhost:8080"
}

output "nginx_ip" {
  value = "172.30.0.10"
}

output "worker_id" {
  value = podman_container.worker.id
}

output "init_logs" {
  value = data.podman_logs.init.logs_list_string
}

output "alpine_digest" {
  value = data.podman_image.alpine.repo_digest
}

output "network_subnet" {
  value = data.podman_network.app.ipam_config
}

output "tagged_image" {
  value = podman_tag.alpine_custom.source_image_id
}
