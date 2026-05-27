# kreuzwerker/docker parity audit (vs terraform-provider-podman)

> **Status (2026-05-28):** The gaps listed in this audit have been implemented
> as part of the v0.4.0 parity pass. Items still marked **investigate**, or
> tagged "podman API limitation" below, are intentionally deferred. Notable
> gaps that remain by design (per podman platform constraints):
>
> - `docker_service`, `docker_config`, `docker_buildx_builder`, `docker_compose` — out of scope.
> - `docker_volume.cluster` — Swarm CSI; not supported by podman.
> - `docker_container.device_requests` — implementable via libpod CDI but
>   no GPU is available in our test environment; not wired today.
> - `docker_container.gpus` — schema field accepted, no test coverage (no GPU).
> - `docker_container.networks_advanced.gw_priority` — newer docker engine field;
>   libpod support not confirmed for the podman versions we target.
> - `docker_image.build.{cache_to, secrets, additional_contexts, provenance, sbom}` —
>   BuildKit-only or buildah version-dependent; not exposed by the docker-compat API.
> - `docker_image.build.{ulimit, security_opt, extra_hosts}` are accepted in the
>   schema for parity but the docker SDK and podman's `/build` endpoint disagree
>   on encoding (docker SDK sends repeated `key=v1&key=v2`, podman expects a
>   JSON array). Setting these at apply time fails today; a follow-up that
>   builds against the libpod `/build` endpoint directly will restore them.
> - `data.podman_registry_image` & `data.podman_registry_image_manifests`
>   bypass the docker `DistributionInspect` endpoint (which podman does not
>   implement) and query the registry HTTP API directly.



Compared against `kreuzwerker/terraform-provider-docker` master branch
(`internal/provider/`). Sources used: `resource_docker_*.go`,
`data_source_docker_*.go`, `provider.go`, `authentication_helpers.go`.

Classification:
- **ok** — present in both, attribute parity is good enough.
- **add** — docker has it, podman lacks it, podman/libpod supports it.
- **investigate** — docker has it; unclear whether podman exposes it.
- **skip** — docker-only / swarm-only / not meaningful for podman.

The "Configurable arguments" rows are the planned diff target; computed-only
attributes are listed at the end of each resource where they differ.

---

## Provider configuration

| Attribute | docker | podman | Status |
| --- | --- | --- | --- |
| host | yes | yes | ok |
| ssh_opts | yes | yes | ok |
| cert_path | yes | yes | ok |
| ca_material | yes | yes | ok |
| cert_material | yes | yes | ok |
| key_material | yes | yes | ok |
| registry_auth (address/username/password/config_file/config_file_content/auth_disabled) | yes | yes | ok |
| context | yes | no | add (Docker context name — podman has its own contexts via `podman system connection`; map to selecting connection or skip if too docker-cli-specific). Mark **investigate**. |
| disable_docker_daemon_check | yes | no | add (rename: `disable_podman_daemon_check`; useful for plan-only/CI). |

---

## Resources

### Existing in both: container (docker_container ↔ podman_container)

