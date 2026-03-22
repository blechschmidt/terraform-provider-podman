---
page_title: "Provider: Podman"
subcategory: ""
description: |-
  The Podman provider interacts with Podman using its Docker-compatible API.
---

# Podman Provider

The Podman provider is used to interact with Podman containers, images, volumes, and networks. It communicates with Podman through its Docker-compatible API, allowing you to manage container infrastructure using Terraform.

Use the navigation to the left to read about the available resources and data sources.

## Example Usage

### Basic Configuration (Rootless Socket)

```hcl
# Basic provider configuration using rootless Podman socket
provider "podman" {
  host = "unix:///run/user/1000/podman/podman.sock"
}

# Provider configuration with registry authentication
provider "podman" {
  host = "unix:///run/user/1000/podman/podman.sock"

  registry_auth {
    address  = "registry.example.com"
    username = "user"
    password = "pass"
  }
}
```

## Authentication

The Podman provider can authenticate against container registries using the `registry_auth` block. You can configure multiple `registry_auth` blocks to authenticate against different registries.

Registry credentials can also be provided via environment variables `DOCKER_REGISTRY_USER` and `DOCKER_REGISTRY_PASS`, or through a Docker-compatible configuration file.

## Schema

### Optional

- `host` (String) The address of the Podman daemon socket. Defaults to `unix:///run/podman/podman.sock`. Can also be set with the `DOCKER_HOST` environment variable. For rootless Podman, use `unix:///run/user/<UID>/podman/podman.sock`.
- `cert_path` (String) Path to a directory containing TLS certificates (`ca.pem`, `cert.pem`, `key.pem`) to use for authenticating to the Podman daemon. Can also be set with the `DOCKER_CERT_PATH` environment variable.
- `ca_material` (String) PEM-encoded CA certificate to use for TLS authentication to the Podman daemon.
- `cert_material` (String) PEM-encoded client certificate to use for TLS authentication to the Podman daemon.
- `key_material` (String, Sensitive) PEM-encoded client private key to use for TLS authentication to the Podman daemon.
- `ssh_opts` (List of String) Additional SSH options for connections using the `ssh://` protocol.
- `registry_auth` (Block Set) Configuration blocks for authenticating against container registries. See [Registry Auth](#nestedblock--registry_auth) below for details.

<a id="nestedblock--registry_auth"></a>

### Nested Schema for `registry_auth`

#### Required

- `address` (String) The address of the registry.

#### Optional

- `config_file` (String) Path to a Docker-compatible config file. Defaults to `~/.docker/config.json`.
- `config_file_content` (String) The content of a Docker-compatible config file. If specified, `config_file` is ignored.
- `username` (String) The username to authenticate with. Can also be set with the `DOCKER_REGISTRY_USER` environment variable.
- `password` (String, Sensitive) The password to authenticate with. Can also be set with the `DOCKER_REGISTRY_PASS` environment variable.
- `auth_disabled` (Boolean) Set to `true` to disable authentication for this registry. Defaults to `false`.
