package provider

import (
	"archive/tar"
	"bytes"
	"context"
	"encoding/base64"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/mount"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/api/types/strslice"
	"github.com/docker/go-connections/nat"
	"github.com/docker/go-units"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func resourcePodmanContainer() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourcePodmanContainerCreate,
		ReadContext:   resourcePodmanContainerRead,
		UpdateContext: resourcePodmanContainerUpdate,
		DeleteContext: resourcePodmanContainerDelete,

		Schema: map[string]*schema.Schema{
			// Required
			"name": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"image": {
				Type:             schema.TypeString,
				Required:         true,
				ForceNew:         true,
				DiffSuppressFunc: suppressIfIDOrNameEqual,
			},

			// Optional
			"attach": {
				Type:     schema.TypeBool,
				Optional: true,
				Default:  false,
			},
			"capabilities": {
				Type:     schema.TypeList,
				Optional: true,
				ForceNew: true,
				MaxItems: 1,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"add": {
							Type:     schema.TypeSet,
							Optional: true,
							Elem:     &schema.Schema{Type: schema.TypeString},
						},
						"drop": {
							Type:     schema.TypeSet,
							Optional: true,
							Elem:     &schema.Schema{Type: schema.TypeString},
						},
					},
				},
			},
			"cgroupns_mode": {
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
				ForceNew: true,
			},
			"command": {
				Type:     schema.TypeList,
				Optional: true,
				Computed: true,
				ForceNew: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
			},
			"cpu_set": {
				Type:     schema.TypeString,
				Optional: true,
			},
			"cpu_shares": {
				Type:     schema.TypeInt,
				Optional: true,
			},
			"cpu_period": {
				Type:     schema.TypeInt,
				Optional: true,
			},
			"cpu_quota": {
				Type:     schema.TypeInt,
				Optional: true,
			},
			"destroy_grace_seconds": {
				Type:     schema.TypeInt,
				Optional: true,
			},
			"devices": {
				Type:     schema.TypeSet,
				Optional: true,
				ForceNew: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"host_path": {
							Type:     schema.TypeString,
							Required: true,
						},
						"container_path": {
							Type:     schema.TypeString,
							Optional: true,
						},
						"permissions": {
							Type:     schema.TypeString,
							Optional: true,
							Default:  "rwm",
						},
					},
				},
			},
			"dns": {
				Type:     schema.TypeSet,
				Optional: true,
				ForceNew: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
			},
			"dns_opts": {
				Type:     schema.TypeSet,
				Optional: true,
				ForceNew: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
			},
			"dns_search": {
				Type:     schema.TypeSet,
				Optional: true,
				ForceNew: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
			},
			"domainname": {
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
			},
			"entrypoint": {
				Type:     schema.TypeList,
				Optional: true,
				Computed: true,
				ForceNew: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
			},
			"env": {
				Type:     schema.TypeSet,
				Optional: true,
				Computed: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
			},
			"group_add": {
				Type:     schema.TypeSet,
				Optional: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
			},
			"healthcheck": {
				Type:     schema.TypeList,
				Optional: true,
				MaxItems: 1,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"test": {
							Type:     schema.TypeList,
							Required: true,
							Elem:     &schema.Schema{Type: schema.TypeString},
						},
						"interval": {
							Type:     schema.TypeString,
							Optional: true,
							Default:  "0s",
						},
						"retries": {
							Type:     schema.TypeInt,
							Optional: true,
							Default:  0,
						},
						"start_period": {
							Type:     schema.TypeString,
							Optional: true,
							Default:  "0s",
						},
						"timeout": {
							Type:     schema.TypeString,
							Optional: true,
							Default:  "0s",
						},
					},
				},
			},
			"host": {
				Type:     schema.TypeSet,
				Optional: true,
				ForceNew: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"host": {
							Type:     schema.TypeString,
							Required: true,
						},
						"ip": {
							Type:     schema.TypeString,
							Required: true,
						},
					},
				},
			},
			"hostname": {
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
				ForceNew: true,
			},
			"init": {
				Type:     schema.TypeBool,
				Optional: true,
			},
			"ipc_mode": {
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
				ForceNew: true,
			},
			"labels": labelsSchema(),
			"log_driver": {
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
			},
			"log_opts": {
				Type:     schema.TypeMap,
				Optional: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
			},
			"logs": {
				Type:     schema.TypeBool,
				Optional: true,
				Default:  false,
			},
			"max_retry_count": {
				Type:     schema.TypeInt,
				Optional: true,
			},
			"memory": {
				Type:     schema.TypeInt,
				Optional: true,
			},
			"memory_swap": {
				Type:     schema.TypeInt,
				Optional: true,
			},
			"mounts": {
				Type:     schema.TypeSet,
				Optional: true,
				ForceNew: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"target": {
							Type:     schema.TypeString,
							Required: true,
						},
						"type": {
							Type:     schema.TypeString,
							Required: true,
						},
						"read_only": {
							Type:     schema.TypeBool,
							Optional: true,
						},
						"source": {
							Type:     schema.TypeString,
							Optional: true,
						},
						"bind_options": {
							Type:     schema.TypeList,
							Optional: true,
							MaxItems: 1,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"propagation": {
										Type:     schema.TypeString,
										Optional: true,
									},
								},
							},
						},
						"tmpfs_options": {
							Type:     schema.TypeList,
							Optional: true,
							MaxItems: 1,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"mode": {
										Type:     schema.TypeInt,
										Optional: true,
									},
									"size_bytes": {
										Type:     schema.TypeInt,
										Optional: true,
									},
								},
							},
						},
						"volume_options": {
							Type:     schema.TypeList,
							Optional: true,
							MaxItems: 1,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"driver_name": {
										Type:     schema.TypeString,
										Optional: true,
									},
									"driver_options": {
										Type:     schema.TypeMap,
										Optional: true,
										Elem:     &schema.Schema{Type: schema.TypeString},
									},
									"labels": labelsSchema(),
									"no_copy": {
										Type:     schema.TypeBool,
										Optional: true,
									},
								},
							},
						},
					},
				},
			},
			"must_run": {
				Type:     schema.TypeBool,
				Optional: true,
				Default:  true,
			},
			"network_mode": {
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
				ForceNew: true,
			},
			"networks_advanced": {
				Type:     schema.TypeSet,
				Optional: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"name": {
							Type:     schema.TypeString,
							Required: true,
						},
						"aliases": {
							Type:     schema.TypeSet,
							Optional: true,
							Elem:     &schema.Schema{Type: schema.TypeString},
						},
						"ipv4_address": {
							Type:     schema.TypeString,
							Optional: true,
						},
						"ipv6_address": {
							Type:     schema.TypeString,
							Optional: true,
						},
					},
				},
			},
			"pid_mode": {
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
				ForceNew: true,
			},
			"ports": {
				Type:     schema.TypeList,
				Optional: true,
				ForceNew: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"internal": {
							Type:     schema.TypeInt,
							Required: true,
						},
						"external": {
							Type:     schema.TypeInt,
							Optional: true,
							Computed: true,
						},
						"ip": {
							Type:     schema.TypeString,
							Optional: true,
							Default:  "0.0.0.0",
						},
						"protocol": {
							Type:     schema.TypeString,
							Optional: true,
							Default:  "tcp",
						},
					},
				},
			},
			"privileged": {
				Type:     schema.TypeBool,
				Optional: true,
				Default:  false,
				ForceNew: true,
			},
			"publish_all_ports": {
				Type:     schema.TypeBool,
				Optional: true,
				Default:  false,
			},
			"read_only": {
				Type:     schema.TypeBool,
				Optional: true,
				Default:  false,
			},
			"remove_volumes": {
				Type:     schema.TypeBool,
				Optional: true,
				Default:  true,
			},
			"restart": {
				Type:     schema.TypeString,
				Optional: true,
				Default:  "no",
			},
			"rm": {
				Type:     schema.TypeBool,
				Optional: true,
				Default:  false,
			},
			"runtime": {
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
				ForceNew: true,
			},
			"security_opts": {
				Type:     schema.TypeSet,
				Optional: true,
				Computed: true,
				ForceNew: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
			},
			"shm_size": {
				Type:     schema.TypeInt,
				Optional: true,
				Computed: true,
				ForceNew: true,
			},
			"start": {
				Type:     schema.TypeBool,
				Optional: true,
				Default:  true,
			},
			"stdin_open": {
				Type:     schema.TypeBool,
				Optional: true,
				Default:  false,
			},
			"stop_signal": {
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
			},
			"stop_timeout": {
				Type:     schema.TypeInt,
				Optional: true,
				Computed: true,
			},
			"storage_opts": {
				Type:     schema.TypeMap,
				Optional: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
			},
			"sysctls": {
				Type:     schema.TypeMap,
				Optional: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
			},
			"tmpfs": {
				Type:     schema.TypeMap,
				Optional: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
			},
			"tty": {
				Type:     schema.TypeBool,
				Optional: true,
				Default:  false,
			},
			"ulimit": {
				Type:     schema.TypeSet,
				Optional: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"hard": {
							Type:     schema.TypeInt,
							Required: true,
						},
						"name": {
							Type:     schema.TypeString,
							Required: true,
						},
						"soft": {
							Type:     schema.TypeInt,
							Required: true,
						},
					},
				},
			},
			"upload": {
				Type:     schema.TypeSet,
				Optional: true,
				ForceNew: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"file": {
							Type:     schema.TypeString,
							Required: true,
						},
						"content": {
							Type:     schema.TypeString,
							Optional: true,
						},
						"content_base64": {
							Type:     schema.TypeString,
							Optional: true,
						},
						"executable": {
							Type:     schema.TypeBool,
							Optional: true,
							Default:  false,
						},
						"source": {
							Type:     schema.TypeString,
							Optional: true,
						},
						"source_hash": {
							Type:     schema.TypeString,
							Optional: true,
						},
					},
				},
			},
			"user": {
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
			},
			"userns_mode": {
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
			},
			"volumes": {
				Type:     schema.TypeSet,
				Optional: true,
				ForceNew: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"container_path": {
							Type:     schema.TypeString,
							Optional: true,
						},
						"from_container": {
							Type:     schema.TypeString,
							Optional: true,
						},
						"host_path": {
							Type:     schema.TypeString,
							Optional: true,
						},
						"read_only": {
							Type:     schema.TypeBool,
							Optional: true,
						},
						"volume_name": {
							Type:     schema.TypeString,
							Optional: true,
						},
					},
				},
			},
			"wait": {
				Type:     schema.TypeBool,
				Optional: true,
				Default:  false,
			},
			"wait_timeout": {
				Type:     schema.TypeInt,
				Optional: true,
				Default:  60,
			},
			"working_dir": {
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
				ForceNew: true,
			},

			// Read-only
			"bridge": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"container_logs": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"exit_code": {
				Type:     schema.TypeInt,
				Computed: true,
			},
			"network_data": {
				Type:     schema.TypeList,
				Computed: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"gateway": {
							Type:     schema.TypeString,
							Computed: true,
						},
						"global_ipv6_address": {
							Type:     schema.TypeString,
							Computed: true,
						},
						"global_ipv6_prefix_length": {
							Type:     schema.TypeInt,
							Computed: true,
						},
						"ip_address": {
							Type:     schema.TypeString,
							Computed: true,
						},
						"ip_prefix_length": {
							Type:     schema.TypeInt,
							Computed: true,
						},
						"ipv6_gateway": {
							Type:     schema.TypeString,
							Computed: true,
						},
						"mac_address": {
							Type:     schema.TypeString,
							Computed: true,
						},
						"network_name": {
							Type:     schema.TypeString,
							Computed: true,
						},
					},
				},
			},
		},
	}
}

func resourcePodmanContainerCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	config := getClient(meta)
	cli := config.Client

	imageName := d.Get("image").(string)
	containerName := d.Get("name").(string)

	// If the image is a sha256 digest, resolve it to a repo tag for Podman compat API
	if strings.HasPrefix(imageName, "sha256:") {
		inspectResult, _, err := cli.ImageInspectWithRaw(ctx, imageName)
		if err == nil && len(inspectResult.RepoTags) > 0 {
			imageName = inspectResult.RepoTags[0]
		}
	}

	// Build container config
	containerConfig := &container.Config{
		Image: imageName,
	}

	if v, ok := d.GetOk("command"); ok {
		containerConfig.Cmd = strslice.StrSlice(stringListToSlice(v))
	}
	if v, ok := d.GetOk("entrypoint"); ok {
		containerConfig.Entrypoint = strslice.StrSlice(stringListToSlice(v))
	}
	if v, ok := d.GetOk("env"); ok {
		containerConfig.Env = stringSetToSlice(v)
	}
	if v, ok := d.GetOk("hostname"); ok {
		containerConfig.Hostname = v.(string)
	}
	if v, ok := d.GetOk("domainname"); ok {
		containerConfig.Domainname = v.(string)
	}
	if v, ok := d.GetOk("user"); ok {
		containerConfig.User = v.(string)
	}
	if v, ok := d.GetOk("working_dir"); ok {
		containerConfig.WorkingDir = v.(string)
	}
	if v, ok := d.GetOk("labels"); ok {
		containerConfig.Labels = labelsToMap(v)
	}
	if v, ok := d.GetOk("stop_signal"); ok {
		containerConfig.StopSignal = v.(string)
	}
	if v, ok := d.GetOk("stop_timeout"); ok {
		timeout := v.(int)
		containerConfig.StopTimeout = &timeout
	}

	containerConfig.AttachStdin = d.Get("stdin_open").(bool)
	containerConfig.OpenStdin = d.Get("stdin_open").(bool)
	containerConfig.Tty = d.Get("tty").(bool)

	// Healthcheck
	if v, ok := d.GetOk("healthcheck"); ok {
		hcList := v.([]interface{})
		if len(hcList) > 0 {
			hcMap := hcList[0].(map[string]interface{})
			hc := &container.HealthConfig{
				Test: stringListToSlice(hcMap["test"]),
			}
			if interval, err := time.ParseDuration(hcMap["interval"].(string)); err == nil {
				hc.Interval = interval
			}
			if timeout, err := time.ParseDuration(hcMap["timeout"].(string)); err == nil {
				hc.Timeout = timeout
			}
			if startPeriod, err := time.ParseDuration(hcMap["start_period"].(string)); err == nil {
				hc.StartPeriod = startPeriod
			}
			hc.Retries = hcMap["retries"].(int)
			containerConfig.Healthcheck = hc
		}
	}

	// Exposed ports and port bindings
	exposedPorts := nat.PortSet{}
	portBindings := nat.PortMap{}
	if v, ok := d.GetOk("ports"); ok {
		portsList := v.([]interface{})
		for _, portRaw := range portsList {
			portMap := portRaw.(map[string]interface{})
			internal := portMap["internal"].(int)
			protocol := portMap["protocol"].(string)
			port, err := nat.NewPort(protocol, strconv.Itoa(internal))
			if err != nil {
				return diag.FromErr(fmt.Errorf("invalid port specification: %w", err))
			}
			exposedPorts[port] = struct{}{}
			binding := nat.PortBinding{
				HostIP: portMap["ip"].(string),
			}
			if ext, ok := portMap["external"]; ok && ext.(int) > 0 {
				binding.HostPort = strconv.Itoa(ext.(int))
			}
			portBindings[port] = append(portBindings[port], binding)
		}
	}
	containerConfig.ExposedPorts = exposedPorts

	// Volumes from volume blocks
	volBinds := []string{}
	if v, ok := d.GetOk("volumes"); ok {
		for _, volRaw := range v.(*schema.Set).List() {
			vol := volRaw.(map[string]interface{})
			fromContainer := vol["from_container"].(string)
			if fromContainer != "" {
				continue
			}
			hostPath := vol["host_path"].(string)
			containerPath := vol["container_path"].(string)
			volumeName := vol["volume_name"].(string)
			readOnly := vol["read_only"].(bool)

			source := hostPath
			if source == "" {
				source = volumeName
			}
			if source != "" && containerPath != "" {
				bind := source + ":" + containerPath
				if readOnly {
					bind += ":ro"
				}
				volBinds = append(volBinds, bind)
			} else if containerPath != "" {
				volBinds = append(volBinds, containerPath)
			}
		}
	}

	// Volumes from containers
	volumesFrom := []string{}
	if v, ok := d.GetOk("volumes"); ok {
		for _, volRaw := range v.(*schema.Set).List() {
			vol := volRaw.(map[string]interface{})
			fromContainer := vol["from_container"].(string)
			if fromContainer != "" {
				mode := ""
				if vol["read_only"].(bool) {
					mode = ":ro"
				}
				volumesFrom = append(volumesFrom, fromContainer+mode)
			}
		}
	}

	// Host config
	hostConfig := &container.HostConfig{
		Binds:        volBinds,
		VolumesFrom:  volumesFrom,
		PortBindings: portBindings,
		PublishAllPorts: d.Get("publish_all_ports").(bool),
		Privileged:     d.Get("privileged").(bool),
		ReadonlyRootfs: d.Get("read_only").(bool),
		AutoRemove:     d.Get("rm").(bool),
	}

	// Restart policy
	restartPolicy := d.Get("restart").(string)
	maxRetry := d.Get("max_retry_count").(int)
	hostConfig.RestartPolicy = container.RestartPolicy{
		Name:              container.RestartPolicyMode(restartPolicy),
		MaximumRetryCount: maxRetry,
	}

	// Resources
	hostConfig.Resources = container.Resources{}
	if v, ok := d.GetOk("cpu_shares"); ok {
		hostConfig.Resources.CPUShares = int64(v.(int))
	}
	if v, ok := d.GetOk("cpu_period"); ok {
		hostConfig.Resources.CPUPeriod = int64(v.(int))
	}
	if v, ok := d.GetOk("cpu_quota"); ok {
		hostConfig.Resources.CPUQuota = int64(v.(int))
	}
	if v, ok := d.GetOk("cpu_set"); ok {
		hostConfig.Resources.CpusetCpus = v.(string)
	}
	if v, ok := d.GetOk("memory"); ok {
		hostConfig.Resources.Memory = int64(v.(int))
	}
	if v, ok := d.GetOk("memory_swap"); ok {
		hostConfig.Resources.MemorySwap = int64(v.(int))
	}

	// Capabilities
	if v, ok := d.GetOk("capabilities"); ok {
		capsList := v.([]interface{})
		if len(capsList) > 0 {
			capsMap := capsList[0].(map[string]interface{})
			var capAdd, capDrop []string
			if add, ok := capsMap["add"]; ok {
				capAdd = stringSetToSlice(add)
			}
			if drop, ok := capsMap["drop"]; ok {
				capDrop = stringSetToSlice(drop)
			}
			hostConfig.CapAdd = strslice.StrSlice(capAdd)
			hostConfig.CapDrop = strslice.StrSlice(capDrop)
		}
	}

	// Devices
	if v, ok := d.GetOk("devices"); ok {
		for _, devRaw := range v.(*schema.Set).List() {
			dev := devRaw.(map[string]interface{})
			cPath := dev["container_path"].(string)
			if cPath == "" {
				cPath = dev["host_path"].(string)
			}
			hostConfig.Resources.Devices = append(hostConfig.Resources.Devices, container.DeviceMapping{
				PathOnHost:        dev["host_path"].(string),
				PathInContainer:   cPath,
				CgroupPermissions: dev["permissions"].(string),
			})
		}
	}

	// Ulimits
	if v, ok := d.GetOk("ulimit"); ok {
		for _, uRaw := range v.(*schema.Set).List() {
			u := uRaw.(map[string]interface{})
			hostConfig.Resources.Ulimits = append(hostConfig.Resources.Ulimits, &units.Ulimit{
				Name: u["name"].(string),
				Hard: int64(u["hard"].(int)),
				Soft: int64(u["soft"].(int)),
			})
		}
	}

	// DNS
	if v, ok := d.GetOk("dns"); ok {
		hostConfig.DNS = stringSetToSlice(v)
	}
	if v, ok := d.GetOk("dns_opts"); ok {
		hostConfig.DNSOptions = stringSetToSlice(v)
	}
	if v, ok := d.GetOk("dns_search"); ok {
		hostConfig.DNSSearch = stringSetToSlice(v)
	}

	// Extra hosts
	if v, ok := d.GetOk("host"); ok {
		for _, hRaw := range v.(*schema.Set).List() {
			h := hRaw.(map[string]interface{})
			hostConfig.ExtraHosts = append(hostConfig.ExtraHosts, h["host"].(string)+":"+h["ip"].(string))
		}
	}

	// Network mode
	if v, ok := d.GetOk("network_mode"); ok {
		hostConfig.NetworkMode = container.NetworkMode(v.(string))
	}

	// PID mode
	if v, ok := d.GetOk("pid_mode"); ok {
		hostConfig.PidMode = container.PidMode(v.(string))
	}

	// IPC mode
	if v, ok := d.GetOk("ipc_mode"); ok {
		hostConfig.IpcMode = container.IpcMode(v.(string))
	}

	// Userns mode
	if v, ok := d.GetOk("userns_mode"); ok {
		hostConfig.UsernsMode = container.UsernsMode(v.(string))
	}

	// Cgroupns mode
	if v, ok := d.GetOk("cgroupns_mode"); ok {
		mode := container.CgroupnsMode(v.(string))
		hostConfig.CgroupnsMode = mode
	}

	// Runtime
	if v, ok := d.GetOk("runtime"); ok {
		hostConfig.Runtime = v.(string)
	}

	// Shm size
	if v, ok := d.GetOk("shm_size"); ok {
		hostConfig.ShmSize = int64(v.(int))
	}

	// Security opts
	if v, ok := d.GetOk("security_opts"); ok {
		hostConfig.SecurityOpt = stringSetToSlice(v)
	}

	// Sysctls
	if v, ok := d.GetOk("sysctls"); ok {
		hostConfig.Sysctls = mapStringInterfaceToStringString(v.(map[string]interface{}))
	}

	// Tmpfs
	if v, ok := d.GetOk("tmpfs"); ok {
		hostConfig.Tmpfs = mapStringInterfaceToStringString(v.(map[string]interface{}))
	}

	// Storage opts
	if v, ok := d.GetOk("storage_opts"); ok {
		hostConfig.StorageOpt = mapStringInterfaceToStringString(v.(map[string]interface{}))
	}

	// Log driver
	if v, ok := d.GetOk("log_driver"); ok {
		hostConfig.LogConfig.Type = v.(string)
	}
	if v, ok := d.GetOk("log_opts"); ok {
		hostConfig.LogConfig.Config = mapStringInterfaceToStringString(v.(map[string]interface{}))
	}

	// Group add
	if v, ok := d.GetOk("group_add"); ok {
		hostConfig.GroupAdd = stringSetToSlice(v)
	}

	// Init
	if v, ok := d.GetOk("init"); ok {
		init := v.(bool)
		hostConfig.Init = &init
	}

	// Mounts
	if v, ok := d.GetOk("mounts"); ok {
		for _, mRaw := range v.(*schema.Set).List() {
			m := mRaw.(map[string]interface{})
			mnt := mount.Mount{
				Target:   m["target"].(string),
				Type:     mount.Type(m["type"].(string)),
				ReadOnly: m["read_only"].(bool),
				Source:   m["source"].(string),
			}
			if bOpts, ok := m["bind_options"].([]interface{}); ok && len(bOpts) > 0 {
				bMap := bOpts[0].(map[string]interface{})
				mnt.BindOptions = &mount.BindOptions{
					Propagation: mount.Propagation(bMap["propagation"].(string)),
				}
			}
			if tOpts, ok := m["tmpfs_options"].([]interface{}); ok && len(tOpts) > 0 {
				tMap := tOpts[0].(map[string]interface{})
				mnt.TmpfsOptions = &mount.TmpfsOptions{
					SizeBytes: int64(tMap["size_bytes"].(int)),
					Mode:      os.FileMode(tMap["mode"].(int)),
				}
			}
			if vOpts, ok := m["volume_options"].([]interface{}); ok && len(vOpts) > 0 {
				vMap := vOpts[0].(map[string]interface{})
				volOpts := &mount.VolumeOptions{
					NoCopy: vMap["no_copy"].(bool),
				}
				if dn, ok := vMap["driver_name"].(string); ok && dn != "" {
					volOpts.DriverConfig = &mount.Driver{
						Name: dn,
					}
					if do, ok := vMap["driver_options"].(map[string]interface{}); ok {
						volOpts.DriverConfig.Options = mapStringInterfaceToStringString(do)
					}
				}
				if lbls, ok := vMap["labels"]; ok {
					volOpts.Labels = labelsToMap(lbls)
				}
				mnt.VolumeOptions = volOpts
			}
			hostConfig.Mounts = append(hostConfig.Mounts, mnt)
		}
	}

	// Networking config - build from first network in networks_advanced if present
	networkingConfig := &network.NetworkingConfig{}
	if v, ok := d.GetOk("networks_advanced"); ok {
		endpointsConfig := make(map[string]*network.EndpointSettings)
		nets := v.(*schema.Set).List()
		if len(nets) > 0 {
			// Only the first network can be attached at create time
			first := nets[0].(map[string]interface{})
			epConfig := &network.EndpointSettings{}
			epConfig.IPAMConfig = &network.EndpointIPAMConfig{}
			if ipv4, ok := first["ipv4_address"].(string); ok && ipv4 != "" {
				epConfig.IPAMConfig.IPv4Address = ipv4
			}
			if ipv6, ok := first["ipv6_address"].(string); ok && ipv6 != "" {
				epConfig.IPAMConfig.IPv6Address = ipv6
			}
			if aliases, ok := first["aliases"]; ok {
				epConfig.Aliases = stringSetToSlice(aliases)
			}
			endpointsConfig[first["name"].(string)] = epConfig
		}
		networkingConfig.EndpointsConfig = endpointsConfig
	}

	// Create container
	body, err := cli.ContainerCreate(ctx, containerConfig, hostConfig, networkingConfig, nil, containerName)
	if err != nil {
		return diag.FromErr(fmt.Errorf("error creating container %s: %w", containerName, err))
	}

	d.SetId(body.ID)

	// Handle uploads before starting
	if v, ok := d.GetOk("upload"); ok {
		for _, uploadRaw := range v.(*schema.Set).List() {
			upload := uploadRaw.(map[string]interface{})
			if err := uploadFileToContainer(ctx, cli, body.ID, upload); err != nil {
				return diag.FromErr(fmt.Errorf("error uploading file to container: %w", err))
			}
		}
	}

	// Connect additional networks (skip the first since it was attached at create)
	if v, ok := d.GetOk("networks_advanced"); ok {
		nets := v.(*schema.Set).List()
		for i, netRaw := range nets {
			if i == 0 {
				continue
			}
			n := netRaw.(map[string]interface{})
			epConfig := &network.EndpointSettings{}
			epConfig.IPAMConfig = &network.EndpointIPAMConfig{}
			if ipv4, ok := n["ipv4_address"].(string); ok && ipv4 != "" {
				epConfig.IPAMConfig.IPv4Address = ipv4
			}
			if ipv6, ok := n["ipv6_address"].(string); ok && ipv6 != "" {
				epConfig.IPAMConfig.IPv6Address = ipv6
			}
			if aliases, ok := n["aliases"]; ok {
				epConfig.Aliases = stringSetToSlice(aliases)
			}
			if err := cli.NetworkConnect(ctx, n["name"].(string), body.ID, epConfig); err != nil {
				return diag.FromErr(fmt.Errorf("error connecting container to network %s: %w", n["name"].(string), err))
			}
		}
	}

	// Handle attach mode
	if d.Get("attach").(bool) {
		attachOpts := container.AttachOptions{
			Stream: true,
			Stdout: true,
			Stderr: true,
			Logs:   d.Get("logs").(bool),
		}
		attachResp, err := cli.ContainerAttach(ctx, body.ID, attachOpts)
		if err != nil {
			return diag.FromErr(fmt.Errorf("error attaching to container %s: %w", containerName, err))
		}
		defer attachResp.Close()

		// Start the container
		if err := cli.ContainerStart(ctx, body.ID, container.StartOptions{}); err != nil {
			return diag.FromErr(fmt.Errorf("error starting container %s: %w", containerName, err))
		}

		// Read logs
		var logBuf bytes.Buffer
		_, _ = io.Copy(&logBuf, attachResp.Reader)

		// Wait for the container to finish
		waitCh, errCh := cli.ContainerWait(ctx, body.ID, container.WaitConditionNotRunning)
		select {
		case waitResult := <-waitCh:
			d.Set("exit_code", int(waitResult.StatusCode))
		case err := <-errCh:
			if err != nil {
				return diag.FromErr(fmt.Errorf("error waiting for container %s: %w", containerName, err))
			}
		}

		d.Set("container_logs", logBuf.String())
	} else if d.Get("start").(bool) {
		// Start the container
		if err := cli.ContainerStart(ctx, body.ID, container.StartOptions{}); err != nil {
			return diag.FromErr(fmt.Errorf("error starting container %s: %w", containerName, err))
		}

		if d.Get("wait").(bool) {
			waitTimeout := d.Get("wait_timeout").(int)
			waitCtx, cancel := context.WithTimeout(ctx, time.Duration(waitTimeout)*time.Second)
			defer cancel()

			waitCh, errCh := cli.ContainerWait(waitCtx, body.ID, container.WaitConditionNotRunning)
			select {
			case waitResult := <-waitCh:
				d.Set("exit_code", int(waitResult.StatusCode))
			case err := <-errCh:
				if err != nil {
					return diag.FromErr(fmt.Errorf("error waiting for container %s: %w", containerName, err))
				}
			case <-waitCtx.Done():
				return diag.FromErr(fmt.Errorf("timeout waiting for container %s to finish", containerName))
			}
		}
	}

	// Fetch logs if requested
	if d.Get("logs").(bool) && !d.Get("attach").(bool) {
		logsOpts := container.LogsOptions{
			ShowStdout: true,
			ShowStderr: true,
		}
		logsReader, err := cli.ContainerLogs(ctx, body.ID, logsOpts)
		if err != nil {
			return diag.FromErr(fmt.Errorf("error reading logs for container %s: %w", containerName, err))
		}
		defer logsReader.Close()
		var logBuf bytes.Buffer
		_, _ = io.Copy(&logBuf, logsReader)
		d.Set("container_logs", logBuf.String())
	}

	return resourcePodmanContainerRead(ctx, d, meta)
}

func resourcePodmanContainerRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	config := getClient(meta)
	cli := config.Client

	containerID := d.Id()

	containerJSON, err := cli.ContainerInspect(ctx, containerID)
	if err != nil {
		// If container not found, remove from state
		if strings.Contains(err.Error(), "No such container") || strings.Contains(err.Error(), "not found") {
			d.SetId("")
			return nil
		}
		return diag.FromErr(fmt.Errorf("error inspecting container %s: %w", containerID, err))
	}

	c := containerJSON.Config
	hc := containerJSON.HostConfig

	d.Set("name", strings.TrimPrefix(containerJSON.Name, "/"))
	// Preserve the image reference from config/state to avoid sha256 vs name drift
	if currentImage := d.Get("image").(string); currentImage == "" {
		d.Set("image", c.Image)
	}
	d.Set("hostname", c.Hostname)
	d.Set("domainname", c.Domainname)
	d.Set("user", c.User)
	d.Set("tty", c.Tty)
	d.Set("stdin_open", c.OpenStdin)
	d.Set("working_dir", c.WorkingDir)

	if c.Cmd != nil {
		d.Set("command", []string(c.Cmd))
	}
	if c.Entrypoint != nil {
		d.Set("entrypoint", []string(c.Entrypoint))
	}
	if c.Env != nil {
		d.Set("env", c.Env)
	}

	if c.StopSignal != "" {
		d.Set("stop_signal", c.StopSignal)
	}
	if c.StopTimeout != nil {
		d.Set("stop_timeout", *c.StopTimeout)
	}

	// Labels
	if c.Labels != nil {
		d.Set("labels", mapToLabelsSet(c.Labels))
	}

	// Healthcheck
	if c.Healthcheck != nil {
		hcData := []interface{}{
			map[string]interface{}{
				"test":         c.Healthcheck.Test,
				"interval":     c.Healthcheck.Interval.String(),
				"timeout":      c.Healthcheck.Timeout.String(),
				"start_period": c.Healthcheck.StartPeriod.String(),
				"retries":      c.Healthcheck.Retries,
			},
		}
		d.Set("healthcheck", hcData)
	}

	// Host config fields
	if hc != nil {
		d.Set("privileged", hc.Privileged)
		d.Set("publish_all_ports", hc.PublishAllPorts)
		d.Set("read_only", hc.ReadonlyRootfs)
		d.Set("rm", hc.AutoRemove)
		d.Set("runtime", hc.Runtime)
		d.Set("shm_size", int(hc.ShmSize))
		d.Set("network_mode", string(hc.NetworkMode))
		d.Set("pid_mode", string(hc.PidMode))
		d.Set("ipc_mode", string(hc.IpcMode))
		d.Set("userns_mode", string(hc.UsernsMode))
		d.Set("cgroupns_mode", string(hc.CgroupnsMode))
		d.Set("log_driver", hc.LogConfig.Type)

		if hc.LogConfig.Config != nil {
			d.Set("log_opts", hc.LogConfig.Config)
		}

		// Restart policy
		d.Set("restart", hc.RestartPolicy.Name)
		d.Set("max_retry_count", hc.RestartPolicy.MaximumRetryCount)

		// Resources
		d.Set("cpu_shares", int(hc.Resources.CPUShares))
		d.Set("cpu_period", int(hc.Resources.CPUPeriod))
		d.Set("cpu_quota", int(hc.Resources.CPUQuota))
		d.Set("cpu_set", hc.Resources.CpusetCpus)
		d.Set("memory", int(hc.Resources.Memory))
		d.Set("memory_swap", int(hc.Resources.MemorySwap))

		// Security opts
		if hc.SecurityOpt != nil {
			d.Set("security_opts", hc.SecurityOpt)
		}

		// DNS
		if hc.DNS != nil {
			d.Set("dns", hc.DNS)
		}
		if hc.DNSOptions != nil {
			d.Set("dns_opts", hc.DNSOptions)
		}
		if hc.DNSSearch != nil {
			d.Set("dns_search", hc.DNSSearch)
		}

		// Group add
		if hc.GroupAdd != nil {
			d.Set("group_add", hc.GroupAdd)
		}

		// Sysctls
		if hc.Sysctls != nil {
			d.Set("sysctls", hc.Sysctls)
		}

		// Tmpfs
		if hc.Tmpfs != nil {
			d.Set("tmpfs", hc.Tmpfs)
		}

		// Storage opts
		if hc.StorageOpt != nil {
			d.Set("storage_opts", hc.StorageOpt)
		}

		// Init
		if hc.Init != nil {
			d.Set("init", *hc.Init)
		}

		// Capabilities - only set if user originally configured them
		if _, ok := d.GetOk("capabilities"); ok {
			if hc.CapAdd != nil || hc.CapDrop != nil {
				caps := make([]interface{}, 1)
				capsMap := map[string]interface{}{}
				if hc.CapAdd != nil {
					capsMap["add"] = []string(hc.CapAdd)
				}
				if hc.CapDrop != nil {
					capsMap["drop"] = []string(hc.CapDrop)
				}
				caps[0] = capsMap
				d.Set("capabilities", caps)
			}
		}

		// Devices
		if hc.Resources.Devices != nil {
			devices := make([]interface{}, len(hc.Resources.Devices))
			for i, dev := range hc.Resources.Devices {
				devices[i] = map[string]interface{}{
					"host_path":      dev.PathOnHost,
					"container_path": dev.PathInContainer,
					"permissions":    dev.CgroupPermissions,
				}
			}
			d.Set("devices", devices)
		}

		// Ulimits - only set if user originally configured them
		if _, ok := d.GetOk("ulimit"); ok && hc.Resources.Ulimits != nil {
			ulimits := make([]interface{}, len(hc.Resources.Ulimits))
			for i, u := range hc.Resources.Ulimits {
				ulimits[i] = map[string]interface{}{
					"name": u.Name,
					"hard": int(u.Hard),
					"soft": int(u.Soft),
				}
			}
			d.Set("ulimit", ulimits)
		}

		// Extra hosts
		if hc.ExtraHosts != nil {
			hosts := make([]interface{}, len(hc.ExtraHosts))
			for i, eh := range hc.ExtraHosts {
				parts := strings.SplitN(eh, ":", 2)
				hostMap := map[string]interface{}{
					"host": parts[0],
					"ip":   "",
				}
				if len(parts) == 2 {
					hostMap["ip"] = parts[1]
				}
				hosts[i] = hostMap
			}
			d.Set("host", hosts)
		}

		// Ports
		if containerJSON.NetworkSettings != nil && containerJSON.NetworkSettings.Ports != nil {
			portsList := make([]interface{}, 0)
			for port, bindings := range containerJSON.NetworkSettings.Ports {
				for _, binding := range bindings {
					extPort := 0
					if binding.HostPort != "" {
						extPort, _ = strconv.Atoi(binding.HostPort)
					}
					portsList = append(portsList, map[string]interface{}{
						"internal": port.Int(),
						"external": extPort,
						"ip":       binding.HostIP,
						"protocol": port.Proto(),
					})
				}
				if len(bindings) == 0 {
					portsList = append(portsList, map[string]interface{}{
						"internal": port.Int(),
						"external": 0,
						"ip":       "0.0.0.0",
						"protocol": port.Proto(),
					})
				}
			}
			d.Set("ports", portsList)
		}

		// Mounts
		if hc.Mounts != nil {
			mountsList := make([]interface{}, len(hc.Mounts))
			for i, m := range hc.Mounts {
				mMap := map[string]interface{}{
					"target":    m.Target,
					"type":      string(m.Type),
					"read_only": m.ReadOnly,
					"source":    m.Source,
				}
				if m.BindOptions != nil {
					mMap["bind_options"] = []interface{}{
						map[string]interface{}{
							"propagation": string(m.BindOptions.Propagation),
						},
					}
				} else {
					mMap["bind_options"] = []interface{}{}
				}
				if m.TmpfsOptions != nil {
					mMap["tmpfs_options"] = []interface{}{
						map[string]interface{}{
							"mode":       int(m.TmpfsOptions.Mode),
							"size_bytes": int(m.TmpfsOptions.SizeBytes),
						},
					}
				} else {
					mMap["tmpfs_options"] = []interface{}{}
				}
				if m.VolumeOptions != nil {
					voMap := map[string]interface{}{
						"no_copy": m.VolumeOptions.NoCopy,
					}
					if m.VolumeOptions.DriverConfig != nil {
						voMap["driver_name"] = m.VolumeOptions.DriverConfig.Name
						voMap["driver_options"] = m.VolumeOptions.DriverConfig.Options
					} else {
						voMap["driver_name"] = ""
						voMap["driver_options"] = map[string]string{}
					}
					if m.VolumeOptions.Labels != nil {
						voMap["labels"] = mapToLabelsSet(m.VolumeOptions.Labels)
					} else {
						voMap["labels"] = []interface{}{}
					}
					mMap["volume_options"] = []interface{}{voMap}
				} else {
					mMap["volume_options"] = []interface{}{}
				}
				mountsList[i] = mMap
			}
			d.Set("mounts", mountsList)
		}

		// Volumes
		if hc.Binds != nil {
			volumesList := make([]interface{}, 0)
			for _, bind := range hc.Binds {
				parts := strings.SplitN(bind, ":", 3)
				volMap := map[string]interface{}{
					"host_path":      "",
					"container_path": "",
					"volume_name":    "",
					"read_only":      false,
					"from_container": "",
				}
				if len(parts) >= 2 {
					if strings.HasPrefix(parts[0], "/") {
						volMap["host_path"] = parts[0]
					} else {
						volMap["volume_name"] = parts[0]
					}
					volMap["container_path"] = parts[1]
					if len(parts) == 3 && parts[2] == "ro" {
						volMap["read_only"] = true
					}
				} else if len(parts) == 1 {
					volMap["container_path"] = parts[0]
				}
				volumesList = append(volumesList, volMap)
			}
			for _, vf := range hc.VolumesFrom {
				parts := strings.SplitN(vf, ":", 2)
				volMap := map[string]interface{}{
					"host_path":      "",
					"container_path": "",
					"volume_name":    "",
					"read_only":      false,
					"from_container": parts[0],
				}
				if len(parts) == 2 && parts[1] == "ro" {
					volMap["read_only"] = true
				}
				volumesList = append(volumesList, volMap)
			}
			d.Set("volumes", volumesList)
		}
	}

	// Network data
	if containerJSON.NetworkSettings != nil {
		d.Set("bridge", containerJSON.NetworkSettings.Bridge)

		networkData := make([]interface{}, 0)
		for name, net := range containerJSON.NetworkSettings.Networks {
			nd := map[string]interface{}{
				"network_name":              name,
				"ip_address":                net.IPAddress,
				"ip_prefix_length":          net.IPPrefixLen,
				"gateway":                   net.Gateway,
				"global_ipv6_address":       net.GlobalIPv6Address,
				"global_ipv6_prefix_length": net.GlobalIPv6PrefixLen,
				"ipv6_gateway":              net.IPv6Gateway,
				"mac_address":               net.MacAddress,
			}
			networkData = append(networkData, nd)
		}
		d.Set("network_data", networkData)

		// Populate networks_advanced from network settings
		if _, ok := d.GetOk("networks_advanced"); ok {
			netsAdvanced := make([]interface{}, 0)
			for name, net := range containerJSON.NetworkSettings.Networks {
				nMap := map[string]interface{}{
					"name": name,
				}
				if net.IPAMConfig != nil {
					nMap["ipv4_address"] = net.IPAMConfig.IPv4Address
					nMap["ipv6_address"] = net.IPAMConfig.IPv6Address
				} else {
					nMap["ipv4_address"] = ""
					nMap["ipv6_address"] = ""
				}
				if net.Aliases != nil {
					nMap["aliases"] = net.Aliases
				} else {
					nMap["aliases"] = []string{}
				}
				netsAdvanced = append(netsAdvanced, nMap)
			}
			d.Set("networks_advanced", netsAdvanced)
		}
	}

	// Container state
	if containerJSON.State != nil {
		d.Set("exit_code", containerJSON.State.ExitCode)
	}

	// Fetch container logs if logs is enabled
	if d.Get("logs").(bool) {
		cli := config.Client
		logsOpts := container.LogsOptions{
			ShowStdout: true,
			ShowStderr: true,
		}
		logsReader, err := cli.ContainerLogs(ctx, containerID, logsOpts)
		if err == nil {
			defer logsReader.Close()
			var logBuf bytes.Buffer
			_, _ = io.Copy(&logBuf, logsReader)
			d.Set("container_logs", logBuf.String())
		}
	}

	return nil
}

