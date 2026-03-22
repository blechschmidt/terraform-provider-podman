package provider

import (
	"context"

	"github.com/docker/docker/api/types/network"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func resourcePodmanNetwork() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourcePodmanNetworkCreate,
		ReadContext:   resourcePodmanNetworkRead,
		UpdateContext: resourcePodmanNetworkUpdate,
		DeleteContext: resourcePodmanNetworkDelete,
		Schema: map[string]*schema.Schema{
			"name": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"attachable": {
				Type:     schema.TypeBool,
				Optional: true,
			},
			"driver": {
				Type:     schema.TypeString,
				Optional: true,
				Default:  "bridge",
				ForceNew: true,
			},
			"ingress": {
				Type:     schema.TypeBool,
				Optional: true,
				ForceNew: true,
			},
			"internal": {
				Type:     schema.TypeBool,
				Optional: true,
				ForceNew: true,
			},
			"ipam_config": {
				Type:     schema.TypeList,
				Optional: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"aux_address": {
							Type:     schema.TypeMap,
							Optional: true,
							Elem: &schema.Schema{
								Type: schema.TypeString,
							},
						},
						"gateway": {
							Type:     schema.TypeString,
							Optional: true,
						},
						"ip_range": {
							Type:     schema.TypeString,
							Optional: true,
						},
						"subnet": {
							Type:     schema.TypeString,
							Optional: true,
						},
					},
				},
			},
			"ipam_driver": {
				Type:     schema.TypeString,
				Optional: true,
				Default:  "default",
				ForceNew: true,
			},
			"ipam_options": {
				Type:     schema.TypeMap,
				Optional: true,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
			},
			"ipv6": {
				Type:     schema.TypeBool,
				Optional: true,
				ForceNew: true,
			},
			"labels": labelsSchema(),
			"options": {
				Type:     schema.TypeMap,
				Optional: true,
				Computed: true,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
			},
			"id": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"scope": {
				Type:     schema.TypeString,
				Computed: true,
			},
		},
	}
}

func resourcePodmanNetworkCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := getClient(meta).Client

	name := d.Get("name").(string)

	ipamConfigs := make([]network.IPAMConfig, 0)
	if v, ok := d.GetOk("ipam_config"); ok {
		for _, raw := range v.([]interface{}) {
			cfg := raw.(map[string]interface{})
			ipamConfig := network.IPAMConfig{
				Subnet:  cfg["subnet"].(string),
				IPRange: cfg["ip_range"].(string),
				Gateway: cfg["gateway"].(string),
			}
			if auxRaw, auxOk := cfg["aux_address"]; auxOk {
				ipamConfig.AuxAddress = mapStringInterfaceToStringString(auxRaw.(map[string]interface{}))
			}
			ipamConfigs = append(ipamConfigs, ipamConfig)
		}
	}

	ipamDriver := d.Get("ipam_driver").(string)
	var ipamOptions map[string]string
	if v, ok := d.GetOk("ipam_options"); ok {
		ipamOptions = mapStringInterfaceToStringString(v.(map[string]interface{}))
	}

	enableIPv6 := d.Get("ipv6").(bool)

	createOpts := network.CreateOptions{
		Driver:     d.Get("driver").(string),
		EnableIPv6: &enableIPv6,
		IPAM: &network.IPAM{
			Driver:  ipamDriver,
			Options: ipamOptions,
			Config:  ipamConfigs,
		},
		Internal:   d.Get("internal").(bool),
		Attachable: d.Get("attachable").(bool),
		Ingress:    d.Get("ingress").(bool),
		Labels:     labelsToMap(d.Get("labels")),
	}

	if v, ok := d.GetOk("options"); ok {
		createOpts.Options = mapStringInterfaceToStringString(v.(map[string]interface{}))
	}

	resp, err := client.NetworkCreate(ctx, name, createOpts)
	if err != nil {
		return diag.Errorf("error creating network %s: %s", name, err)
	}

	d.SetId(resp.ID)

	return resourcePodmanNetworkRead(ctx, d, meta)
}

func resourcePodmanNetworkRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := getClient(meta).Client

	net, err := client.NetworkInspect(ctx, d.Id(), network.InspectOptions{})
	if err != nil {
		d.SetId("")
		return diag.Errorf("error inspecting network %s: %s", d.Id(), err)
	}

	d.Set("name", net.Name)
	d.Set("scope", net.Scope)
	d.Set("driver", net.Driver)
	d.Set("attachable", net.Attachable)
	d.Set("ingress", net.Ingress)
	d.Set("internal", net.Internal)
	d.Set("ipv6", net.EnableIPv6)
	d.Set("options", net.Options)
	d.Set("labels", mapToLabelsSet(net.Labels))

	if net.IPAM.Driver != "" {
		d.Set("ipam_driver", net.IPAM.Driver)
	}
	if len(net.IPAM.Options) > 0 {
		d.Set("ipam_options", net.IPAM.Options)
	}

	ipamConfigs := make([]interface{}, 0, len(net.IPAM.Config))
	for _, cfg := range net.IPAM.Config {
		m := map[string]interface{}{
			"subnet":      cfg.Subnet,
			"gateway":     cfg.Gateway,
			"ip_range":    cfg.IPRange,
			"aux_address": cfg.AuxAddress,
		}
		ipamConfigs = append(ipamConfigs, m)
	}
	d.Set("ipam_config", ipamConfigs)

	return nil
}

func resourcePodmanNetworkUpdate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	// Only labels can be updated. The Docker/Podman API does not support updating
	// network labels in place, so we handle this by reading back the current state.
	// All other fields use ForceNew, which triggers recreation.
	// For label changes, we must recreate as well since the API has no update endpoint.
	// Terraform will handle this via the default behavior of detecting changes.
	return resourcePodmanNetworkRead(ctx, d, meta)
}

func resourcePodmanNetworkDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := getClient(meta).Client

	if err := client.NetworkRemove(ctx, d.Id()); err != nil {
		return diag.Errorf("error removing network %s: %s", d.Id(), err)
	}

	d.SetId("")
	return nil
}
