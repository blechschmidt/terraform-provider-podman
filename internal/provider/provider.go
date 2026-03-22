package provider

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/docker/docker/client"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

// New returns a new Podman provider.
func New() *schema.Provider {
	return &schema.Provider{
		Schema: map[string]*schema.Schema{
			"host": {
				Type:        schema.TypeString,
				Optional:    true,
				DefaultFunc: schema.EnvDefaultFunc("DOCKER_HOST", "unix:///run/podman/podman.sock"),
				Description: "The Podman daemon address. Defaults to `unix:///run/podman/podman.sock`. Can also be set via DOCKER_HOST env var for compatibility.",
			},
			"cert_path": {
				Type:        schema.TypeString,
				Optional:    true,
				DefaultFunc: schema.EnvDefaultFunc("DOCKER_CERT_PATH", ""),
				Description: "Path to directory with TLS certificates.",
			},
			"ca_material": {
				Type:        schema.TypeString,
				Optional:    true,
				DefaultFunc: schema.EnvDefaultFunc("DOCKER_CA_MATERIAL", ""),
				Description: "PEM-encoded CA certificate for TLS.",
			},
			"cert_material": {
				Type:        schema.TypeString,
				Optional:    true,
				DefaultFunc: schema.EnvDefaultFunc("DOCKER_CERT_MATERIAL", ""),
				Description: "PEM-encoded client certificate for TLS.",
			},
			"key_material": {
				Type:        schema.TypeString,
				Optional:    true,
				Sensitive:   true,
				DefaultFunc: schema.EnvDefaultFunc("DOCKER_KEY_MATERIAL", ""),
				Description: "PEM-encoded client private key for TLS.",
			},
			"ssh_opts": {
				Type:        schema.TypeList,
				Optional:    true,
				Elem:        &schema.Schema{Type: schema.TypeString},
				Description: "Additional SSH options for ssh:// protocol.",
			},
			"registry_auth": {
				Type:     schema.TypeSet,
				Optional: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"address": {
							Type:        schema.TypeString,
							Required:    true,
							Description: "Address of the registry.",
						},
						"config_file": {
							Type:        schema.TypeString,
							Optional:    true,
							Default:     "~/.docker/config.json",
							Description: "Path to docker json file for registry auth. Defaults to `~/.docker/config.json`.",
						},
						"config_file_content": {
							Type:        schema.TypeString,
							Optional:    true,
							Description: "Plain content of the docker json file for registry auth.",
						},
						"username": {
							Type:        schema.TypeString,
							Optional:    true,
							DefaultFunc: schema.EnvDefaultFunc("DOCKER_REGISTRY_USER", ""),
							Description: "Username for the registry.",
						},
						"password": {
							Type:        schema.TypeString,
							Optional:    true,
							Sensitive:   true,
							DefaultFunc: schema.EnvDefaultFunc("DOCKER_REGISTRY_PASS", ""),
							Description: "Password for the registry.",
						},
						"auth_disabled": {
							Type:        schema.TypeBool,
							Optional:    true,
							Default:     false,
							Description: "Setting this to true will tell the provider that this registry does not need authentication.",
						},
					},
				},
				Description: "Registry authentication configuration.",
			},
		},
		ResourcesMap: map[string]*schema.Resource{
			"podman_container":      resourcePodmanContainer(),
			"podman_image":          resourcePodmanImage(),
			"podman_network":        resourcePodmanNetwork(),
			"podman_volume":         resourcePodmanVolume(),
			"podman_tag":            resourcePodmanTag(),
			"podman_plugin":         resourcePodmanPlugin(),
			"podman_secret":         resourcePodmanSecret(),
			"podman_registry_image": resourcePodmanRegistryImage(),
		},
		DataSourcesMap: map[string]*schema.Resource{
			"podman_image":                    dataSourcePodmanImage(),
			"podman_network":                  dataSourcePodmanNetwork(),
			"podman_plugin":                   dataSourcePodmanPlugin(),
			"podman_registry_image":           dataSourcePodmanRegistryImage(),
			"podman_logs":                     dataSourcePodmanLogs(),
			"podman_registry_image_manifests": dataSourcePodmanRegistryImageManifests(),
		},
		ConfigureContextFunc: providerConfigure,
	}
}

// ProviderConfig holds the configured Docker client and registry auth info.
type ProviderConfig struct {
	Client       *client.Client
	RegistryAuth map[string]RegistryAuth
}

// RegistryAuth holds authentication data for a single registry.
type RegistryAuth struct {
	Username        string
	Password        string
	ConfigFile      string
	ConfigContent   string
	AuthDisabled    bool
}

func providerConfigure(_ context.Context, d *schema.ResourceData) (interface{}, diag.Diagnostics) {
	var diags diag.Diagnostics

	host := d.Get("host").(string)

	opts := []client.Opt{
		client.WithHost(host),
		client.WithAPIVersionNegotiation(),
	}

	certPath := d.Get("cert_path").(string)
	caMaterial := d.Get("ca_material").(string)
	certMaterial := d.Get("cert_material").(string)
	keyMaterial := d.Get("key_material").(string)

	if certPath != "" {
		opts = append(opts, client.WithTLSClientConfig(
			certPath+"/ca.pem",
			certPath+"/cert.pem",
			certPath+"/key.pem",
		))
	} else if caMaterial != "" && certMaterial != "" && keyMaterial != "" {
		// Write temp files for TLS material
		tmpDir, err := os.MkdirTemp("", "podman-tls")
		if err != nil {
			return nil, diag.FromErr(fmt.Errorf("failed to create temp dir for TLS: %w", err))
		}
		if err := os.WriteFile(tmpDir+"/ca.pem", []byte(caMaterial), 0600); err != nil {
			return nil, diag.FromErr(err)
		}
		if err := os.WriteFile(tmpDir+"/cert.pem", []byte(certMaterial), 0600); err != nil {
			return nil, diag.FromErr(err)
		}
		if err := os.WriteFile(tmpDir+"/key.pem", []byte(keyMaterial), 0600); err != nil {
			return nil, diag.FromErr(err)
		}
		opts = append(opts, client.WithTLSClientConfig(
			tmpDir+"/ca.pem",
			tmpDir+"/cert.pem",
			tmpDir+"/key.pem",
		))
	}

	// Handle SSH options
	if v, ok := d.GetOk("ssh_opts"); ok {
		sshOpts := make([]string, 0)
		for _, opt := range v.([]interface{}) {
			sshOpts = append(sshOpts, opt.(string))
		}
		if len(sshOpts) > 0 && strings.HasPrefix(host, "ssh://") {
			// SSH options are set via environment variable
			os.Setenv("DOCKER_SSH_OPTS", strings.Join(sshOpts, " "))
		}
	}

	cli, err := client.NewClientWithOpts(opts...)
	if err != nil {
		return nil, diag.FromErr(fmt.Errorf("failed to create Podman client: %w", err))
	}

	// Parse registry auth
	registryAuth := make(map[string]RegistryAuth)
	if v, ok := d.GetOk("registry_auth"); ok {
		for _, raw := range v.(*schema.Set).List() {
			authMap := raw.(map[string]interface{})
			address := authMap["address"].(string)
			registryAuth[address] = RegistryAuth{
				Username:      authMap["username"].(string),
				Password:      authMap["password"].(string),
				ConfigFile:    authMap["config_file"].(string),
				ConfigContent: authMap["config_file_content"].(string),
				AuthDisabled:  authMap["auth_disabled"].(bool),
			}
		}
	}

	return &ProviderConfig{
		Client:       cli,
		RegistryAuth: registryAuth,
	}, diags
}
