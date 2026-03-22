---
page_title: "podman_registry_image Data Source - Podman"
subcategory: ""
description: |-
  Reads the image digest from a container registry.
---

# podman_registry_image (Data Source)

Reads the image digest from a container registry without pulling the image. This is useful for tracking image updates or triggering resource recreation when a remote image changes.

## Example Usage

{{tffile "examples/data-sources/podman_registry_image/data-source.tf"}}

## Argument Reference

- `name` (String, Required) - The name of the image, including the tag (e.g., `docker.io/library/nginx:latest`).
- `insecure_skip_verify` (Boolean, Optional) - Whether to skip TLS verification when communicating with the registry. Defaults to `false`.

## Attribute Reference

In addition to all arguments above, the following attributes are exported:

- `id` (String) - The image name.
- `sha256_digest` (String) - The SHA256 digest of the image manifest from the registry.
