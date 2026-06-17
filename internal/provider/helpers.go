package provider

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/docker/docker/api/types/registry"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

// getRegistryAuth returns the base64-encoded auth string for a given image name.
func getRegistryAuth(config *ProviderConfig, imageName string) (string, error) {
	registryHost := getRegistryFromImageName(imageName)

	if auth, ok := config.RegistryAuth[registryHost]; ok {
		if auth.AuthDisabled {
			return "", nil
		}
		if auth.Username != "" && auth.Password != "" {
			authConfig := registry.AuthConfig{
				Username: auth.Username,
				Password: auth.Password,
			}
			encoded, err := json.Marshal(authConfig)
			if err != nil {
				return "", fmt.Errorf("failed to marshal auth config: %w", err)
			}
			return base64.URLEncoding.EncodeToString(encoded), nil
		}
	}

	return "", nil
}

// getRegistryFromImageName extracts the registry host from an image name.
func getRegistryFromImageName(imageName string) string {
	parts := strings.SplitN(imageName, "/", 2)
	if len(parts) == 1 {
		return "docker.io"
	}
	if strings.Contains(parts[0], ".") || strings.Contains(parts[0], ":") || parts[0] == "localhost" {
		return parts[0]
	}
	return "docker.io"
}

// suppressIfIDOrNameEqual is a DiffSuppressFunc that suppresses diffs between the
// different spellings of the *same* image ID (e.g. "sha256:abc123…" vs the bare or
// truncated "abc123…"). It must NOT suppress diffs between two distinct image names or
// tags such as "alpine" -> "alpine:3.19" or "nginx" -> "nginx-unprivileged": those are
// real changes, and because the image field is ForceNew they have to recreate the
// container.
func suppressIfIDOrNameEqual(_, old, new string, _ *schema.ResourceData) bool {
	if old == new {
		return true
	}
	if old == "" || new == "" {
		return false
	}
	// Collapse the sha256:/short-ID spellings of the same image ID, but only when both
	// values actually look like hex image IDs. Restricting the prefix match to IDs keeps
	// human-readable image names that merely share a prefix from being mistaken for the
	// same image (and thus silently skipping the required recreation).
	oldClean := strings.TrimPrefix(old, "sha256:")
	newClean := strings.TrimPrefix(new, "sha256:")
	if isImageID(oldClean) && isImageID(newClean) &&
		(strings.HasPrefix(oldClean, newClean) || strings.HasPrefix(newClean, oldClean)) {
		return true
	}
	return false
}

// isImageID reports whether s looks like a (possibly truncated) container image ID: a
// lowercase hex string of at least 12 characters, the minimum length Podman and Docker
// use for a short ID. Image names and tags always contain non-hex characters (':', '/',
// or letters beyond a-f), so they are never mistaken for an ID.
func isImageID(s string) bool {
	if len(s) < 12 {
		return false
	}
	for _, r := range s {
		if !((r >= '0' && r <= '9') || (r >= 'a' && r <= 'f')) {
			return false
		}
	}
	return true
}

// mapStringInterfaceToStringString converts map[string]interface{} to map[string]string.
func mapStringInterfaceToStringString(m map[string]interface{}) map[string]string {
	result := make(map[string]string)
	for k, v := range m {
		result[k] = v.(string)
	}
	return result
}

// stringListToSlice converts a TypeList of strings to []string.
func stringListToSlice(v interface{}) []string {
	raw := v.([]interface{})
	result := make([]string, len(raw))
	for i, val := range raw {
		result[i] = val.(string)
	}
	return result
}

// stringSetToSlice converts a TypeSet of strings to []string.
func stringSetToSlice(v interface{}) []string {
	raw := v.(*schema.Set).List()
	result := make([]string, len(raw))
	for i, val := range raw {
		result[i] = val.(string)
	}
	return result
}

// labelsSchema returns the schema for the labels block used by multiple resources.
func labelsSchema() *schema.Schema {
	return &schema.Schema{
		Type:     schema.TypeSet,
		Optional: true,
		Elem: &schema.Resource{
			Schema: map[string]*schema.Schema{
				"label": {
					Type:     schema.TypeString,
					Required: true,
				},
				"value": {
					Type:     schema.TypeString,
					Required: true,
				},
			},
		},
	}
}

// labelsToMap converts the labels set from the schema to a map[string]string.
func labelsToMap(v interface{}) map[string]string {
	labels := make(map[string]string)
	if v == nil {
		return labels
	}
	for _, raw := range v.(*schema.Set).List() {
		l := raw.(map[string]interface{})
		labels[l["label"].(string)] = l["value"].(string)
	}
	return labels
}

// mapToLabelsSet converts a map[string]string to the labels set format.
func mapToLabelsSet(labels map[string]string) []interface{} {
	result := make([]interface{}, 0, len(labels))
	for k, v := range labels {
		result = append(result, map[string]interface{}{
			"label": k,
			"value": v,
		})
	}
	return result
}

// getClient retrieves the ProviderConfig from the meta interface.
func getClient(meta interface{}) *ProviderConfig {
	return meta.(*ProviderConfig)
}

