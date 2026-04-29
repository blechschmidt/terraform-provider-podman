---
page_title: "podman_container Resource - terraform-provider-podman"
subcategory: ""
description: |-
  Manages the lifecycle of a Podman container.
---

# podman_container (Resource)

Manages the lifecycle of a Podman container.

## Example Usage

### Basic Container

```terraform
resource "podman_image" "ubuntu" {
  name = "docker.io/library/ubuntu:latest"
}

resource "podman_container" "basic" {
  name  = "my-container"
  image = podman_image.ubuntu.image_id

  command = ["sleep", "infinity"]
}
```

### Container with Ports and Environment Variables

```terraform
resource "podman_image" "nginx" {
  name = "docker.io/library/nginx:latest"
}

resource "podman_container" "web" {
  name  = "web-server"
  image = podman_image.nginx.image_id

  env = [
    "NGINX_HOST=example.com",
    "NGINX_PORT=80",
  ]

  ports {
    internal = 80
    external = 8080
    protocol = "tcp"
  }

  ports {
    internal = 443
    external = 8443
  }

  restart = "unless-stopped"
}
```

### Container with Volumes

```terraform
resource "podman_volume" "data" {
  name = "app_data"
}

resource "podman_container" "with_volumes" {
  name  = "app"
  image = podman_image.ubuntu.image_id

  volumes {
    volume_name    = podman_volume.data.name
    container_path = "/data"
  }

  volumes {
    host_path      = "/var/log/app"
    container_path = "/var/log/app"
    read_only      = true
  }

  command = ["sleep", "infinity"]
}
```

### Container with Mounts

```terraform
resource "podman_container" "with_mounts" {
  name  = "app-mounts"
  image = podman_image.ubuntu.image_id

  mounts {
    target = "/data"
    type   = "volume"
    source = "my-volume"

    volume_options {
      driver_name = "local"
    }
  }

  mounts {
    target = "/config"
    type   = "bind"
    source = "/etc/app/config"

    bind_options {
      propagation = "rprivate"
    }
  }

  mounts {
    target = "/tmp/cache"
    type   = "tmpfs"

    tmpfs_options {
      size_bytes = 67108864
      mode       = 1777
    }
  }

  command = ["sleep", "infinity"]
}
```

### Container with Networking

```terraform
resource "podman_network" "app_net" {
  name = "app-network"
}

resource "podman_container" "with_network" {
  name  = "networked-app"
  image = podman_image.nginx.image_id

  networks_advanced {
    name         = podman_network.app_net.name
    aliases      = ["web", "frontend"]
    ipv4_address = "10.89.0.10"
  }

  hostname = "web.example.local"

  dns        = ["8.8.8.8", "8.8.4.4"]
  dns_search = ["example.local"]

  host {
    host = "api.internal"
    ip   = "10.89.0.20"
  }
}
```

### Container with Healthcheck

```terraform
resource "podman_container" "with_healthcheck" {
  name  = "healthy-app"
  image = podman_image.nginx.image_id

  healthcheck {
    test     = ["CMD-SHELL", "curl -f http://localhost/ || exit 1"]
    interval = "30s"
    timeout  = "10s"
    retries  = 3
  }

  ports {
    internal = 80
    external = 8080
  }

  restart = "on-failure"
}
```

### Container with File Upload

```terraform
resource "podman_container" "with_upload" {
  name  = "configured-app"
  image = podman_image.nginx.image_id

  upload {
    file    = "/etc/nginx/conf.d/default.conf"
    content = <<-EOT
      server {
        listen 80;
        server_name example.com;
        location / {
          root /usr/share/nginx/html;
        }
      }
    EOT
  }

  upload {
    file           = "/usr/share/nginx/html/index.html"
    content_base64 = base64encode("<h1>Hello World</h1>")
  }

  ports {
    internal = 80
    external = 8080
  }
}
```

### Container from a Rootfs Directory