| Attribute | docker | podman | Status |
| --- | --- | --- | --- |
| name | yes | yes | ok |
| image | yes | yes | ok |
| rootfs | no | yes | podman extension (libpod-only; keep) |
| rootfs_overlay | no | yes | podman extension |
| rootfs_mapping | no | yes | podman extension |
| rm | yes | yes | ok |
| read_only | yes | yes | ok |
| start | yes | yes | ok |
| wait | yes | yes | ok |
| wait_timeout | yes | yes | ok |
| attach | yes | yes | ok |
| logs | yes | yes | ok |
| must_run | yes | yes | ok |
| hostname | yes | yes | ok |
| domainname | yes | yes | ok |
| command | yes | yes | ok |
| entrypoint | yes | yes | ok |
| user | yes | yes | ok |
| dns | yes | yes | ok |
| dns_opts | yes | yes | ok |
| dns_search | yes | yes | ok |
| publish_all_ports | yes | yes | ok |
| restart | yes | yes | ok |
| max_retry_count | yes | yes | ok |
| working_dir | yes | yes | ok |
| remove_volumes | yes | yes | ok |
| capabilities (add, drop) | yes | yes | ok |
| security_opts | yes | yes | ok |
| runtime | yes | yes | ok |
| platform | yes | no | **add** (podman pull/run supports `--platform`; container create accepts it via libpod). |
| stop_signal | yes | yes | ok |
| stop_timeout | yes | yes | ok |
| mounts.target | yes | yes | ok |
| mounts.source | yes | yes | ok |
| mounts.type | yes | yes | ok |
| mounts.read_only | yes | yes | ok |
| mounts.bind_options.propagation | yes | yes | ok |
| mounts.volume_options.no_copy | yes | yes | ok |
| mounts.volume_options.labels | yes | yes | ok |
| mounts.volume_options.driver_name | yes | yes | ok |
| mounts.volume_options.driver_options | yes | yes | ok |
| mounts.volume_options.subpath | yes | no | **add** (podman supports mount subpath via libpod and compat). |
| mounts.tmpfs_options.size_bytes | yes | yes | ok |
| mounts.tmpfs_options.mode | yes | yes | ok |
| volumes.from_container | yes | yes | ok |
| volumes.container_path | yes | yes | ok |
| volumes.host_path | yes | yes | ok |
| volumes.volume_name | yes | yes | ok |
| volumes.read_only | yes | yes | ok |
| volumes.selinux_relabel | yes | no | **add** (podman natively supports `:z`/`:Z` SELinux relabel — important on Fedora/RHEL). |
| tmpfs | yes | yes | ok |
| ports.internal | yes | yes | ok |
| ports.external | yes | yes | ok |
| ports.ip | yes | yes | ok |
| ports.protocol | yes | yes | ok |
| host.host | yes | yes | ok |
| host.ip | yes | yes | ok |
| ulimit (name/soft/hard) | yes | yes | ok |
| env | yes | yes | ok |
| privileged | yes | yes | ok |
| devices.host_path | yes | yes | ok |
| devices.container_path | yes | yes | ok |
| devices.permissions | yes | yes | ok |
| device_read_bps (path, rate) | yes | no | **add** (block I/O throttle; libpod supports it). |
| device_read_iops (path, rate) | yes | no | **add** |
| device_write_bps (path, rate) | yes | no | **add** |
| device_write_iops (path, rate) | yes | no | **add** |
| device_requests (driver, count, device_ids, capabilities, options) | yes | no | **add** (GPU device requests; docker compat field — podman supports it via CDI / `--device`). Mark **investigate** if device_requests struct maps cleanly; otherwise prefer the simpler `gpus` field. |
| destroy_grace_seconds | yes | yes | ok |
| labels | yes | yes | ok |
| memory | yes | yes | ok |
| memory_reservation | yes | no | **add** (soft memory limit; libpod supports it). |
| memory_swap | yes | yes | ok |
| shm_size | yes | yes | ok |
| cpu_shares | yes | yes | ok |
| cpu_set | yes | yes | ok |
| cpus | yes | no | **add** (string form, e.g. "1.5"; convenience over cpu_period+cpu_quota — libpod has `CPUs`). |
| cpu_period | yes | yes | ok |
| cpu_quota | yes | yes | ok |
| log_driver | yes | yes | ok |
| log_opts | yes | yes | ok |
| network_mode | yes | yes | ok |
| networks_advanced.name | yes | yes | ok |
| networks_advanced.aliases | yes | yes | ok |
| networks_advanced.ipv4_address | yes | yes | ok |
| networks_advanced.ipv6_address | yes | yes | ok |
| networks_advanced.link_local_ips | yes | no | **add** (libpod-compat exposes link-local IPs). |
| networks_advanced.mac_address | yes | no | **add** (podman supports per-endpoint MAC). |
| networks_advanced.driver_opts | yes | no | **add** (per-endpoint driver opts; libpod supports). |
| networks_advanced.gw_priority | yes | no | **investigate** (newer docker compose field; podman support unclear). |
| pid_mode | yes | yes | ok |
| ipc_mode | yes | yes | ok |
| userns_mode | yes | yes | ok |
| cgroupns_mode | yes | yes | ok |
| cgroup_parent | yes | no | **add** (libpod supports `--cgroup-parent`). |
| upload.file | yes | yes | ok |
| upload.content | yes | yes | ok |
| upload.content_base64 | yes | yes | ok |
| upload.executable | yes | yes | ok |
| upload.source | yes | yes | ok |
| upload.source_hash | yes | yes | ok |
| upload.permissions | yes | no | **add** (explicit file mode beyond just the executable bool). |
| healthcheck.test | yes | yes | ok |
| healthcheck.interval | yes | yes | ok |
| healthcheck.timeout | yes | yes | ok |
| healthcheck.start_period | yes | yes | ok |
| healthcheck.start_interval | yes | no | **add** (newer docker engine field; libpod has `StartInterval`). |
| healthcheck.retries | yes | yes | ok |
| container_read_refresh_timeout_milliseconds | yes | no | **add** (tunes how long Read polls; pure provider-side knob). |
| sysctls | yes | yes | ok |
| group_add | yes | yes | ok |
| init | yes | yes | ok |
| tty | yes | yes | ok |
| stdin_open | yes | yes | ok |
| storage_opts | yes | yes | ok |
| gpus | yes | no | **add** (string convenience for GPU passthrough; podman supports `--gpus`/CDI). |
| exit_code (computed) | yes | yes | ok |
| container_logs (computed) | yes | yes | ok |
| bridge (computed) | yes | yes | ok |
| network_data.* (computed) | yes | yes | ok |

