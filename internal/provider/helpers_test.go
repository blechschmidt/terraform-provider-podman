package provider

import (
	"encoding/base64"
	"encoding/json"
	"testing"

	"github.com/docker/docker/api/types/registry"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

// --- getRegistryAuth tests ---

func TestGetRegistryAuth_AuthDisabled(t *testing.T) {
	config := &ProviderConfig{
		RegistryAuth: map[string]RegistryAuth{
			"docker.io": {
				AuthDisabled: true,
				Username:     "user",
				Password:     "pass",
			},
		},
	}
	result, err := getRegistryAuth(config, "nginx")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "" {
		t.Errorf("expected empty string when auth disabled, got %q", result)
	}
}

func TestGetRegistryAuth_UsernamePassword(t *testing.T) {
	config := &ProviderConfig{
		RegistryAuth: map[string]RegistryAuth{
			"docker.io": {
				Username: "myuser",
				Password: "mypass",
			},
		},
	}
	result, err := getRegistryAuth(config, "nginx")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result == "" {
		t.Fatal("expected non-empty auth string")
	}
	decoded, err := base64.URLEncoding.DecodeString(result)
	if err != nil {
		t.Fatalf("failed to decode base64: %v", err)
	}
	var authConfig registry.AuthConfig
	if err := json.Unmarshal(decoded, &authConfig); err != nil {
		t.Fatalf("failed to unmarshal auth config: %v", err)
	}
	if authConfig.Username != "myuser" {
		t.Errorf("expected username 'myuser', got %q", authConfig.Username)
	}
	if authConfig.Password != "mypass" {
		t.Errorf("expected password 'mypass', got %q", authConfig.Password)
	}
}

func TestGetRegistryAuth_NoAuthConfigured(t *testing.T) {
	config := &ProviderConfig{
		RegistryAuth: map[string]RegistryAuth{},
	}
	result, err := getRegistryAuth(config, "nginx")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "" {
		t.Errorf("expected empty string when no auth configured, got %q", result)
	}
}

func TestGetRegistryAuth_NoMatchingRegistry(t *testing.T) {
	config := &ProviderConfig{
		RegistryAuth: map[string]RegistryAuth{
			"registry.example.com": {
				Username: "user",
				Password: "pass",
			},
		},
	}
	result, err := getRegistryAuth(config, "nginx")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "" {
		t.Errorf("expected empty string for non-matching registry, got %q", result)
	}
}

// --- getRegistryFromImageName tests ---

func TestGetRegistryFromImageName(t *testing.T) {
	tests := []struct {
		name     string
		image    string
		expected string
	}{
		{
			name:     "simple image",
			image:    "nginx",
			expected: "docker.io",
		},
		{
			name:     "library image",
			image:    "library/nginx",
			expected: "docker.io",
		},
		{
			name:     "registry with dot",
			image:    "registry.example.com/repo",
			expected: "registry.example.com",
		},
		{
			name:     "registry with port",
			image:    "localhost:5000/repo",
			expected: "localhost:5000",
		},
		{
			name:     "localhost repo",
			image:    "localhost/repo",
			expected: "localhost",
		},
		{
			name:     "user repo no dot",
			image:    "user/repo",
			expected: "docker.io",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := getRegistryFromImageName(tc.image)
			if result != tc.expected {
				t.Errorf("getRegistryFromImageName(%q) = %q, want %q", tc.image, result, tc.expected)
			}
		})
	}
}

// --- suppressIfIDOrNameEqual tests ---

func TestSuppressIfIDOrNameEqual(t *testing.T) {
	tests := []struct {
		name     string
		old      string
		new      string
		expected bool
	}{
		{
			name:     "equal strings",
			old:      "abc123",
			new:      "abc123",
			expected: true,
		},
		{
			name:     "old empty",
			old:      "",
			new:      "abc123",
			expected: false,
		},
		{
			name:     "new empty",
			old:      "abc123",
			new:      "",
			expected: false,
		},
		{
			name:     "both empty",
			old:      "",
			new:      "",
			expected: true,
		},
		{
			name:     "sha256 prefix match",
			old:      "sha256:abc123def456",
			new:      "abc123def456",
			expected: true,
		},
		{
			name:     "no match",
			old:      "abc",
			new:      "xyz",
			expected: false,
		},
		{
			name:     "different strings",
			old:      "nginx:latest",
			new:      "nginx:1.21",
			expected: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := suppressIfIDOrNameEqual("", tc.old, tc.new, nil)
			if result != tc.expected {
				t.Errorf("suppressIfIDOrNameEqual(%q, %q) = %v, want %v", tc.old, tc.new, result, tc.expected)
			}
		})
	}
}

// --- mapStringInterfaceToStringString tests ---

func TestMapStringInterfaceToStringString_Basic(t *testing.T) {
	input := map[string]interface{}{
		"key1": "value1",
		"key2": "value2",
	}
	result := mapStringInterfaceToStringString(input)
	if len(result) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(result))
	}
	if result["key1"] != "value1" {
		t.Errorf("expected key1=value1, got %q", result["key1"])
	}
	if result["key2"] != "value2" {
		t.Errorf("expected key2=value2, got %q", result["key2"])
	}
}