Instead of an image, the container can be created from an exploded
rootfs on the host. This mirrors the `podman run --rootfs` flag and is
useful for chrooting into a directory tree, running OCI bundles you
unpacked yourself, or running containers without pulling an image.

`image` and `rootfs` are mutually exclusive — exactly one must be set.

```terraform
resource "podman_container" "from_rootfs" {
  name   = "alpine-rootfs"
  rootfs = "/var/lib/rootfs/alpine"

  command = ["/bin/sh", "-c", "echo hello && sleep infinity"]
}
```

Use `rootfs_overlay = true` to mount the rootfs as a read-only overlay
(equivalent to `--rootfs /path:O`), so changes the container makes are
discarded on stop:

```terraform
resource "podman_container" "overlay_rootfs" {
  name           = "ephemeral-rootfs"
  rootfs         = "/var/lib/rootfs/alpine"
  rootfs_overlay = true

  command = ["/bin/sh"]
}
```

Because rootfs containers are created via Podman's native libpod API
rather than the Docker compatibility API, a few image-only options are
silently ignored when `rootfs` is set: `healthcheck`, `runtime`,
`cgroupns_mode`, `pid_mode`, `ipc_mode`, `userns_mode`, `storage_opts`,
`tmpfs`, `ulimit`, `domainname`, and `devices`. Most other options
(command, env, labels, mounts, volumes, ports, capabilities, dns, host,
network_mode, networks_advanced, sysctls, security_opts, log_driver,
log_opts, restart, privileged, read_only, init, shm_size,
stop_signal/stop_timeout, tty, stdin_open, user, working_dir, hostname,
group_add, rm) work the same as in image-based containers.

### Container with Resource Limits

```terraform
resource "podman_container" "with_limits" {
  name  = "limited-app"
  image = podman_image.ubuntu.image_id

  memory      = 536870912  # 512 MB
  memory_swap = 1073741824 # 1 GB
  cpu_shares  = 512
  cpu_set     = "0,1"
  cpu_period  = 100000
  cpu_quota   = 50000
  shm_size    = 67108864 # 64 MB

  ulimit {
    name = "nofile"
    soft = 65536
    hard = 65536
  }

  ulimit {
    name = "nproc"
    soft = 4096
    hard = 8192
  }

  command = ["sleep", "infinity"]
}
```

## Schema

### Required

- `name` (String) The name of the container. Forces recreation if changed.

Exactly one of `image` or `rootfs` must be set.

### Optional

