# Terraform Provider for Podman

A Terraform provider for managing [Podman](https://podman.io/) containers, images, networks, volumes, and more. It communicates with Podman through its Docker-compatible REST API, providing the same resource model as the [kreuzwerker/docker](https://registry.terraform.io/providers/kreuzwerker/docker/latest/docs) Terraform provider.

## Requirements

- [Terraform](https://developer.hashicorp.com/terraform/install) >= 1.0
- [Go](https://go.dev/dl/) >= 1.22 (for building from source)
- [Podman](https://podman.io/docs/installation) >= 4.0 with the API socket enabled

### Enabling the Podman API socket

```bash
# Rootless (recommended)
systemctl --user start podman.socket

# Root
sudo systemctl start podman.socket
```

## Installation

### From source

```bash
git clone https://github.com/blechschmidt/terraform-provider-podman.git
cd terraform-provider-podman
make install
```

This installs the provider to `~/.terraform.d/plugins/`.

### Dev override (for development)

Add a dev override to your `~/.terraformrc`:

```hcl
provider_installation {
  dev_overrides {
    "blechschmidt/podman" = "/path/to/terraform-provider-podman"
  }
  direct {}
}
```

## Usage

```hcl
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

resource "podman_image" "nginx" {
  name = "docker.io/library/nginx:latest"
}

resource "podman_volume" "data" {
  name = "web-data"
}

resource "podman_container" "web" {
  name  = "my-nginx"
  image = podman_image.nginx.image_id

  ports {
    internal = 80
    external = 8080
  }

  volumes {
    volume_name    = podman_volume.data.name
    container_path = "/usr/share/nginx/html"
  }
}
```

## Resources

| Resource | Description |
|---|---|
| `podman_container` | Manages the lifecycle of a container |
| `podman_image` | Pulls or builds container images |
| `podman_network` | Manages container networks |
| `podman_volume` | Manages persistent volumes |
| `podman_tag` | Creates image tags |
| `podman_plugin` | Manages plugins |
| `podman_secret` | Manages secrets |
| `podman_registry_image` | Pushes images to a registry |

## Data Sources

| Data Source | Description |
|---|---|
| `podman_image` | Reads image information |
| `podman_network` | Reads network information |
| `podman_plugin` | Reads plugin information |
| `podman_registry_image` | Reads a registry image digest |
| `podman_logs` | Reads container log output |
| `podman_registry_image_manifests` | Reads multi-platform manifest lists |

## Provider Configuration

| Argument | Description |
|---|---|
| `host` | Podman socket address. Default: `unix:///run/podman/podman.sock`. Env: `DOCKER_HOST` |
| `cert_path` | Path to TLS certificate directory |
| `ca_material` | PEM-encoded CA certificate |
| `cert_material` | PEM-encoded client certificate |
| `key_material` | PEM-encoded client private key |
| `ssh_opts` | Additional SSH options for `ssh://` connections |
| `registry_auth` | Registry authentication blocks |

See the [full documentation](docs/index.md) for details.

## Building

```bash
make build    # Build the provider binary
make test     # Run unit tests
make testacc  # Run acceptance tests (requires running Podman)
make lint     # Run fmt + vet
```

## License

[MIT](LICENSE)
