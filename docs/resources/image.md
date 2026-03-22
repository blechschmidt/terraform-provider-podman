---
page_title: "podman_image Resource - terraform-provider-podman"
subcategory: ""
description: |-
  Manages the lifecycle of a Podman image.
---

# podman_image (Resource)

Manages the lifecycle of a Podman image. This resource can pull images from a
registry or build them from a Dockerfile.

## Example Usage

### Basic Image Pull

```terraform
resource "podman_image" "nginx" {
  name = "docker.io/library/nginx:latest"
}
```

### Build from Dockerfile

```terraform
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
```

### Pull with Platform

```terraform
resource "podman_image" "nginx_arm" {
  name     = "docker.io/library/nginx:latest"
  platform = "linux/arm64"

  pull_triggers = [
    data.podman_registry_image.nginx.sha256_digest,
  ]
}
```

## Argument Reference

The following arguments are supported:

- `name` - (Required, String) The name of the image to pull or build (e.g.
  `docker.io/library/nginx:latest`).
- `build` - (Optional, Block, Max 1) Configuration block for building the image
  from a Dockerfile. See [Build](#build) below for details.
- `force_remove` - (Optional, Boolean) Force remove the image on destroy.
  Defaults to `false`.
- `keep_locally` - (Optional, Boolean) If `true`, the image will not be deleted
  on `terraform destroy`. Defaults to `false`.
- `platform` - (Optional, String) The target platform for pulling the image
  (e.g. `linux/amd64`, `linux/arm64`).
- `pull_triggers` - (Optional, List of String) A list of values that, when
  changed, will cause the image to be re-pulled.
- `triggers` - (Optional, Map of String) A map of arbitrary values that, when
  changed, will cause the image to be re-pulled or rebuilt.

### Build

The `build` block supports the following arguments:

- `context` - (Required, String) The path to the build context directory.
- `dockerfile` - (Optional, String) The path to the Dockerfile relative to the
  build context. Defaults to `Dockerfile`.
- `build_args` - (Optional, Map of String) A map of build-time variables passed
  to the builder.
- `cache_from` - (Optional, List of String) A list of images to use as cache
  sources during the build.
- `force_remove` - (Optional, Boolean) Always remove intermediate containers,
  even upon failure.
- `labels` - (Optional, Map of String) A map of metadata labels to apply to the
  image.
- `no_cache` - (Optional, Boolean) Do not use the build cache.
- `platform` - (Optional, String) The target platform for the build (e.g.
  `linux/amd64`).
- `remove` - (Optional, Boolean) Remove intermediate containers after a
  successful build. Defaults to `true`.
- `tag` - (Optional, List of String) Additional tags to apply to the built
  image.
- `target` - (Optional, String) The target build stage to stop at in a
  multi-stage Dockerfile.
- `network_mode` - (Optional, String) The networking mode for RUN instructions
  during the build (e.g. `host`, `none`).
- `extra_hosts` - (Optional, List of String) A list of extra hosts to add to
  `/etc/hosts` during the build, in the form `hostname:IP`.
- `shm_size` - (Optional, Number) Size of `/dev/shm` in bytes.
- `cpu_period` - (Optional, Number) The length of a CPU CFS period in
  microseconds.
- `cpu_quota` - (Optional, Number) The CPU CFS quota in microseconds.
- `cpu_set_cpus` - (Optional, String) CPUs in which to allow execution (e.g.
  `0-3`, `0,1`).
- `cpu_shares` - (Optional, Number) CPU shares (relative weight).
- `memory` - (Optional, Number) Memory limit in bytes.
- `memory_swap` - (Optional, Number) Total memory limit (memory + swap) in
  bytes. Set to `-1` to disable swap.

## Attribute Reference

In addition to all arguments above, the following attributes are exported:

- `id` - The SHA256 image ID.
- `image_id` - The SHA256 image ID.
- `repo_digest` - The image repo digest.

## Import

Import an image using the image name:

```shell
terraform import podman_image.example IMAGE_NAME
```

For example:

```shell
terraform import podman_image.example docker.io/library/nginx:latest
```
