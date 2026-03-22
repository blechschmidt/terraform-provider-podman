package provider

import (
	"archive/tar"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/image"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func resourcePodmanImage() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourcePodmanImageCreate,
		ReadContext:   resourcePodmanImageRead,
		UpdateContext: resourcePodmanImageUpdate,
		DeleteContext: resourcePodmanImageDelete,

		Schema: map[string]*schema.Schema{
			"name": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "The name of the image, including the tag or digest.",
			},
			"build": {
				Type:     schema.TypeList,
				Optional: true,
				MaxItems: 1,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"context": {
							Type:        schema.TypeString,
							Required:    true,
							Description: "The path to the build context.",
						},
						"dockerfile": {
							Type:        schema.TypeString,
							Optional:    true,
							Default:     "Dockerfile",
							Description: "Name of the Dockerfile. Defaults to 'Dockerfile'.",
						},
						"build_args": {
							Type:     schema.TypeMap,
							Optional: true,
							Elem: &schema.Schema{
								Type: schema.TypeString,
							},
							Description: "Build-time variables.",
						},
						"cache_from": {
							Type:     schema.TypeList,
							Optional: true,
							Elem: &schema.Schema{
								Type: schema.TypeString,
							},
							Description: "Images to consider as cache sources.",
						},
						"force_remove": {
							Type:        schema.TypeBool,
							Optional:    true,
							Description: "Always remove intermediate containers.",
						},
						"labels": {
							Type:     schema.TypeMap,
							Optional: true,
							Elem: &schema.Schema{
								Type: schema.TypeString,
							},
							Description: "Set metadata for the image.",
						},
						"no_cache": {
							Type:        schema.TypeBool,
							Optional:    true,
							Description: "Do not use cache when building the image.",
						},
						"platform": {
							Type:        schema.TypeString,
							Optional:    true,
							Description: "Set platform if server is multi-platform capable.",
						},
						"remove": {
							Type:        schema.TypeBool,
							Optional:    true,
							Default:     true,
							Description: "Remove intermediate containers after a successful build. Defaults to true.",
						},
						"tag": {
							Type:     schema.TypeList,
							Optional: true,
							Elem: &schema.Schema{
								Type: schema.TypeString,
							},
							Description: "Name and optionally a tag in the 'name:tag' format.",
						},
						"target": {
							Type:        schema.TypeString,
							Optional:    true,
							Description: "Set the target build stage to build.",
						},
						"network_mode": {
							Type:        schema.TypeString,
							Optional:    true,
							Description: "Set the networking mode for the RUN instructions during build.",
						},
						"extra_hosts": {
							Type:     schema.TypeList,
							Optional: true,
							Elem: &schema.Schema{
								Type: schema.TypeString,
							},
							Description: "A list of hostnames/IP mappings to add to /etc/hosts during the build.",
						},
						"shm_size": {
							Type:        schema.TypeInt,
							Optional:    true,
							Description: "Size of /dev/shm in bytes.",
						},
						"cpu_period": {
							Type:        schema.TypeInt,
							Optional:    true,
							Description: "The length of a CPU period in microseconds.",
						},
						"cpu_quota": {
							Type:        schema.TypeInt,
							Optional:    true,
							Description: "Microseconds of CPU time that the container can get in a CPU period.",
						},
						"cpu_set_cpus": {
							Type:        schema.TypeString,
							Optional:    true,
							Description: "CPUs in which to allow execution (e.g., 0-3, 0,1).",
						},
						"cpu_shares": {
							Type:        schema.TypeInt,
							Optional:    true,
							Description: "CPU shares (relative weight).",
						},
						"memory": {
							Type:        schema.TypeInt,
							Optional:    true,
							Description: "Memory limit for the build in bytes.",
						},
						"memory_swap": {
							Type:        schema.TypeInt,
							Optional:    true,
							Description: "Total memory (memory + swap). Set -1 for unlimited swap.",
						},
					},
				},
				Description: "Configuration for building the image.",
			},
			"force_remove": {
				Type:        schema.TypeBool,
				Optional:    true,
				Default:     false,
				Description: "Force remove the image on destroy.",
			},
			"keep_locally": {
				Type:        schema.TypeBool,
				Optional:    true,
				Default:     false,
				Description: "If true, the image will not be deleted on destroy.",
			},
			"platform": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "The platform to pull the image for (e.g., linux/amd64).",
			},
			"pull_triggers": {
				Type:     schema.TypeList,
				Optional: true,
				ForceNew: true,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
				Description: "List of values which cause an image pull when changed.",
			},
			"triggers": {
				Type:     schema.TypeMap,
				Optional: true,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
				Description: "A map of arbitrary strings that, when changed, will force the image to be re-created.",
			},
			"image_id": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The ID of the image.",
			},
			"repo_digest": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The image digest (repo@sha256:...).",
			},
		},
	}
}

func resourcePodmanImageCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	config := getClient(meta)
	imageName := d.Get("name").(string)

	if v, ok := d.GetOk("build"); ok {
		buildList := v.([]interface{})
		buildConfig := buildList[0].(map[string]interface{})
		return imageBuild(ctx, d, config, imageName, buildConfig)
	}

	return imagePull(ctx, d, config, imageName)
}

func imageBuild(ctx context.Context, d *schema.ResourceData, config *ProviderConfig, imageName string, buildConfig map[string]interface{}) diag.Diagnostics {
	cli := config.Client
	buildCtxPath := buildConfig["context"].(string)

	buildOpts := types.ImageBuildOptions{
		Tags:       []string{imageName},
		Dockerfile: buildConfig["dockerfile"].(string),
		Remove:     buildConfig["remove"].(bool),
	}

	if v, ok := buildConfig["build_args"]; ok {
		args := make(map[string]*string)
		for k, val := range v.(map[string]interface{}) {
			s := val.(string)
			args[k] = &s
		}
		buildOpts.BuildArgs = args
	}

	if v, ok := buildConfig["cache_from"]; ok {
		buildOpts.CacheFrom = stringListToSlice(v)
	}

	if v, ok := buildConfig["force_remove"]; ok {
		buildOpts.ForceRemove = v.(bool)
	}

	if v, ok := buildConfig["labels"]; ok {
		buildOpts.Labels = mapStringInterfaceToStringString(v.(map[string]interface{}))
	}

	if v, ok := buildConfig["no_cache"]; ok {
		buildOpts.NoCache = v.(bool)
	}

	if v, ok := buildConfig["platform"]; ok && v.(string) != "" {
		buildOpts.Platform = v.(string)
	}

	if v, ok := buildConfig["tag"]; ok {
		tags := stringListToSlice(v)
		if len(tags) > 0 {
			buildOpts.Tags = append(buildOpts.Tags, tags...)
		}
	}

	if v, ok := buildConfig["target"]; ok && v.(string) != "" {
		buildOpts.Target = v.(string)
	}

	if v, ok := buildConfig["network_mode"]; ok && v.(string) != "" {
		buildOpts.NetworkMode = v.(string)
	}

	if v, ok := buildConfig["extra_hosts"]; ok {
		buildOpts.ExtraHosts = stringListToSlice(v)
	}

	if v, ok := buildConfig["shm_size"]; ok && v.(int) > 0 {
		buildOpts.ShmSize = int64(v.(int))
	}

	if v, ok := buildConfig["cpu_period"]; ok && v.(int) > 0 {
		buildOpts.CPUPeriod = int64(v.(int))
	}

	if v, ok := buildConfig["cpu_quota"]; ok && v.(int) > 0 {
		buildOpts.CPUQuota = int64(v.(int))
	}

	if v, ok := buildConfig["cpu_set_cpus"]; ok && v.(string) != "" {
		buildOpts.CPUSetCPUs = v.(string)
	}

	if v, ok := buildConfig["cpu_shares"]; ok && v.(int) > 0 {
		buildOpts.CPUShares = int64(v.(int))
	}

	if v, ok := buildConfig["memory"]; ok && v.(int) > 0 {
		buildOpts.Memory = int64(v.(int))
	}

	if v, ok := buildConfig["memory_swap"]; ok && v.(int) != 0 {
		buildOpts.MemorySwap = int64(v.(int))
	}

	buildContext, err := createBuildContext(buildCtxPath)
	if err != nil {
		return diag.FromErr(fmt.Errorf("unable to create build context: %w", err))
	}
	defer buildContext.Close()

	resp, err := cli.ImageBuild(ctx, buildContext, buildOpts)
	if err != nil {
		return diag.FromErr(fmt.Errorf("unable to build image: %w", err))
	}
	defer resp.Body.Close()

	type buildMessage struct {
		Stream string `json:"stream"`
		Error  string `json:"error"`
	}
	decoder := json.NewDecoder(resp.Body)
	for {
		var msg buildMessage
		if err := decoder.Decode(&msg); err != nil {
			if err == io.EOF {
				break
			}
			return diag.FromErr(fmt.Errorf("error reading build output: %w", err))
		}
		if msg.Error != "" {
			return diag.FromErr(fmt.Errorf("build error: %s", msg.Error))
		}
	}

	inspectResult, _, err := cli.ImageInspectWithRaw(ctx, imageName)
	if err != nil {
		return diag.FromErr(fmt.Errorf("unable to inspect built image: %w", err))
	}

	d.SetId(inspectResult.ID)
	d.Set("image_id", inspectResult.ID)
	if len(inspectResult.RepoDigests) > 0 {
		d.Set("repo_digest", inspectResult.RepoDigests[0])
	}

	return nil
}

