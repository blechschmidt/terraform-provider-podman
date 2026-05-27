package provider

import (
	"context"
	"crypto/sha256"
	"fmt"
	"sort"
	"strings"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/filters"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func dataSourcePodmanContainers() *schema.Resource {
	return &schema.Resource{
		Description: "Lists podman containers visible to the configured host.",
		ReadContext: dataSourcePodmanContainersRead,
		Schema: map[string]*schema.Schema{
			"all": {
				Type:        schema.TypeBool,
				Optional:    true,
				Default:     false,
				Description: "If true, include stopped containers; otherwise only running ones.",
			},
			"limit": {
				Type:        schema.TypeInt,
				Optional:    true,
				Default:     -1,
				Description: "Maximum number of containers to return (-1 means no limit).",
			},
			"name": {
				Type:        schema.TypeList,
				Optional:    true,
				Elem:        &schema.Schema{Type: schema.TypeString},
				Description: "Filter by container name(s) (regex supported).",
			},
			"containers": {
				Type:     schema.TypeList,
				Computed: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"id":       {Type: schema.TypeString, Computed: true},
						"image":    {Type: schema.TypeString, Computed: true},
						"image_id": {Type: schema.TypeString, Computed: true},
						"command":  {Type: schema.TypeString, Computed: true},
						"created":  {Type: schema.TypeInt, Computed: true},
						"state":    {Type: schema.TypeString, Computed: true},
						"status":   {Type: schema.TypeString, Computed: true},
						"names": {
							Type:     schema.TypeList,
							Computed: true,
							Elem:     &schema.Schema{Type: schema.TypeString},
						},
						"labels": {
							Type:     schema.TypeMap,
							Computed: true,
							Elem:     &schema.Schema{Type: schema.TypeString},
						},
					},
				},
			},
		},
	}
}

func dataSourcePodmanContainersRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	cli := getClient(meta).Client

	opts := container.ListOptions{
		All:   d.Get("all").(bool),
		Limit: d.Get("limit").(int),
	}
	if v, ok := d.GetOk("name"); ok {
		f := filters.NewArgs()
		for _, n := range v.([]interface{}) {
			f.Add("name", n.(string))
		}
		opts.Filters = f
	}

	list, err := cli.ContainerList(ctx, opts)
	if err != nil {
		return diag.FromErr(fmt.Errorf("error listing containers: %w", err))
	}

	out := make([]interface{}, 0, len(list))
	idDigest := sha256.New()
	for _, c := range list {
		names := make([]string, 0, len(c.Names))
		for _, n := range c.Names {
			names = append(names, strings.TrimPrefix(n, "/"))
		}
		sort.Strings(names)
		out = append(out, map[string]interface{}{
			"id":       c.ID,
			"image":    c.Image,
			"image_id": c.ImageID,
			"command":  c.Command,
			"created":  int(c.Created),
			"state":    c.State,
			"status":   c.Status,
			"names":    names,
			"labels":   c.Labels,
		})
		idDigest.Write([]byte(c.ID))
	}

	if err := d.Set("containers", out); err != nil {
		return diag.FromErr(err)
	}
	d.SetId(fmt.Sprintf("%x", idDigest.Sum(nil)))
	return nil
}
