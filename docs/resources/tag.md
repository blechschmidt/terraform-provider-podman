---
page_title: "podman_tag Resource - terraform-provider-podman"
subcategory: ""
description: |-
  Creates a tag for an existing Podman image.
---

# podman_tag (Resource)

Creates a tag for an existing Podman image. This resource allows you to assign
additional names or tags to images that already exist in local storage.

## Example Usage

```terraform
resource "podman_image" "nginx" {
  name = "docker.io/library/nginx:latest"
}

resource "podman_tag" "custom" {
  source_image = podman_image.nginx.name
  target_image = "my-registry.example.com/nginx:v1"
}
```

## Argument Reference

The following arguments are supported:

- `source_image` - (Required, String) The name or ID of the source image to
  tag.
- `target_image` - (Required, String) The target image reference including the
  new tag (e.g. `my-registry.example.com/nginx:v1`).

## Attribute Reference

In addition to all arguments above, the following attributes are exported:

- `id` - The resource identifier.
- `source_image_id` - The SHA256 ID of the source image.