### Existing in both: image (docker_image ↔ podman_image)

Top-level:

| Attribute | docker | podman | Status |
| --- | --- | --- | --- |
| name | yes | yes | ok |
| keep_locally | yes | yes | ok |
| pull_triggers | yes | yes | ok |
| force_remove | yes | yes | ok |
| triggers | yes | yes | ok |
| platform | yes | yes | ok |
| build | yes | yes | ok (but contents differ — see below) |
| image_id (computed) | yes | yes | ok |
| repo_digest (computed) | yes | yes | ok |

Inside the `build` block:

| Attribute | docker | podman | Status |
| --- | --- | --- | --- |
| context | yes | yes | ok |
| dockerfile | yes | yes | ok |
| build_args | yes | yes | ok |
| cache_from | yes | yes | ok |
| cache_to | yes | no | **add** (buildah/podman build supports `--cache-to`). |
| force_remove | yes | yes | ok |
| labels | yes | yes | ok |
| label (map alt) | yes | no | **add** (docker provider accepts both `label` and `labels`; keep `labels`, also expose `label` alias for compatibility). |
| no_cache | yes | yes | ok |
| platform | yes | yes | ok |
| remove | yes | yes | ok |
| tag | yes | yes | ok |
| target | yes | yes | ok |
| network_mode | yes | yes | ok |
| extra_hosts | yes | yes | ok |
| shm_size | yes | yes | ok |
| cpu_period | yes | yes | ok |
| cpu_quota | yes | yes | ok |
| cpu_set_cpus | yes | yes | ok |
| cpu_set_mems | yes | no | **add** (NUMA mem nodes; libpod build supports). |
| cpu_shares | yes | yes | ok |
| memory | yes | yes | ok |
| memory_swap | yes | yes | ok |
| cgroup_parent | yes | no | **add** |
| ulimit (name/hard/soft) | yes | no | **add** (build-time ulimits; libpod build supports). |
| auth_config (host_name/user_name/password/auth/email/server_address/identity_token/registry_token) | yes | no | **add** (per-build registry auth — useful when pulling base image from auth'd registry). |
| secrets (id/src/env) | yes | no | **add** (buildah supports build-time secrets via `--secret`). |
| additional_contexts | yes | no | **add** (buildkit-style named contexts; buildah recent versions support). Mark **investigate** if podman version targeted lacks it. |
| isolation | yes | no | **skip** (Windows containers concept). |
| pull_parent | yes | no | **add** (`--pull` flag on build). |
| squash | yes | no | **add** (`buildah --squash`). |
| remote_context | yes | no | **investigate** (URL/Git build context; podman build accepts URL). |
| security_opt | yes | no | **add** (build-time security opts; libpod build supports). |
| session_id | yes | no | **skip** (BuildKit session; podman uses buildah, no equivalent). |
| version | yes | no | **skip** (BuildKit vs legacy builder selector — docker-specific). |
| build_id | yes | no | **skip** (docker BuildKit cancel ID). |
| use_legacy_builder | yes | no | **skip** (BuildKit toggle). |
| builder | yes | no | **skip** (BuildKit named builder; buildah has no equivalent). |
| build_log_file | yes | no | **add** (write build output to file — useful, easy). |
| provenance | yes | no | **add** (buildah supports SLSA provenance attestation). |
| sbom | yes | no | **add** (buildah supports SBOM attestation). |
| suppress_output | yes | no | **add** (cosmetic; suppress build stream). |

### Existing in both: network (docker_network ↔ podman_network)

| Attribute | docker | podman | Status |
| --- | --- | --- | --- |
| name | yes | yes | ok |
| labels | yes | yes | ok |
| driver | yes | yes | ok |
| options | yes | yes | ok |
| internal | yes | yes | ok |
| attachable | yes | yes | ok |
| ingress | yes | yes | ok (swarm-only on docker; harmless on podman — keep) |
| ipv6 | yes | yes | ok |
| ipam_driver | yes | yes | ok |
| ipam_options | yes | yes | ok |
| ipam_config.subnet | yes | yes | ok |
| ipam_config.ip_range | yes | yes | ok |
| ipam_config.gateway | yes | yes | ok |
| ipam_config.aux_address | yes | yes | ok |
| scope (computed) | yes | yes | ok |
| id (computed) | yes | yes | ok |

(No gaps.)

### Existing in both: volume (docker_volume ↔ podman_volume)

| Attribute | docker | podman | Status |
| --- | --- | --- | --- |
| name | yes | yes | ok |
| labels | yes | yes | ok |
| driver | yes | yes | ok |
| driver_opts | yes | yes | ok |
| mountpoint (computed) | yes | yes | ok |
| cluster (block: scope, sharing, type, topology_preferred, topology_required, required_bytes, limit_bytes, group, availability) | yes | no | **skip** (Docker Swarm cluster volumes; not supported by podman). |

### Existing in both: secret (docker_secret ↔ podman_secret)

| Attribute | docker | podman | Status |
| --- | --- | --- | --- |
| name | yes | yes | ok |
| data | yes | yes | ok |
| labels | yes | yes | ok |

Note: docker_secret is swarm-only, but podman has its own native secret store
(`podman secret`) which is what podman_secret already targets. Parity is fine.
Implementation should make sure `podman_secret` does **not** require swarm —
review current implementation: it currently uses `swarm.SecretSpec` via the
docker SDK — this likely works on the compat API but should be tested against
non-swarm podman. **investigate** runtime behavior.

### Existing in both: plugin (docker_plugin ↔ podman_plugin)

| Attribute | docker | podman | Status |
| --- | --- | --- | --- |
| name | yes | yes | ok |
| alias | yes | yes | ok |
| enabled | yes | yes | ok |
| grant_all_permissions | yes | yes | ok |
| grant_permissions (name/value) | yes | yes | ok |
| env | yes | yes | ok |
| force_destroy | yes | yes | ok |
| force_disable | yes | yes | ok |
| enable_timeout | yes | yes | ok |
| plugin_reference (computed) | yes | yes | ok |

Note: podman does not actually implement Docker's plugin system natively
(podman has no `podman plugin` command). The compat API may return errors.
**investigate** whether to keep this resource or mark it as "best-effort".

### Existing in both: registry_image (docker_registry_image ↔ podman_registry_image)

| Attribute | docker | podman | Status |
| --- | --- | --- | --- |
| name | yes | yes | ok |
| keep_remotely | yes | yes | ok |
| insecure_skip_verify | yes | yes | ok |
| triggers | yes | yes | ok |
| sha256_digest (computed) | yes | yes | ok |
| auth_config (host_name/user_name/password/auth/email/server_address/identity_token/registry_token) | yes | no | **add** (per-resource registry credentials override). |
| build (full image-build schema) | yes | no | **add** (docker_registry_image can build+push in one step; useful — podman already has a build path in resource_image, so wire it in here too). |

### Existing in both: tag (docker_tag ↔ podman_tag)

| Attribute | docker | podman | Status |
| --- | --- | --- | --- |
| source_image | yes | yes | ok |
| target_image | yes | yes | ok |
| source_image_id (computed) | yes | yes | ok |
| tag_triggers | yes | no | **add** (trivial; re-tag when a referenced trigger changes). |

### docker_service
**SKIP** — Docker Swarm services. Podman has no swarm mode equivalent.

### docker_config
**SKIP** — Docker Swarm configs. Podman has no swarm equivalent. (Podman
secrets cover the analogous use case and are already implemented.)

### docker_buildx_builder
**SKIP** — Docker buildx-specific (BuildKit). Podman uses buildah; no
analogous "named builder instance" concept. If we ever want a builder
abstraction it would be a separate podman-native design.

---

## Data sources

### Existing in both: image (data.docker_image ↔ data.podman_image)

| Attribute | docker | podman | Status |
| --- | --- | --- | --- |
| name | yes | yes | ok |
| repo_digest (computed) | yes | yes | ok |
| image_id / id (computed) | (id via SetId) | yes | ok |

(No gaps.)

### Existing in both: network (data.docker_network ↔ data.podman_network)

| Attribute | docker | podman | Status |
| --- | --- | --- | --- |
| name | yes | yes | ok |
| driver (computed) | yes | yes | ok |
| options (computed) | yes | yes | ok |
| internal (computed) | yes | yes | ok |
| scope (computed) | yes | yes | ok |
| ipam_config.* (computed) | yes | yes | ok |
| containers (computed: container_id, name, endpoint_id, mac_address, ipv4_address, ipv6_address) | yes | no | **add** (list of attached containers — libpod network inspect returns this). |

### Existing in both: plugin (data.docker_plugin ↔ data.podman_plugin)

| Attribute | docker | podman | Status |
| --- | --- | --- | --- |
| id | yes | yes | ok |
| alias | yes | yes | ok |
| name (computed) | yes | yes | ok |
| plugin_reference (computed) | yes | yes | ok |
| enabled (computed) | yes | yes | ok |
| grant_all_permissions (computed) | yes | yes | ok |
| env (computed) | yes | yes | ok |

(No gaps. Same caveat as resource_plugin about podman support.)

### Existing in both: registry_image (data.docker_registry_image ↔ data.podman_registry_image)

| Attribute | docker | podman | Status |
| --- | --- | --- | --- |
| name | yes | yes | ok |
| insecure_skip_verify | yes | yes | ok |
| sha256_digest (computed) | yes | yes | ok |
| auth_config (block) | (no, only on manifests data source) | no | n/a |

(No gaps.)

### Existing in both: registry_image_manifests (data.docker_registry_image_manifests ↔ data.podman_registry_image_manifests)

| Attribute | docker | podman | Status |
| --- | --- | --- | --- |
| name | yes | yes | ok |
| insecure_skip_verify | yes | yes | ok |
| auth_config.address | yes | yes | ok |
| auth_config.username | yes | yes | ok |
| auth_config.password | yes | yes | ok |
| manifests.media_type | yes | yes | ok |
| manifests.sha256_digest | yes | yes | ok |
| manifests.architecture | yes | yes | ok |
| manifests.os | yes | yes | ok |

(No gaps.)

### Existing in both: logs (data.docker_logs ↔ data.podman_logs)

| Attribute | docker | podman | Status |
| --- | --- | --- | --- |
| name | yes | yes | ok |
| logs_list_string_enabled | yes | yes | ok |
| discard_headers | yes | yes | ok |
| show_stdout | yes | yes | ok |
| show_stderr | yes | yes | ok |
| since | yes | yes | ok |
| until | yes | yes | ok |
| timestamps | yes | yes | ok |
| follow | yes | yes | ok |
| tail | yes | yes | ok |
| details | yes | yes | ok |
| logs_list_string (computed) | yes | yes | ok |

(No gaps.)

### data.docker_registry_image_tags

**Not in podman.** docker provides this data source (lists all tags of a repo
from the registry).
| Attribute | docker | podman | Status |
| --- | --- | --- | --- |
| name | yes | no | **add** (new `data.podman_registry_image_tags` resource; calls registry `/v2/<repo>/tags/list`). |
| insecure_skip_verify | yes | no | **add** |
| strict_semver | yes | no | **add** (filter to semver tags only). |
| tags (computed) | yes | no | **add** |

### data.docker_containers (framework)

**Not in podman.** docker provides a data source that lists running/all
containers with id/names/image/image_id/command/created/state/status/labels.
**add** as `data.podman_containers` (libpod and compat both expose
`/containers/json`).

---

## Summary of gaps to implement (add)

### Provider
- `disable_podman_daemon_check` (skip ping on configure)
- `context` (investigate — map to podman connection name, or skip)

### podman_container — missing attributes
- `platform`
- `mounts.volume_options.subpath`
- `volumes.selinux_relabel` (HIGH priority for RHEL/Fedora users)
- `device_read_bps` (block: path, rate)
- `device_read_iops` (block: path, rate)
- `device_write_bps` (block: path, rate)
- `device_write_iops` (block: path, rate)
- `device_requests` (block: driver, count, device_ids, capabilities, options) — investigate libpod mapping
- `memory_reservation`
- `cpus` (string)
- `cgroup_parent`
- `networks_advanced.link_local_ips`
- `networks_advanced.mac_address`
- `networks_advanced.driver_opts`
- `networks_advanced.gw_priority` (investigate)
- `upload.permissions` (explicit mode override)
- `healthcheck.start_interval`
- `container_read_refresh_timeout_milliseconds`
- `gpus` (string, convenience)

### podman_image — missing build sub-attributes
- `build.cache_to`
- `build.label` (map alias for labels)
- `build.cpu_set_mems`
- `build.cgroup_parent`
- `build.ulimit` (block)
- `build.auth_config` (block: host_name/user_name/password/auth/email/server_address/identity_token/registry_token)
- `build.secrets` (block: id/src/env)
- `build.additional_contexts` (investigate)
- `build.pull_parent`
- `build.squash`
- `build.remote_context` (investigate — accepts URL/git)
- `build.security_opt`
- `build.build_log_file`
- `build.provenance`
- `build.sbom`
- `build.suppress_output`

### podman_registry_image — missing attributes
- `auth_config` (block: host_name/user_name/password/auth/email/server_address/identity_token/registry_token)
- `build` (full nested build schema — same as podman_image.build, to allow build+push)

### podman_tag — missing attributes
- `tag_triggers`

### New data sources to add
- `data.podman_registry_image_tags` (name, insecure_skip_verify, strict_semver, tags)
- `data.podman_containers` (computed list with id/names/image/image_id/command/created/state/status/labels)

### data.podman_network — missing attributes
- `containers` (computed list: container_id, name, endpoint_id, mac_address, ipv4_address, ipv6_address)

---

## Summary of gaps to SKIP

### Whole resources
- `docker_service` — Swarm only.
- `docker_config` — Swarm only (podman_secret covers the analogous case).
- `docker_buildx_builder` — BuildKit/buildx-specific. Podman uses buildah.
- `docker_compose` — Out of scope (compose orchestration is a separate tool;
  podman has `podman-compose` / `podman kube` as separate UX).

### Within docker_container
(none — see "investigate" entries below for borderline cases)

### Within docker_image build block
- `build.isolation` — Windows containers concept.
- `build.session_id` — BuildKit session ID.
- `build.version` — BuildKit-vs-legacy selector (docker-specific).
- `build.build_id` — BuildKit cancel ID.
- `build.use_legacy_builder` — BuildKit toggle.
- `build.builder` — BuildKit named builder (no buildah equivalent).

### Within docker_volume
- `cluster` block — Swarm cluster volumes (CSI in swarm). Not supported by podman.

---

## Items to investigate before implementation

- **provider.context**: how to map docker context → podman connection (`podman system connection`). May be skipped if it adds little value vs. the existing `host` argument.
- **container.device_requests**: libpod has a `DeviceRequests` field in container create spec (docker-compat); confirm field-for-field mapping before exposing the same nested schema, otherwise just expose `gpus`.
- **container.networks_advanced.gw_priority**: newer docker engine field — confirm libpod 4.x/5.x supports it.
- **image.build.additional_contexts**: requires recent buildah; confirm minimum podman version we want to target.
- **image.build.remote_context**: podman build accepts URL/git contexts via CLI; confirm REST API exposes it.
- **resource_secret** runtime: current implementation calls `SecretCreate` with a `swarm.SecretSpec`. Podman's compat API accepts this and writes into the native secret store, but confirm via acceptance test on a podman socket with swarm disabled (default).
- **resource_plugin** runtime: podman has no plugin manager. The compat `/plugins` endpoints likely 404 or return empty. Decide whether to keep the resource as a non-functional shim, gate it on a feature detect, or remove it.
- **data.podman_containers**: choose between schema-v2 framework style (matches docker) or SDKv2 to keep code uniform with the rest of the provider.
