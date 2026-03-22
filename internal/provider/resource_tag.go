package provider

import (
	"context"
	"fmt"
	"strings"

	"github.com/docker/docker/api/types/image"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func resourcePodmanTag() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourcePodmanTagCreate,
		ReadContext:   resourcePodmanTagRead,
		UpdateContext: resourcePodmanTagUpdate,
		DeleteContext: resourcePodmanTagDelete,

		Schema: map[string]*schema.Schema{
			"source_image": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "The name or ID of the source image to tag.",
			},
			"target_image": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "The target image name with optional tag (e.g., 'repo:tag' or 'registry/repo:tag').",
			},
			"source_image_id": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The ID of the source image.",
			},
		},
	}
}

func formatImageRef(imageName string) string {
	// ImageTag expects target as "repo:tag" format
	// If no tag is specified, append ":latest"
	lastSlash := strings.LastIndex(imageName, "/")
	var namePart string
	if lastSlash >= 0 {
		namePart = imageName[lastSlash+1:]
	} else {
		namePart = imageName
	}

	colonIdx := strings.LastIndex(namePart, ":")
	if colonIdx >= 0 {
		return imageName
	}

	return imageName + ":latest"
}

func resourcePodmanTagCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	config := getClient(meta)
	cli := config.Client

	sourceImage := d.Get("source_image").(string)
	targetImage := d.Get("target_image").(string)

	targetRef := formatImageRef(targetImage)

	err := cli.ImageTag(ctx, sourceImage, targetRef)
	if err != nil {
		return diag.FromErr(fmt.Errorf("unable to tag image %s as %s: %w", sourceImage, targetImage, err))
	}

	d.SetId(sourceImage + ":" + targetImage)

	return resourcePodmanTagRead(ctx, d, meta)
}

func resourcePodmanTagRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	config := getClient(meta)
	cli := config.Client

	sourceImage := d.Get("source_image").(string)

	inspectResult, _, err := cli.ImageInspectWithRaw(ctx, sourceImage)
	if err != nil {
		d.SetId("")
		return nil
	}

	d.Set("source_image_id", inspectResult.ID)

	return nil
}

func resourcePodmanTagUpdate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	config := getClient(meta)
	cli := config.Client

	if d.HasChange("source_image") || d.HasChange("target_image") {
		// Remove old tag if target changed
		if d.HasChange("target_image") {
			oldTarget, _ := d.GetChange("target_image")
			oldTargetStr := oldTarget.(string)
			_, err := cli.ImageRemove(ctx, oldTargetStr, image.RemoveOptions{
				Force:         false,
				PruneChildren: false,
			})
			if err != nil {
				return diag.FromErr(fmt.Errorf("unable to remove old tag %s: %w", oldTargetStr, err))
			}
		}

		sourceImage := d.Get("source_image").(string)
		targetImage := d.Get("target_image").(string)
		targetRef := formatImageRef(targetImage)

		err := cli.ImageTag(ctx, sourceImage, targetRef)
		if err != nil {
			return diag.FromErr(fmt.Errorf("unable to re-tag image %s as %s: %w", sourceImage, targetImage, err))
		}

		d.SetId(sourceImage + ":" + targetImage)
	}

	return resourcePodmanTagRead(ctx, d, meta)
}

func resourcePodmanTagDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	config := getClient(meta)
	cli := config.Client

	targetImage := d.Get("target_image").(string)

	_, err := cli.ImageRemove(ctx, targetImage, image.RemoveOptions{
		Force:         false,
		PruneChildren: false,
	})
	if err != nil {
		return diag.FromErr(fmt.Errorf("unable to remove tag %s: %w", targetImage, err))
	}

	return nil
}
