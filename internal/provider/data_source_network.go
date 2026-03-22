package provider

import (
	"context"

	"github.com/docker/docker/api/types/network"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func dataSourcePodmanNetwork() *schema.Resource {
	return &schema.Resource{
		Description: "Reads the network metadata from the Podman host.",
		ReadContext: dataSourcePodmanNetworkRead,
		Schema: map[string]*schema.Schema{
			"name": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "The name of the network.",
			},
			"id": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The ID of the network.",
			},
			"driver": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The driver of the network (e.g. `bridge`, `overlay`).",
			},
			"internal": {
				Type:        schema.TypeBool,
				Computed:    true,
				Description: "Whether the network is internal.",
			},
			"scope": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The scope of the network (e.g. `local`, `swarm`).",
			},
			"options": {
				Type:        schema.TypeMap,
				Computed:    true,
				Elem:        &schema.Schema{Type: schema.TypeString},
				Description: "Network-specific options.",
			},
			"ipam_config": {
				Type:        schema.TypeList,
				Computed:    true,
				Description: "IPAM configuration blocks for the network.",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"aux_address": {
							Type:        schema.TypeMap,
							Computed:    true,
							Elem:        &schema.Schema{Type: schema.TypeString},
							Description: "Auxiliary addresses for the subnet.",
						},
						"gateway": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "The gateway for the subnet.",
						},
						"ip_range": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "The IP range for the subnet.",
						},
						"subnet": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "The subnet in CIDR notation.",
						},
					},
				},
			},
		},
	}
}

func dataSourcePodmanNetworkRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	config := getClient(meta)
	cli := config.Client

	name := d.Get("name").(string)

	net, err := cli.NetworkInspect(ctx, name, network.InspectOptions{})
	if err != nil {
		return diag.Errorf("error inspecting network %s: %s", name, err)
	}

	d.SetId(net.ID)

	if err := d.Set("driver", net.Driver); err != nil {
		return diag.FromErr(err)
	}
	if err := d.Set("internal", net.Internal); err != nil {
		return diag.FromErr(err)
	}
	if err := d.Set("scope", net.Scope); err != nil {
		return diag.FromErr(err)
	}
	if err := d.Set("options", net.Options); err != nil {
		return diag.FromErr(err)
	}

	ipamConfigs := make([]interface{}, 0, len(net.IPAM.Config))
	for _, ipamCfg := range net.IPAM.Config {
		m := map[string]interface{}{
			"aux_address": ipamCfg.AuxAddress,
			"gateway":     ipamCfg.Gateway,
			"ip_range":    ipamCfg.IPRange,
			"subnet":      ipamCfg.Subnet,
		}
		ipamConfigs = append(ipamConfigs, m)
	}
	if err := d.Set("ipam_config", ipamConfigs); err != nil {
		return diag.FromErr(err)
	}

	return nil
}
