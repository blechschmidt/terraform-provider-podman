package provider

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/docker/docker/client"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

// libpodAPIVersion is the libpod API version embedded in the URL path.
// Podman accepts requests for any version equal to or below the running
// podman version; v4.0.0 has been stable since 2022.
const libpodAPIVersion = "v4.0.0"

// libpodSpecGen builds a libpod SpecGenerator JSON object from the resource
// schema. It is only used when "rootfs" is set, because the Docker compat
// API used by the rest of the provider does not understand rootfs.
//
// Only the subset of options that map cleanly onto SpecGenerator is
// translated. Image-only options (e.g. ulimits, cgroup ns mode, runtime,
// healthcheck) are accepted by the schema but quietly skipped here — see
// the rootfs section of the docs for the supported list.
func libpodSpecGen(d *schema.ResourceData) map[string]interface{} {
	spec := map[string]interface{}{
		"name":   d.Get("name").(string),
		"rootfs": d.Get("rootfs").(string),
	}

	if d.Get("rootfs_overlay").(bool) {
		spec["rootfs_overlay"] = true
	}
	if v, ok := d.GetOk("rootfs_mapping"); ok {
		spec["rootfs_mapping"] = v.(string)
	}

	if v, ok := d.GetOk("command"); ok {
		spec["command"] = stringListToSlice(v)
	}
	if v, ok := d.GetOk("entrypoint"); ok {
		spec["entrypoint"] = stringListToSlice(v)
	}
	if v, ok := d.GetOk("env"); ok {
		envMap := map[string]string{}
		for _, e := range stringSetToSlice(v) {
			parts := strings.SplitN(e, "=", 2)
			if len(parts) == 2 {
				envMap[parts[0]] = parts[1]
			} else {
				envMap[parts[0]] = ""
			}
		}
		spec["env"] = envMap
	}
	if v, ok := d.GetOk("labels"); ok {
		spec["labels"] = labelsToMap(v)
	}
	if v, ok := d.GetOk("hostname"); ok {
		spec["hostname"] = v.(string)
	}
	if v, ok := d.GetOk("user"); ok {
		spec["user"] = v.(string)
	}
	if v, ok := d.GetOk("working_dir"); ok {
		spec["work_dir"] = v.(string)
	}
	if d.Get("tty").(bool) {
		spec["terminal"] = true
	}
	if d.Get("stdin_open").(bool) {
		spec["stdin"] = true
	}
	if d.Get("privileged").(bool) {
		spec["privileged"] = true
	}
	if d.Get("read_only").(bool) {
		spec["read_only_filesystem"] = true
	}
	if d.Get("rm").(bool) {
		spec["remove"] = true
	}

	if restart := d.Get("restart").(string); restart != "" && restart != "no" {
		spec["restart_policy"] = restart
		if v, ok := d.GetOk("max_retry_count"); ok {
			spec["restart_tries"] = uint(v.(int))
		}
	}

	if v, ok := d.GetOk("stop_signal"); ok {
		spec["stop_signal"] = v.(string)
	}
	if v, ok := d.GetOk("stop_timeout"); ok {
		spec["stop_timeout"] = uint(v.(int))
	}

	if v, ok := d.GetOk("capabilities"); ok {
		capsList := v.([]interface{})
		if len(capsList) > 0 {
			capsMap := capsList[0].(map[string]interface{})
			if add, ok := capsMap["add"]; ok {
				if s := stringSetToSlice(add); len(s) > 0 {
					spec["cap_add"] = s
				}
			}
			if drop, ok := capsMap["drop"]; ok {
				if s := stringSetToSlice(drop); len(s) > 0 {
					spec["cap_drop"] = s
				}
			}
		}
	}

	if v, ok := d.GetOk("dns"); ok {
		spec["dns_server"] = stringSetToSlice(v)
	}
	if v, ok := d.GetOk("dns_opts"); ok {
		spec["dns_option"] = stringSetToSlice(v)
	}
	if v, ok := d.GetOk("dns_search"); ok {
		spec["dns_search"] = stringSetToSlice(v)
	}

	if v, ok := d.GetOk("host"); ok {
		hosts := []string{}
		for _, hRaw := range v.(*schema.Set).List() {
			h := hRaw.(map[string]interface{})
			hosts = append(hosts, h["host"].(string)+":"+h["ip"].(string))
		}
		if len(hosts) > 0 {
			spec["hostadd"] = hosts
		}
	}

	if v, ok := d.GetOk("group_add"); ok {
		spec["groups"] = stringSetToSlice(v)
	}
	if v, ok := d.GetOk("sysctls"); ok {
		spec["sysctl"] = mapStringInterfaceToStringString(v.(map[string]interface{}))
	}
	if v, ok := d.GetOk("security_opts"); ok {
		spec["selinux_opts"] = stringSetToSlice(v)
	}

	if v, ok := d.GetOk("network_mode"); ok {
		spec["netns"] = map[string]interface{}{
			"nsmode": v.(string),
		}
	}

	if v, ok := d.GetOk("ports"); ok {
		ports := []map[string]interface{}{}
		for _, p := range v.([]interface{}) {
			pMap := p.(map[string]interface{})
			pm := map[string]interface{}{
				"container_port": uint16(pMap["internal"].(int)),
				"protocol":       pMap["protocol"].(string),
			}
			if ext := pMap["external"].(int); ext > 0 {
				pm["host_port"] = uint16(ext)
			}
			if ip := pMap["ip"].(string); ip != "" {
				pm["host_ip"] = ip
			}
			ports = append(ports, pm)
		}
		spec["portmappings"] = ports
	}

	mounts := []map[string]interface{}{}
	if v, ok := d.GetOk("mounts"); ok {
		for _, mRaw := range v.(*schema.Set).List() {
			m := mRaw.(map[string]interface{})
			options := []string{}
			if m["read_only"].(bool) {
				options = append(options, "ro")
			}
			if bOpts, ok := m["bind_options"].([]interface{}); ok && len(bOpts) > 0 {
				bMap := bOpts[0].(map[string]interface{})
				if prop, _ := bMap["propagation"].(string); prop != "" {
					options = append(options, prop)
				}
			}
			mounts = append(mounts, map[string]interface{}{
				"destination": m["target"].(string),
				"source":      m["source"].(string),
				"type":        m["type"].(string),
				"options":     options,
			})
		}
	}

	var namedVolumes []map[string]interface{}
	if v, ok := d.GetOk("volumes"); ok {
		for _, volRaw := range v.(*schema.Set).List() {
			vol := volRaw.(map[string]interface{})
			if vol["from_container"].(string) != "" {
				continue
			}
			hostPath := vol["host_path"].(string)
			containerPath := vol["container_path"].(string)
			volumeName := vol["volume_name"].(string)
			options := []string{}
			if vol["read_only"].(bool) {
				options = append(options, "ro")
			}
			if hostPath != "" && containerPath != "" {
				mounts = append(mounts, map[string]interface{}{
					"destination": containerPath,
					"source":      hostPath,
					"type":        "bind",
					"options":     options,
				})
			} else if volumeName != "" && containerPath != "" {
				namedVolumes = append(namedVolumes, map[string]interface{}{
					"Name":    volumeName,
					"Dest":    containerPath,
					"Options": options,
				})
			}
		}
	}
	if len(mounts) > 0 {
		spec["mounts"] = mounts
	}
	if len(namedVolumes) > 0 {
		spec["volumes"] = namedVolumes
	}

	if v, ok := d.GetOk("init"); ok {
		spec["init"] = v.(bool)
	}
	if v, ok := d.GetOk("shm_size"); ok {
		spec["shm_size"] = int64(v.(int))
	}
	if v, ok := d.GetOk("log_driver"); ok {
		logCfg := map[string]interface{}{
			"driver": v.(string),
		}
		if opts, ok := d.GetOk("log_opts"); ok {
			logCfg["options"] = mapStringInterfaceToStringString(opts.(map[string]interface{}))
		}
		spec["log_configuration"] = logCfg
	}

	return spec
}

