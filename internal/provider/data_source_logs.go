package provider

import (
	"bufio"
	"context"
	"io"
	"strings"

	"github.com/docker/docker/api/types/container"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func dataSourcePodmanLogs() *schema.Resource {
	return &schema.Resource{
		Description: "Reads the logs from a container.",
		ReadContext: dataSourcePodmanLogsRead,
		Schema: map[string]*schema.Schema{
			"name": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "The name or ID of the container.",
			},
			"details": {
				Type:        schema.TypeBool,
				Optional:    true,
				Default:     false,
				Description: "Show extra details provided to logs.",
			},
			"discard_headers": {
				Type:        schema.TypeBool,
				Optional:    true,
				Default:     false,
				Description: "Discard the 8-byte Docker log header from each line.",
			},
			"follow": {
				Type:        schema.TypeBool,
				Optional:    true,
				Default:     false,
				Description: "Follow log output.",
			},
			"logs_list_string_enabled": {
				Type:        schema.TypeBool,
				Optional:    true,
				Default:     false,
				Description: "If true, the output will be stored as a list of strings.",
			},
			"show_stderr": {
				Type:        schema.TypeBool,
				Optional:    true,
				Default:     true,
				Description: "Show stderr log output.",
			},
			"show_stdout": {
				Type:        schema.TypeBool,
				Optional:    true,
				Default:     true,
				Description: "Show stdout log output.",
			},
			"since": {
				Type:        schema.TypeString,
				Optional:    true,
				Default:     "",
				Description: "Show logs since this timestamp or relative duration.",
			},
			"tail": {
				Type:        schema.TypeString,
				Optional:    true,
				Default:     "all",
				Description: "Number of lines to show from the end of the logs. Set to `all` to show all lines.",
			},
			"timestamps": {
				Type:        schema.TypeBool,
				Optional:    true,
				Default:     false,
				Description: "Add timestamps to every log line.",
			},
			"until": {
				Type:        schema.TypeString,
				Optional:    true,
				Default:     "",
				Description: "Show logs before this timestamp or relative duration.",
			},
			"id": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The ID of the data source.",
			},
			"logs_list_string": {
				Type:        schema.TypeList,
				Computed:    true,
				Elem:        &schema.Schema{Type: schema.TypeString},
				Description: "The log output as a list of strings, one per line.",
			},
		},
	}
}

func dataSourcePodmanLogsRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	config := getClient(meta)
	cli := config.Client

	name := d.Get("name").(string)

	opts := container.LogsOptions{
		ShowStdout: d.Get("show_stdout").(bool),
		ShowStderr: d.Get("show_stderr").(bool),
		Since:      d.Get("since").(string),
		Until:      d.Get("until").(string),
		Timestamps: d.Get("timestamps").(bool),
		Follow:     d.Get("follow").(bool),
		Tail:       d.Get("tail").(string),
		Details:    d.Get("details").(bool),
	}

	reader, err := cli.ContainerLogs(ctx, name, opts)
	if err != nil {
		return diag.Errorf("error reading logs for container %s: %s", name, err)
	}
	defer reader.Close()

	discardHeaders := d.Get("discard_headers").(bool)

	var lines []string
	scanner := bufio.NewScanner(reader)
	for scanner.Scan() {
		line := scanner.Text()
		if discardHeaders && len(line) >= 8 {
			line = line[8:]
		}
		lines = append(lines, line)
	}
	if err := scanner.Err(); err != nil && err != io.EOF {
		return diag.Errorf("error scanning logs for container %s: %s", name, err)
	}

	d.SetId(name)

	logsListStringEnabled := d.Get("logs_list_string_enabled").(bool)
	if logsListStringEnabled {
		if err := d.Set("logs_list_string", lines); err != nil {
			return diag.FromErr(err)
		}
	} else {
		_ = d.Set("logs_list_string", strings.Split(strings.Join(lines, "\n"), "\n"))
	}

	return nil
}
