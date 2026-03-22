---
page_title: "podman_registry_image Resource - terraform-provider-podman"
subcategory: ""
description: |-
  Manages the push lifecycle of a Podman image to a container registry.
---

# podman_registry_image (Resource)

Manages the push lifecycle of a Podman image to a container registry. This
resource pushes a locally available image to a remote registry and can
optionally manage its deletion on destroy.

## Example Usage

```terraform
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
```

## Argument Reference

The following arguments are supported:

- `name` - (Required, String) The full image name including the registry (e.g.
  `registry.example.com/my-app:latest`).
- `insecure_skip_verify` - (Optional, Boolean) Skip TLS verification when
  communicating with the registry. Defaults to `false`.
- `keep_remotely` - (Optional, Boolean) If `true`, the image will not be
  deleted from the remote registry on `terraform destroy`. Defaults to `false`.
- `triggers` - (Optional, Map of String) A map of arbitrary values that, when
  changed, will cause the image to be re-pushed to the registry.

## Attribute Reference

In addition to all arguments above, the following attributes are exported:

- `id` - The resource identifier.
- `sha256_digest` - The digest of the image in the registry.