func resourcePodmanContainerUpdate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	config := getClient(meta)
	cli := config.Client
	containerID := d.Id()

	// Update container resources if changed
	if d.HasChanges("cpu_shares", "cpu_period", "cpu_quota", "cpu_set", "memory", "memory_swap") {
		resources := container.Resources{}
		if v, ok := d.GetOk("cpu_shares"); ok {
			resources.CPUShares = int64(v.(int))
		}
		if v, ok := d.GetOk("cpu_period"); ok {
			resources.CPUPeriod = int64(v.(int))
		}
		if v, ok := d.GetOk("cpu_quota"); ok {
			resources.CPUQuota = int64(v.(int))
		}
		if v, ok := d.GetOk("cpu_set"); ok {
			resources.CpusetCpus = v.(string)
		}
		if v, ok := d.GetOk("memory"); ok {
			resources.Memory = int64(v.(int))
		}
		if v, ok := d.GetOk("memory_swap"); ok {
			resources.MemorySwap = int64(v.(int))
		}
		updateConfig := container.UpdateConfig{
			Resources: resources,
		}
		if _, err := cli.ContainerUpdate(ctx, containerID, updateConfig); err != nil {
			return diag.FromErr(fmt.Errorf("error updating container %s resources: %w", containerID, err))
		}
	}

	// Update restart policy if changed
	if d.HasChanges("restart", "max_retry_count") {
		restartPolicy := d.Get("restart").(string)
		maxRetry := d.Get("max_retry_count").(int)
		updateConfig := container.UpdateConfig{
			RestartPolicy: container.RestartPolicy{
				Name:              container.RestartPolicyMode(restartPolicy),
				MaximumRetryCount: maxRetry,
			},
		}
		if _, err := cli.ContainerUpdate(ctx, containerID, updateConfig); err != nil {
			return diag.FromErr(fmt.Errorf("error updating container %s restart policy: %w", containerID, err))
		}
	}

	// Handle network changes
	if d.HasChange("networks_advanced") {
		oldRaw, newRaw := d.GetChange("networks_advanced")
		oldNets := oldRaw.(*schema.Set)
		newNets := newRaw.(*schema.Set)

		// Disconnect removed networks
		for _, nRaw := range oldNets.Difference(newNets).List() {
			n := nRaw.(map[string]interface{})
			if err := cli.NetworkDisconnect(ctx, n["name"].(string), containerID, false); err != nil {
				return diag.FromErr(fmt.Errorf("error disconnecting network %s: %w", n["name"].(string), err))
			}
		}

		// Connect new networks
		for _, nRaw := range newNets.Difference(oldNets).List() {
			n := nRaw.(map[string]interface{})
			epConfig := &network.EndpointSettings{}
			epConfig.IPAMConfig = &network.EndpointIPAMConfig{}
			if ipv4, ok := n["ipv4_address"].(string); ok && ipv4 != "" {
				epConfig.IPAMConfig.IPv4Address = ipv4
			}
			if ipv6, ok := n["ipv6_address"].(string); ok && ipv6 != "" {
				epConfig.IPAMConfig.IPv6Address = ipv6
			}
			if aliases, ok := n["aliases"]; ok {
				epConfig.Aliases = stringSetToSlice(aliases)
			}
			if err := cli.NetworkConnect(ctx, n["name"].(string), containerID, epConfig); err != nil {
				return diag.FromErr(fmt.Errorf("error connecting network %s: %w", n["name"].(string), err))
			}
		}
	}

	// Handle env changes
	if d.HasChange("env") {
		// Environment changes require a container restart for Podman
		// We stop, update config indirectly, and start
		// In practice env changes on a running container require recreation,
		// but we handle it gracefully
		return diag.Diagnostics{
			diag.Diagnostic{
				Severity: diag.Warning,
				Summary:  "Environment variable changes may require container recreation",
				Detail:   "Some changes to environment variables may not take effect until the container is recreated.",
			},
		}
	}

	// Handle labels change
	if d.HasChange("labels") {
		// Labels on running containers cannot be changed; this is informational
		// The state will be updated on read
	}

	return resourcePodmanContainerRead(ctx, d, meta)
}

func resourcePodmanContainerDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	config := getClient(meta)
	cli := config.Client
	containerID := d.Id()

	// If the container is set to rm (AutoRemove), check if it still exists
	if d.Get("rm").(bool) {
		_, err := cli.ContainerInspect(ctx, containerID)
		if err != nil {
			// Container already removed
			d.SetId("")
			return nil
		}
	}

	// Stop the container first
	if d.Get("must_run").(bool) || d.Get("start").(bool) {
		graceSeconds := 0
		if v, ok := d.GetOk("destroy_grace_seconds"); ok {
			graceSeconds = v.(int)
		}

		stopOpts := container.StopOptions{}
		if graceSeconds > 0 {
			stopOpts.Timeout = &graceSeconds
		}

		if err := cli.ContainerStop(ctx, containerID, stopOpts); err != nil {
			// Ignore "not running" or "not found" errors
			if !strings.Contains(err.Error(), "is not running") &&
				!strings.Contains(err.Error(), "No such container") &&
				!strings.Contains(err.Error(), "not found") {
				return diag.FromErr(fmt.Errorf("error stopping container %s: %w", containerID, err))
			}
		}

		// Wait for container to stop
		waitCh, errCh := cli.ContainerWait(ctx, containerID, container.WaitConditionNotRunning)
		select {
		case <-waitCh:
		case err := <-errCh:
			if err != nil {
				// Ignore if container is already gone
				if !strings.Contains(err.Error(), "No such container") && !strings.Contains(err.Error(), "not found") {
					return diag.FromErr(fmt.Errorf("error waiting for container %s to stop: %w", containerID, err))
				}
			}
		case <-time.After(2 * time.Minute):
			return diag.FromErr(fmt.Errorf("timeout waiting for container %s to stop", containerID))
		}
	}

	removeVolumes := d.Get("remove_volumes").(bool)
	removeOpts := container.RemoveOptions{
		RemoveVolumes: removeVolumes,
		Force:         true,
	}

	if err := cli.ContainerRemove(ctx, containerID, removeOpts); err != nil {
		if !strings.Contains(err.Error(), "No such container") && !strings.Contains(err.Error(), "not found") {
			return diag.FromErr(fmt.Errorf("error removing container %s: %w", containerID, err))
		}
	}

	d.SetId("")
	return nil
}