func imagePull(ctx context.Context, d *schema.ResourceData, config *ProviderConfig, imageName string) diag.Diagnostics {
	cli := config.Client

	pullOpts := image.PullOptions{}

	authStr, err := getRegistryAuth(config, imageName)
	if err != nil {
		return diag.FromErr(fmt.Errorf("unable to get registry auth: %w", err))
	}
	if authStr != "" {
		pullOpts.RegistryAuth = authStr
	}

	if v, ok := d.GetOk("platform"); ok {
		pullOpts.Platform = v.(string)
	}

	reader, err := cli.ImagePull(ctx, imageName, pullOpts)
	if err != nil {
		return diag.FromErr(fmt.Errorf("unable to pull image %s: %w", imageName, err))
	}
	defer reader.Close()

	_, err = io.Copy(io.Discard, reader)
	if err != nil {
		return diag.FromErr(fmt.Errorf("error reading pull output: %w", err))
	}

	inspectResult, _, err := cli.ImageInspectWithRaw(ctx, imageName)
	if err != nil {
		return diag.FromErr(fmt.Errorf("unable to inspect pulled image: %w", err))
	}

	d.SetId(inspectResult.ID)
	d.Set("image_id", inspectResult.ID)
	if len(inspectResult.RepoDigests) > 0 {
		d.Set("repo_digest", inspectResult.RepoDigests[0])
	}

	return nil
}

func resourcePodmanImageRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	config := getClient(meta)
	cli := config.Client
	imageName := d.Get("name").(string)

	inspectResult, _, err := cli.ImageInspectWithRaw(ctx, imageName)
	if err != nil {
		d.SetId("")
		return nil
	}

	d.Set("image_id", inspectResult.ID)
	if len(inspectResult.RepoDigests) > 0 {
		d.Set("repo_digest", inspectResult.RepoDigests[0])
	}

	return nil
}

func resourcePodmanImageUpdate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	config := getClient(meta)
	imageName := d.Get("name").(string)

	if d.HasChange("triggers") || d.HasChange("build") {
		if v, ok := d.GetOk("build"); ok {
			buildList := v.([]interface{})
			buildConfig := buildList[0].(map[string]interface{})
			diags := imageBuild(ctx, d, config, imageName, buildConfig)
			if diags != nil {
				return diags
			}
		} else {
			diags := imagePull(ctx, d, config, imageName)
			if diags != nil {
				return diags
			}
		}
	}

	return resourcePodmanImageRead(ctx, d, meta)
}

func resourcePodmanImageDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	if d.Get("keep_locally").(bool) {
		return nil
	}

	config := getClient(meta)
	cli := config.Client
	imageName := d.Get("name").(string)

	removeOpts := image.RemoveOptions{
		Force: d.Get("force_remove").(bool),
	}

	_, err := cli.ImageRemove(ctx, imageName, removeOpts)
	if err != nil {
		return diag.FromErr(fmt.Errorf("unable to remove image %s: %w", imageName, err))
	}

	return nil
}

func createBuildContext(contextDir string) (io.ReadCloser, error) {
	pr, pw := io.Pipe()
	go func() {
		tw := tar.NewWriter(pw)
		err := filepath.Walk(contextDir, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}

			relPath, err := filepath.Rel(contextDir, path)
			if err != nil {
				return err
			}

			header, err := tar.FileInfoHeader(info, "")
			if err != nil {
				return err
			}
			header.Name = filepath.ToSlash(relPath)

			if err := tw.WriteHeader(header); err != nil {
				return err
			}

			if info.IsDir() {
				return nil
			}

			file, err := os.Open(path)
			if err != nil {
				return err
			}
			defer file.Close()

			_, err = io.Copy(tw, file)
			return err
		})

		tw.Close()
		pw.CloseWithError(err)
	}()

	return pr, nil
}