- `attach` (Boolean) Whether to attach to the container to collect its stdout and stderr output. Defaults to `false`.
- `capabilities` (Block List, Max: 1) Add or drop Linux capabilities for the container. Forces recreation if changed. See [capabilities](#nestedblock--capabilities) below.
- `cgroupns_mode` (String) Cgroup namespace mode to use for the container. Forces recreation if changed.
- `command` (List of String) The command to run in the container. Forces recreation if changed.
- `cpu_period` (Number) The length of a CPU CFS period in microseconds.
- `cpu_quota` (Number) The CPU CFS quota in microseconds.
- `cpu_set` (String) CPUs in which to allow execution (e.g. `0-3`, `0,1`).
- `cpu_shares` (Number) CPU shares (relative weight) for the container.
- `destroy_grace_seconds` (Number) The number of seconds to wait before killing the container during a destroy operation.
- `devices` (Block Set) Bind host devices to the container. Forces recreation if changed. See [devices](#nestedblock--devices) below.
- `dns` (Set of String) DNS servers for the container. Forces recreation if changed.
- `dns_opts` (Set of String) DNS options for the container. Forces recreation if changed.
- `dns_search` (Set of String) DNS search domains for the container. Forces recreation if changed.
- `domainname` (String) The domain name of the container. Forces recreation if changed.
- `entrypoint` (List of String) The entrypoint for the container. Forces recreation if changed.
- `env` (Set of String) Environment variables to set in the container, in the form `KEY=VALUE`.
- `group_add` (Set of String) Additional groups for the container process.
- `healthcheck` (Block List, Max: 1) Healthcheck configuration for the container. See [healthcheck](#nestedblock--healthcheck) below.
- `host` (Block Set) Additional entries to add to the container's `/etc/hosts` file. Forces recreation if changed. See [host](#nestedblock--host) below.
- `hostname` (String) The hostname of the container. Forces recreation if changed.
- `image` (String) The ID or name of the image to use for the container. Forces recreation if changed. Mutually exclusive with `rootfs`.
- `init` (Boolean) Whether to run an init process inside the container that forwards signals and reaps processes.
- `ipc_mode` (String) IPC namespace mode for the container. Forces recreation if changed.
- `labels` (Block Set) Labels to apply to the container. See [labels](#nestedblock--labels) below.
- `log_driver` (String) The logging driver to use for the container (e.g. `journald`, `k8s-file`).
- `log_opts` (Map of String) Options for the logging driver.
- `logs` (Boolean) Whether to capture the container's stdout and stderr logs in the Terraform state. Defaults to `false`.
- `max_retry_count` (Number) The maximum number of restart retries when the restart policy is `on-failure`.
- `memory` (Number) The memory limit for the container in bytes.
- `memory_swap` (Number) The total memory limit (memory + swap) for the container in bytes. Set to `-1` for unlimited swap.
- `mounts` (Block Set) Mount specification for the container. Forces recreation if changed. See [mounts](#nestedblock--mounts) below.
- `must_run` (Boolean) If `true`, the Terraform resource will be marked as tainted if the container stops. Defaults to `true`.
- `network_mode` (String) The network mode for the container (e.g. `bridge`, `host`, `none`, `slirp4netns`). Forces recreation if changed.
- `networks_advanced` (Block Set) Advanced network configuration for the container. See [networks_advanced](#nestedblock--networks_advanced) below.
- `pid_mode` (String) PID namespace mode for the container. Forces recreation if changed.
- `ports` (Block List) Port mappings for the container. Forces recreation if changed. See [ports](#nestedblock--ports) below.
- `privileged` (Boolean) Whether to run the container in privileged mode. Defaults to `false`. Forces recreation if changed.
- `publish_all_ports` (Boolean) Whether to publish all exposed ports to random host ports. Defaults to `false`.
- `read_only` (Boolean) Whether to mount the container's root filesystem as read-only. Defaults to `false`.
- `remove_volumes` (Boolean) Whether to remove anonymous volumes associated with the container on destroy. Defaults to `true`.
- `restart` (String) The restart policy for the container. One of `no`, `on-failure`, `always`, or `unless-stopped`. Defaults to `no`.
- `rm` (Boolean) Whether to automatically remove the container when it stops. Defaults to `false`.
- `rootfs` (String) Path on the host to an exploded container rootfs to use instead of an image. Equivalent to `podman run --rootfs`. Forces recreation if changed. Mutually exclusive with `image`. See [rootfs](#container-from-a-rootfs-directory) below.
- `rootfs_mapping` (String) UID/GID idmap specification for the rootfs (e.g. `idmap` or `idmap=uids=0-1-10;gids=0-1-10`). Equivalent to the `:idmap` modifier of `podman run --rootfs`. Forces recreation if changed. Requires `rootfs`.
- `rootfs_overlay` (Boolean) Mount the rootfs as a read-only overlay so container writes are discarded on stop. Equivalent to the `:O` modifier of `podman run --rootfs`. Defaults to `false`. Forces recreation if changed. Requires `rootfs`.
- `runtime` (String) The OCI runtime to use for the container. Forces recreation if changed.
- `security_opts` (Set of String) Security options for the container. Forces recreation if changed.
- `shm_size` (Number) The size of `/dev/shm` in bytes. Forces recreation if changed.
- `start` (Boolean) Whether to start the container after creation. Defaults to `true`.
- `stdin_open` (Boolean) Whether to keep stdin open for the container even when not attached. Defaults to `false`.
- `stop_signal` (String) The signal to use for stopping the container (e.g. `SIGTERM`).
- `stop_timeout` (Number) The number of seconds to wait for the container to stop gracefully before sending `SIGKILL`.
- `storage_opts` (Map of String) Storage driver options for the container.
- `sysctls` (Map of String) Sysctl settings to apply to the container.
- `tmpfs` (Map of String) A map of tmpfs mounts to add to the container, where the key is the mount path and the value is the mount options.
- `tty` (Boolean) Whether to allocate a pseudo-TTY for the container. Defaults to `false`.
- `ulimit` (Block Set) Resource limits to set in the container. See [ulimit](#nestedblock--ulimit) below.
- `upload` (Block Set) Files or content to upload to the container before it starts. Forces recreation if changed. See [upload](#nestedblock--upload) below.
- `user` (String) The user to run the container process as (e.g. `root`, `1000:1000`). Forces recreation if changed.
- `userns_mode` (String) User namespace mode for the container. Forces recreation if changed.
- `volumes` (Block Set) Volume bindings for the container. Forces recreation if changed. See [volumes](#nestedblock--volumes) below.
- `wait` (Boolean) Whether to wait for the container to finish and capture its exit code. Defaults to `false`.
- `wait_timeout` (Number) The maximum number of seconds to wait for the container to finish when `wait` is enabled. Defaults to `60`.
- `working_dir` (String) The working directory inside the container. Forces recreation if changed.

### Read-Only

- `bridge` (String) The network bridge address of the container.
- `container_logs` (String) The logs of the container. Only populated when `logs` is set to `true`.
- `exit_code` (Number) The exit code of the container. Only populated when `wait` is set to `true` or the container has stopped.
- `id` (String) The ID of the container.
- `network_data` (List of Object) The network data of the container. See [network_data](#nestedatt--network_data) below.

<a id="nestedblock--capabilities"></a>

### Nested Schema for `capabilities`

Optional:

- `add` (Set of String) A set of Linux capabilities to add to the container (e.g. `NET_ADMIN`, `SYS_PTRACE`).
- `drop` (Set of String) A set of Linux capabilities to drop from the container (e.g. `ALL`, `NET_RAW`).

<a id="nestedblock--devices"></a>

### Nested Schema for `devices`

Required:

- `host_path` (String) The path to the device on the host.

Optional:

- `container_path` (String) The path to the device inside the container. Defaults to the `host_path`.
- `permissions` (String) The cgroup permissions for the device. Defaults to `rwm`.

<a id="nestedblock--healthcheck"></a>

### Nested Schema for `healthcheck`

Required:

- `test` (List of String) The command to run to check the health of the container. Use `CMD` or `CMD-SHELL` as the first element.

Optional:

- `interval` (String) The time between running the healthcheck. Defaults to `0s`.
- `retries` (Number) The number of consecutive failures needed to report unhealthy. Defaults to `0`.
- `start_period` (String) The grace period to allow the container to start before counting healthcheck failures. Defaults to `0s`.
- `timeout` (String) The maximum time to allow a healthcheck to complete. Defaults to `0s`.

<a id="nestedblock--host"></a>

### Nested Schema for `host`

Required:

- `host` (String) The hostname to add.
- `ip` (String) The IP address to map to the hostname.

<a id="nestedblock--labels"></a>

### Nested Schema for `labels`

Required:

- `label` (String) The label key.
- `value` (String) The label value.

<a id="nestedblock--mounts"></a>

### Nested Schema for `mounts`

Required:

- `target` (String) The path inside the container where the mount is placed.
- `type` (String) The mount type. One of `bind`, `volume`, or `tmpfs`.

Optional:

- `bind_options` (Block List, Max: 1) Options specific to bind mounts. See [bind_options](#nestedblock--mounts--bind_options) below.
- `read_only` (Boolean) Whether the mount is read-only.
- `source` (String) The source of the mount. For `bind` mounts this is a path on the host. For `volume` mounts this is the volume name.
- `tmpfs_options` (Block List, Max: 1) Options specific to tmpfs mounts. See [tmpfs_options](#nestedblock--mounts--tmpfs_options) below.
- `volume_options` (Block List, Max: 1) Options specific to volume mounts. See [volume_options](#nestedblock--mounts--volume_options) below.

<a id="nestedblock--mounts--bind_options"></a>

### Nested Schema for `mounts.bind_options`

Optional:

- `propagation` (String) The bind propagation type (e.g. `rprivate`, `rshared`, `rslave`).

<a id="nestedblock--mounts--tmpfs_options"></a>

### Nested Schema for `mounts.tmpfs_options`

Optional:

- `mode` (Number) The file mode for the tmpfs mount.
- `size_bytes` (Number) The size of the tmpfs mount in bytes.

<a id="nestedblock--mounts--volume_options"></a>

### Nested Schema for `mounts.volume_options`

Optional:

- `driver_name` (String) The name of the volume driver.
- `driver_options` (Map of String) Options for the volume driver.
- `labels` (Block Set) Labels to apply to the volume.
- `no_copy` (Boolean) Whether to disable copying files from the container path when the volume is empty.

<a id="nestedblock--networks_advanced"></a>

### Nested Schema for `networks_advanced`

Required:

- `name` (String) The name of the network to connect to.

Optional:

- `aliases` (Set of String) Network-scoped aliases for the container.
- `ipv4_address` (String) The static IPv4 address for the container on this network.
- `ipv6_address` (String) The static IPv6 address for the container on this network.

<a id="nestedblock--ports"></a>

### Nested Schema for `ports`

Required:

- `internal` (Number) The port inside the container.

Optional:

- `external` (Number) The port on the host.
- `ip` (String) The host IP address to bind the port to. Defaults to `0.0.0.0`.
- `protocol` (String) The protocol for the port mapping. Defaults to `tcp`.

<a id="nestedblock--ulimit"></a>

### Nested Schema for `ulimit`

Required:

- `hard` (Number) The hard limit value.
- `name` (String) The name of the limit (e.g. `nofile`, `nproc`).
- `soft` (Number) The soft limit value.

<a id="nestedblock--upload"></a>

### Nested Schema for `upload`

Required:

- `file` (String) The path inside the container to upload the file to.

Optional:

- `content` (String) Literal string content to upload.
- `content_base64` (String) Base64-encoded content to upload. Conflicts with `content`.
- `executable` (Boolean) Whether to set the uploaded file as executable. Defaults to `false`.
- `source` (String) Path to a local file to upload.
- `source_hash` (String) A hash of the source file to detect content changes.

<a id="nestedblock--volumes"></a>

### Nested Schema for `volumes`

Optional:

- `container_path` (String) The path inside the container where the volume is mounted.
- `from_container` (String) The name or ID of another container to mount volumes from.
- `host_path` (String) The path on the host to bind mount into the container.
- `read_only` (Boolean) Whether the volume is mounted read-only.
- `volume_name` (String) The name of a Podman volume to mount.

<a id="nestedatt--network_data"></a>

### Nested Schema for `network_data`

Read-Only:

- `gateway` (String) The gateway address of the network.
- `global_ipv6_address` (String) The global IPv6 address of the container on this network.
- `global_ipv6_prefix_length` (Number) The IPv6 prefix length.
- `ip_address` (String) The IPv4 address of the container on this network.
- `ip_prefix_length` (Number) The IPv4 prefix length.
- `ipv6_gateway` (String) The IPv6 gateway address.
- `mac_address` (String) The MAC address of the container on this network.
- `network_name` (String) The name of the network.

## Import

Import is supported using the container ID.

```shell
terraform import podman_container.example CONTAINER_ID
```
