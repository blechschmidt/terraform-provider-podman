---
page_title: "podman_secret Resource - terraform-provider-podman"
subcategory: ""
description: |-
  Manages the lifecycle of a Podman secret.
---

# podman_secret (Resource)

Manages the lifecycle of a Podman secret.

## Example Usage

```terraform
resource "podman_secret" "db_credentials" {
  name = "db_password"
  data = var.db_password

  labels {
    label = "application"
    value = "my-app"
  }
}
```

## Argument Reference

The following arguments are supported:

- `name` - (Required, String) The name of the secret. Changing this forces recreation of the resource.
- `data` - (Required, Sensitive, String) The secret data. Changing this forces recreation of the resource.
- `labels` - (Optional, Block Set) A set of labels to apply to the secret. Changing this forces recreation of the resource. See [labels](#labels) below for details.

### labels

- `label` - (Required, String) The name of the label.
- `value` - (Required, String) The value of the label.

## Attribute Reference

In addition to all arguments above, the following attributes are exported:

- `id` - (String) The ID of the secret.

## Import

Podman secrets can be imported using the secret ID:

```shell
terraform import podman_secret.db_credentials SECRET_ID
```