// libpodCreateContainer POSTs a SpecGenerator to the libpod
// /libpod/containers/create endpoint and returns the new container ID. It
// reuses the docker client's HTTP transport so that unix sockets, TLS,
// and ssh tunnels keep working.
func libpodCreateContainer(ctx context.Context, cli *client.Client, spec map[string]interface{}) (string, error) {
	body, err := json.Marshal(spec)
	if err != nil {
		return "", fmt.Errorf("error marshaling libpod spec: %w", err)
	}

	reqURL, dummyHost, err := buildLibpodURL(cli.DaemonHost(), "/"+libpodAPIVersion+"/libpod/containers/create")
	if err != nil {
		return "", err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, reqURL, bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	if dummyHost {
		req.Host = "d"
	}

	resp, err := cli.HTTPClient().Do(req)
	if err != nil {
		return "", fmt.Errorf("libpod create call failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 400 {
		return "", fmt.Errorf("libpod create returned %d: %s", resp.StatusCode, strings.TrimSpace(string(respBody)))
	}

	var result struct {
		ID string `json:"Id"`
	}
	if err := json.Unmarshal(respBody, &result); err != nil {
		return "", fmt.Errorf("error parsing libpod response: %w (body: %s)", err, string(respBody))
	}
	if result.ID == "" {
		return "", fmt.Errorf("libpod create succeeded but returned no container ID (body: %s)", string(respBody))
	}
	return result.ID, nil
}

// buildLibpodURL constructs the URL for a libpod request based on the
// configured daemon host. The second return is true when the caller should
// override the Host header with a placeholder (unix/npipe/ssh sockets).
func buildLibpodURL(host, path string) (string, bool, error) {
	u, err := url.Parse(host)
	if err != nil {
		return "", false, fmt.Errorf("invalid host %q: %w", host, err)
	}

	switch u.Scheme {
	case "unix", "npipe", "ssh":
		return "http://d" + path, true, nil
	case "tcp":
		return "http://" + u.Host + path, false, nil
	case "http", "https":
		return u.Scheme + "://" + u.Host + path, false, nil
	default:
		return "", false, fmt.Errorf("unsupported host scheme %q", u.Scheme)
	}
}
