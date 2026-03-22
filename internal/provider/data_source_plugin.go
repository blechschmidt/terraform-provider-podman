package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func dataSourcePodmanPlugin() *schema.Resource {
	return &schema.Resource{
		Description: "Reads the plugin metadata from the Podman host.",
		ReadContext: dataSourcePodmanPluginRead,
		Schema: map[string]*schema.Schema{
			"alias": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "The alias of the plugin.",
			},
			"id": {
				Type:        schema.TypeString,
				Optional:    true,
				Computed:    true,
				Description: "The ID of the plugin.",
			},
			"enabled": {
				Type:        schema.TypeBool,
				Computed:    true,
				Description: "Whether the plugin is enabled.",
			},
			"env": {
				Type:        schema.TypeSet,
				Computed:    true,
				Elem:        &schema.Schema{Type: schema.TypeString},
				Description: "The environment variables for the plugin.",
			},
			"grant_all_permissions": {
				Type:        schema.TypeBool,
				Computed:    true,
				Description: "Whether the plugin has all permissions granted.",
			},
			"name": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The name of the plugin.",
			},
			"plugin_reference": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The plugin reference.",
			},
		},
	}
}

func dataSourcePodmanPluginRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	config := getClient(meta)
	cli := config.Client

	alias, aliasOk := d.GetOk("alias")
	id, idOk := d.GetOk("id")

	if !aliasOk && !idOk {
		return diag.Errorf("one of `alias` or `id` must be specified")
	}

	lookupName := ""
	if aliasOk {
		lookupName = alias.(string)
	} else {
		lookupName = id.(string)
	}

	plugin, _, err := cli.PluginInspectWithRaw(ctx, lookupName)
	if err != nil {
		return diag.Errorf("error inspecting plugin %s: %s", lookupName, err)
	}

	d.SetId(plugin.ID)

	if err := d.Set("enabled", plugin.Enabled); err != nil {
		return diag.FromErr(err)
	}

	envSet := make([]interface{}, len(plugin.Settings.Env))
	for i, e := range plugin.Settings.Env {
		envSet[i] = e
	}
	if err := d.Set("env", envSet); err != nil {
		return diag.FromErr(err)
	}

	// grant_all_permissions is derived from the plugin's privilege set.
	// When reading an existing plugin, we report it as true since the plugin
	// was already installed with whatever permissions it required.
	if err := d.Set("grant_all_permissions", len(plugin.Config.Linux.Capabilities) > 0); err != nil {
		return diag.FromErr(err)
	}

	if err := d.Set("name", plugin.Name); err != nil {
		return diag.FromErr(err)
	}

	pluginRef := plugin.PluginReference
	if pluginRef == "" {
		pluginRef = fmt.Sprintf("%s:latest", plugin.Name)
	}
	if err := d.Set("plugin_reference", pluginRef); err != nil {
		return diag.FromErr(err)
	}

	return nil
}
