---
page_title: "podman_registry_image Data Source - Podman"
subcategory: ""
description: |-
  Reads the image digest from a container registry.
---

# podman_registry_image (Data Source)

Reads the image digest from a container registry without pulling the image. This is useful for tracking image updates or triggering resource recreation when a remote image changes.

## Example Usage

```hcl
# Read the digest of a remote image
data "podman_registry_image" "nginx" {
  name = "docker.io/library/nginx:latest"
}

# Use the digest to pull or track changes
output "nginx_sha256" {
  value = data.podman_registry_image.nginx.sha256_digest
}

# Example with a private registry (TLS verification disabled)
data "podman_registry_image" "internal_app" {
  name                 = "registry.internal.example.com/myapp:v1.0"
  insecure_skip_verify = true
}
```

## Argument Reference

- `name` (String, Required) - The name of the image, including the tag (e.g., `docker.io/library/nginx:latest`).
- `insecure_skip_verify` (Boolean, Optional) - Whether to skip TLS verification when communicating with the registry. Defaults to `false`.

## Attribute Reference

In addition to all arguments above, the following attributes are exported:

- `id` (String) - The image name.
- `sha256_digest` (String) - The SHA256 digest of the image manifest from the registry.