func TestMapStringInterfaceToStringString_Empty(t *testing.T) {
	input := map[string]interface{}{}
	result := mapStringInterfaceToStringString(input)
	if len(result) != 0 {
		t.Fatalf("expected 0 entries, got %d", len(result))
	}
}

// --- stringListToSlice tests ---

func TestStringListToSlice_Normal(t *testing.T) {
	input := []interface{}{"a", "b", "c"}
	result := stringListToSlice(input)
	if len(result) != 3 {
		t.Fatalf("expected 3 elements, got %d", len(result))
	}
	if result[0] != "a" || result[1] != "b" || result[2] != "c" {
		t.Errorf("unexpected result: %v", result)
	}
}

func TestStringListToSlice_Empty(t *testing.T) {
	input := []interface{}{}
	result := stringListToSlice(input)
	if len(result) != 0 {
		t.Fatalf("expected 0 elements, got %d", len(result))
	}
}

// --- stringSetToSlice tests ---

func TestStringSetToSlice_Normal(t *testing.T) {
	items := []interface{}{"x", "y", "z"}
	set := schema.NewSet(schema.HashString, items)
	result := stringSetToSlice(set)
	if len(result) != 3 {
		t.Fatalf("expected 3 elements, got %d", len(result))
	}
	found := make(map[string]bool)
	for _, v := range result {
		found[v] = true
	}
	for _, expected := range []string{"x", "y", "z"} {
		if !found[expected] {
			t.Errorf("expected %q in result, but not found", expected)
		}
	}
}

func TestStringSetToSlice_Empty(t *testing.T) {
	set := schema.NewSet(schema.HashString, []interface{}{})
	result := stringSetToSlice(set)
	if len(result) != 0 {
		t.Fatalf("expected 0 elements, got %d", len(result))
	}
}

// --- labelsToMap tests ---

func TestLabelsToMap_Normal(t *testing.T) {
	labelResource := labelsSchema().Elem.(*schema.Resource)
	items := []interface{}{
		map[string]interface{}{
			"label": "env",
			"value": "production",
		},
		map[string]interface{}{
			"label": "app",
			"value": "web",
		},
	}
	set := schema.NewSet(schema.HashResource(labelResource), items)
	result := labelsToMap(set)
	if len(result) != 2 {
		t.Fatalf("expected 2 labels, got %d", len(result))
	}
	if result["env"] != "production" {
		t.Errorf("expected env=production, got %q", result["env"])
	}
	if result["app"] != "web" {
		t.Errorf("expected app=web, got %q", result["app"])
	}
}

func TestLabelsToMap_Nil(t *testing.T) {
	result := labelsToMap(nil)
	if len(result) != 0 {
		t.Fatalf("expected 0 labels for nil input, got %d", len(result))
	}
}

func TestLabelsToMap_EmptySet(t *testing.T) {
	labelResource := labelsSchema().Elem.(*schema.Resource)
	set := schema.NewSet(schema.HashResource(labelResource), []interface{}{})
	result := labelsToMap(set)
	if len(result) != 0 {
		t.Fatalf("expected 0 labels for empty set, got %d", len(result))
	}
}

// --- mapToLabelsSet tests ---

func TestMapToLabelsSet_Normal(t *testing.T) {
	input := map[string]string{
		"env": "production",
		"app": "web",
	}
	result := mapToLabelsSet(input)
	if len(result) != 2 {
		t.Fatalf("expected 2 items, got %d", len(result))
	}
	found := make(map[string]string)
	for _, item := range result {
		m := item.(map[string]interface{})
		found[m["label"].(string)] = m["value"].(string)
	}
	if found["env"] != "production" {
		t.Errorf("expected env=production, got %q", found["env"])
	}
	if found["app"] != "web" {
		t.Errorf("expected app=web, got %q", found["app"])
	}
}

func TestMapToLabelsSet_Empty(t *testing.T) {
	result := mapToLabelsSet(map[string]string{})
	if len(result) != 0 {
		t.Fatalf("expected 0 items, got %d", len(result))
	}
}

// --- getClient tests ---

func TestGetClient_CastWorks(t *testing.T) {
	config := &ProviderConfig{
		RegistryAuth: map[string]RegistryAuth{},
	}
	var meta interface{} = config
	result := getClient(meta)
	if result != config {
		t.Error("getClient did not return the same ProviderConfig pointer")
	}
}

// --- formatImageRef tests (from resource_tag.go) ---

func TestFormatImageRef(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "with tag",
			input:    "myrepo:v1.0",
			expected: "myrepo:v1.0",
		},
		{
			name:     "without tag",
			input:    "myrepo",
			expected: "myrepo:latest",
		},
		{
			name:     "registry with port and tag",
			input:    "localhost:5000/myrepo:v2",
			expected: "localhost:5000/myrepo:v2",
		},
		{
			name:     "registry with port and no tag",
			input:    "localhost:5000/myrepo",
			expected: "localhost:5000/myrepo:latest",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := formatImageRef(tc.input)
			if result != tc.expected {
				t.Errorf("formatImageRef(%q) = %q, want %q", tc.input, result, tc.expected)
			}
		})
	}
}
