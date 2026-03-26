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

// suppressIfIDOrNameEqual is a DiffSuppressFunc that suppresses diffs for image references
// where one is an ID (sha256:xxx) and the other is a name that resolves to it.
func suppressIfIDOrNameEqual(_, old, new string, _ *schema.ResourceData) bool {
	if old == new {
		return true
	}
	if old == "" || new == "" {
		return false
	}
	// Suppress if both refer to same image by stripping sha256: prefix
	oldClean := strings.TrimPrefix(old, "sha256:")
	newClean := strings.TrimPrefix(new, "sha256:")
	if len(oldClean) > 0 && len(newClean) > 0 &&
		(strings.HasPrefix(oldClean, newClean) || strings.HasPrefix(newClean, oldClean)) {
		return true
	}
	return false
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