// uploadFileToContainer uploads a single file to the container via the Docker API.
func uploadFileToContainer(ctx context.Context, cli interface {
	CopyToContainer(ctx context.Context, containerID, dstPath string, content io.Reader, options types.CopyToContainerOptions) error
}, containerID string, upload map[string]interface{}) error {
	filePath := upload["file"].(string)
	var fileContent []byte

	if content, ok := upload["content"].(string); ok && content != "" {
		fileContent = []byte(content)
	} else if contentB64, ok := upload["content_base64"].(string); ok && contentB64 != "" {
		decoded, err := base64.StdEncoding.DecodeString(contentB64)
		if err != nil {
			return fmt.Errorf("error decoding base64 content for %s: %w", filePath, err)
		}
		fileContent = decoded
	} else if source, ok := upload["source"].(string); ok && source != "" {
		data, err := os.ReadFile(source)
		if err != nil {
			return fmt.Errorf("error reading source file %s: %w", source, err)
		}
		fileContent = data
	} else {
		return fmt.Errorf("one of content, content_base64, or source must be set for upload to %s", filePath)
	}

	// Determine file mode
	fileMode := os.FileMode(0644)
	if upload["executable"].(bool) {
		fileMode = os.FileMode(0755)
	}

	// Build tar archive with the file
	var buf bytes.Buffer
	tw := tar.NewWriter(&buf)

	// Get just the filename from the full path
	parts := strings.Split(filePath, "/")
	fileName := parts[len(parts)-1]

	header := &tar.Header{
		Name: fileName,
		Mode: int64(fileMode),
		Size: int64(len(fileContent)),
	}
	if err := tw.WriteHeader(header); err != nil {
		return fmt.Errorf("error writing tar header for %s: %w", filePath, err)
	}
	if _, err := tw.Write(fileContent); err != nil {
		return fmt.Errorf("error writing tar content for %s: %w", filePath, err)
	}
	if err := tw.Close(); err != nil {
		return fmt.Errorf("error closing tar writer for %s: %w", filePath, err)
	}

	// Determine the directory to copy to
	dir := "/"
	if idx := strings.LastIndex(filePath, "/"); idx > 0 {
		dir = filePath[:idx]
	}

	return cli.CopyToContainer(ctx, containerID, dir, &buf, types.CopyToContainerOptions{})
}
