package provider

import (
	"context"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func resourcePodmanPlugin() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourcePodmanPluginCreate,
		ReadContext:   resourcePodmanPluginRead,
		UpdateContext: resourcePodmanPluginUpdate,
		DeleteContext: resourcePodmanPluginDelete,

		Schema: map[string]*schema.Schema{
			"name": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"alias": {
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
				ForceNew: true,
			},
			"enable_timeout": {
				Type:     schema.TypeInt,
				Optional: true,
				Default:  60,
			},
			"enabled": {
				Type:     schema.TypeBool,
				Optional: true,
				Default:  true,
			},
			"env": {
				Type:     schema.TypeSet,
				Optional: true,
				Computed: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
			},
			"force_destroy": {
				Type:     schema.TypeBool,
				Optional: true,
				Default:  false,
			},
			"force_disable": {
				Type:     schema.TypeBool,
				Optional: true,
				Default:  false,
			},
			"grant_all_permissions": {
				Type:     schema.TypeBool,
				Optional: true,
				Default:  false,
			},
			"grant_permissions": {
				Type:     schema.TypeSet,
				Optional: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"name": {
							Type:     schema.TypeString,
							Required: true,
						},
						"value": {
							Type:     schema.TypeSet,
							Required: true,
							Elem:     &schema.Schema{Type: schema.TypeString},
						},
					},
				},
			},
			"plugin_reference": {
				Type:     schema.TypeString,
				Computed: true,
			},
		},
	}
}

func resourcePodmanPluginCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := getClient(meta).Client
	name := d.Get("name").(string)
	alias := d.Get("alias").(string)
	grantAllPermissions := d.Get("grant_all_permissions").(bool)

	opts := types.PluginInstallOptions{
		RemoteRef:            name,
		AcceptAllPermissions: grantAllPermissions,
		Disabled:             true, // Install disabled, enable separately if needed
	}

	if alias != "" {
		opts.RemoteRef = name
	}

	if v, ok := d.GetOk("grant_permissions"); ok {
		grantPerms := v.(*schema.Set).List()
		opts.AcceptPermissionsFunc = func(_ context.Context, privileges types.PluginPrivileges) (bool, error) {
			grantedMap := make(map[string]map[string]bool)
			for _, raw := range grantPerms {
				perm := raw.(map[string]interface{})
				permName := perm["name"].(string)
				permValues := stringSetToSlice(perm["value"])
				valueSet := make(map[string]bool)
				for _, v := range permValues {
					valueSet[v] = true
				}
				grantedMap[permName] = valueSet
			}
			for _, priv := range privileges {
				if grantAllPermissions {
					continue
				}
				if granted, ok := grantedMap[priv.Name]; ok {
					for _, v := range priv.Value {
						if !granted[v] {
							return false, fmt.Errorf("permission %q value %q not granted", priv.Name, v)
						}
					}
				} else {
					return false, fmt.Errorf("permission %q not granted", priv.Name)
				}
			}
			return true, nil
		}
	}

	if v, ok := d.GetOk("env"); ok {
		opts.Args = stringSetToSlice(v)
	}

	installReader, err := client.PluginInstall(ctx, alias, opts)
	if err != nil {
		return diag.FromErr(fmt.Errorf("error installing plugin %s: %w", name, err))
	}
	// Drain the reader to ensure installation completes
	_, _ = io.ReadAll(installReader)
	installReader.Close()

	// Look up the installed plugin to get its ID
	pluginRef := name
	if alias != "" {
		pluginRef = alias
	}

	plugin, _, err := client.PluginInspectWithRaw(ctx, pluginRef)
	if err != nil {
		return diag.FromErr(fmt.Errorf("error inspecting plugin %s after install: %w", pluginRef, err))
	}

	d.SetId(plugin.ID)

	// Enable if requested
	if d.Get("enabled").(bool) {
		enableTimeout := d.Get("enable_timeout").(int)
		timeout := time.Duration(enableTimeout) * time.Second
		if err := client.PluginEnable(ctx, plugin.ID, types.PluginEnableOptions{Timeout: int(timeout.Seconds())}); err != nil {
			return diag.FromErr(fmt.Errorf("error enabling plugin %s: %w", plugin.ID, err))
		}
	}

	return resourcePodmanPluginRead(ctx, d, meta)
}

func resourcePodmanPluginRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := getClient(meta).Client
	var diags diag.Diagnostics

	plugin, _, err := client.PluginInspectWithRaw(ctx, d.Id())
	if err != nil {
		// If plugin not found, mark as removed
		if strings.Contains(err.Error(), "not found") {
			d.SetId("")
			return diags
		}
		return diag.FromErr(fmt.Errorf("error inspecting plugin %s: %w", d.Id(), err))
	}

	d.Set("name", plugin.PluginReference)
	d.Set("alias", plugin.Name)
	d.Set("enabled", plugin.Enabled)
	d.Set("plugin_reference", plugin.PluginReference)

	// Read env from plugin settings
	if plugin.Settings.Env != nil {
		d.Set("env", plugin.Settings.Env)
	}

	return diags
}

func resourcePodmanPluginUpdate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := getClient(meta).Client
	pluginID := d.Id()

	if d.HasChange("enabled") {
		enabled := d.Get("enabled").(bool)
		enableTimeout := d.Get("enable_timeout").(int)

		if enabled {
			if err := client.PluginEnable(ctx, pluginID, types.PluginEnableOptions{Timeout: enableTimeout}); err != nil {
				return diag.FromErr(fmt.Errorf("error enabling plugin %s: %w", pluginID, err))
			}
		} else {
			forceDisable := d.Get("force_disable").(bool)
			if err := client.PluginDisable(ctx, pluginID, types.PluginDisableOptions{Force: forceDisable}); err != nil {
				return diag.FromErr(fmt.Errorf("error disabling plugin %s: %w", pluginID, err))
			}
		}
	}

	if d.HasChange("env") {
		env := stringSetToSlice(d.Get("env"))
		if err := client.PluginSet(ctx, pluginID, env); err != nil {
			return diag.FromErr(fmt.Errorf("error updating env for plugin %s: %w", pluginID, err))
		}
	}

	return resourcePodmanPluginRead(ctx, d, meta)
}

func resourcePodmanPluginDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := getClient(meta).Client
	var diags diag.Diagnostics
	pluginID := d.Id()
	forceDestroy := d.Get("force_destroy").(bool)

	// Disable the plugin first if it is enabled
	plugin, _, err := client.PluginInspectWithRaw(ctx, pluginID)
	if err != nil {
		return diag.FromErr(fmt.Errorf("error inspecting plugin %s before delete: %w", pluginID, err))
	}

	if plugin.Enabled {
		forceDisable := d.Get("force_disable").(bool)
		if err := client.PluginDisable(ctx, pluginID, types.PluginDisableOptions{Force: forceDisable}); err != nil {
			return diag.FromErr(fmt.Errorf("error disabling plugin %s: %w", pluginID, err))
		}
	}

	if err := client.PluginRemove(ctx, pluginID, types.PluginRemoveOptions{Force: forceDestroy}); err != nil {
		return diag.FromErr(fmt.Errorf("error removing plugin %s: %w", pluginID, err))
	}

	d.SetId("")
	return diags
}
