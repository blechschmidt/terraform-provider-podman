package provider

import (
	"context"

	"github.com/docker/docker/api/types/volume"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func resourcePodmanVolume() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourcePodmanVolumeCreate,
		ReadContext:   resourcePodmanVolumeRead,
		DeleteContext: resourcePodmanVolumeDelete,
		Schema: map[string]*schema.Schema{
			"name": {
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
				ForceNew: true,
			},
			"driver": {
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
				ForceNew: true,
			},
			"driver_opts": {
				Type:     schema.TypeMap,
				Optional: true,
				ForceNew: true,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
			},
			"labels": func() *schema.Schema {
				s := labelsSchema()
				s.ForceNew = true
				return s
			}(),
			"id": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"mountpoint": {
				Type:     schema.TypeString,
				Computed: true,
			},
		},
	}
}

func resourcePodmanVolumeCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := getClient(meta).Client

	createOpts := volume.CreateOptions{
		Name:   d.Get("name").(string),
		Driver: d.Get("driver").(string),
		Labels: labelsToMap(d.Get("labels")),
	}

	if v, ok := d.GetOk("driver_opts"); ok {
		createOpts.DriverOpts = mapStringInterfaceToStringString(v.(map[string]interface{}))
	}

	vol, err := client.VolumeCreate(ctx, createOpts)
	if err != nil {
		return diag.Errorf("error creating volume: %s", err)
	}

	d.SetId(vol.Name)

	return resourcePodmanVolumeRead(ctx, d, meta)
}

func resourcePodmanVolumeRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := getClient(meta).Client

	vol, err := client.VolumeInspect(ctx, d.Id())
	if err != nil {
		d.SetId("")
		return diag.Errorf("error inspecting volume %s: %s", d.Id(), err)
	}

	d.Set("name", vol.Name)
	d.Set("driver", vol.Driver)
	d.Set("mountpoint", vol.Mountpoint)
	d.Set("labels", mapToLabelsSet(vol.Labels))

	if len(vol.Options) > 0 {
		d.Set("driver_opts", vol.Options)
	}

	return nil
}

func resourcePodmanVolumeDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := getClient(meta).Client

	if err := client.VolumeRemove(ctx, d.Id(), true); err != nil {
		return diag.Errorf("error removing volume %s: %s", d.Id(), err)
	}

	d.SetId("")
	return nil
}
