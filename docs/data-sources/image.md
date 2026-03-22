---
page_title: "podman_image Data Source - Podman"
subcategory: ""
description: |-
  Reads information about an existing Podman image.
---

# podman_image (Data Source)

Reads information about an existing Podman image. This data source can be used to look up the ID and repository digest of an image that is already present on the Podman host.

## Example Usage

```hcl
# Look up an existing image by name
data "podman_image" "nginx" {
  name = "docker.io/library/nginx:latest"
}

# Use the image digest in another resource
output "nginx_digest" {
  value = data.podman_image.nginx.repo_digest
}
```

## Argument Reference

- `name` (String, Required) - The name of the image, including the tag (e.g., `docker.io/library/nginx:latest`).

## Attribute Reference

In addition to all arguments above, the following attributes are exported:

- `id` (String) - The image ID.
- `repo_digest` (String) - The image digest in the format `algorithm:hex`.
