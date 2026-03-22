---
page_title: "podman_registry_image_manifests Data Source - Podman"
subcategory: ""
description: |-
  Reads manifest information for a multi-platform image from a container registry.
---

# podman_registry_image_manifests (Data Source)

Reads the manifest list (also known as a fat manifest) for a multi-platform image from a container registry. This data source is useful for discovering which platforms an image supports and retrieving platform-specific digests.

## Example Usage

{{tffile "examples/data-sources/podman_registry_image_manifests/data-source.tf"}}

## Argument Reference

- `name` (String, Required) - The name of the image, including the tag (e.g., `docker.io/library/nginx:latest`).
- `insecure_skip_verify` (Boolean, Optional) - Whether to skip TLS verification when communicating with the registry. Defaults to `false`.
- `auth_config` (Block, Optional, Max: 1) - Override the provider-level registry authentication for this data source. See [Nested Schema for `auth_config`](#nestedblock--auth_config) below.

<a id="nestedblock--auth_config"></a>

### Nested Schema for `auth_config`

#### Required

- `address` (String) - The address of the registry (e.g., `docker.io`).
- `username` (String) - The username to authenticate with.
- `password` (String, Sensitive) - The password to authenticate with.

## Attribute Reference

In addition to all arguments above, the following attributes are exported:

- `id` (String) - The image name.
- `manifests` (Set of Object) - The list of platform-specific manifests. Each object contains the following attributes:
  - `architecture` (String) - The CPU architecture (e.g., `amd64`, `arm64`).
  - `media_type` (String) - The media type of the manifest (e.g., `application/vnd.docker.distribution.manifest.v2+json`).
  - `os` (String) - The operating system (e.g., `linux`).
  - `sha256_digest` (String) - The SHA256 digest of the platform-specific manifest.
