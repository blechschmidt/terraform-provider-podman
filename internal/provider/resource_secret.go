package provider

import (
	"context"
	"encoding/base64"
	"fmt"
	"strings"

	"github.com/docker/docker/api/types/swarm"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func resourcePodmanSecret() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourcePodmanSecretCreate,
		ReadContext:   resourcePodmanSecretRead,
		DeleteContext: resourcePodmanSecretDelete,

		Schema: map[string]*schema.Schema{
			"name": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"data": {
				Type:      schema.TypeString,
				Required:  true,
				Sensitive: true,
				ForceNew:  true,
			},
			"labels": {
				Type:     schema.TypeSet,
				Optional: true,
				ForceNew: true,
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
			},
		},
	}
}

func resourcePodmanSecretCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := getClient(meta).Client

	name := d.Get("name").(string)
	data := d.Get("data").(string)

	// Base64 encode the secret data
	encodedData := base64.StdEncoding.EncodeToString([]byte(data))

	secretSpec := swarm.SecretSpec{
		Annotations: swarm.Annotations{
			Name:   name,
			Labels: labelsToMap(d.Get("labels")),
		},
		Data: []byte(encodedData),
	}

	resp, err := client.SecretCreate(ctx, secretSpec)
	if err != nil {
		return diag.FromErr(fmt.Errorf("error creating secret %s: %w", name, err))
	}

	d.SetId(resp.ID)

	return resourcePodmanSecretRead(ctx, d, meta)
}

func resourcePodmanSecretRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := getClient(meta).Client
	var diags diag.Diagnostics

	secret, _, err := client.SecretInspectWithRaw(ctx, d.Id())
	if err != nil {
		// If secret not found, mark as removed
		if strings.Contains(err.Error(), "not found") {
			d.SetId("")
			return diags
		}
		return diag.FromErr(fmt.Errorf("error inspecting secret %s: %w", d.Id(), err))
	}

	d.Set("name", secret.Spec.Name)
	d.Set("labels", mapToLabelsSet(secret.Spec.Labels))

	// Note: secret data is not readable from the API, so we do not set "data" here.
	// Terraform will retain the value from state.

	return diags
}

func resourcePodmanSecretDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := getClient(meta).Client
	var diags diag.Diagnostics

	if err := client.SecretRemove(ctx, d.Id()); err != nil {
		if strings.Contains(err.Error(), "not found") || strings.Contains(err.Error(), "no such") {
			d.SetId("")
			return diags
		}
		return diag.FromErr(fmt.Errorf("error removing secret %s: %w", d.Id(), err))
	}

	d.SetId("")
	return diags
}
