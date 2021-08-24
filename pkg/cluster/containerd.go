package cluster

import (
	"bytes"
	"context"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"strconv"

	"github.com/pkg/errors"
)

const containerdConfig = `disabled_plugins = []
imports = []
oom_score = 0
plugin_dir = ""
required_plugins = []
root = "%s"
state = "%s"
version = 2

[cgroup]
  path = ""

[debug]
  address = ""
  format = ""
  gid = 0
  level = ""
  uid = 0

[grpc]
  address = "%s"
  gid = %d
  max_recv_message_size = 16777216
  max_send_message_size = 16777216
  tcp_address = ""
  tcp_tls_cert = ""
  tcp_tls_key = ""
  uid = %d

[metrics]
  address = ""
  grpc_histogram = false

[plugins]

  [plugins."io.containerd.gc.v1.scheduler"]
    deletion_threshold = 0
    mutation_threshold = 100
    pause_threshold = 0.02
    schedule_delay = "0s"
    startup_delay = "100ms"

  [plugins."io.containerd.grpc.v1.cri"]
    disable_apparmor = false
    disable_cgroup = false
    disable_hugetlb_controller = true
    disable_proc_mount = false
    disable_tcp_service = true
    enable_selinux = false
    enable_tls_streaming = false
    ignore_image_defined_volumes = false
    max_concurrent_downloads = 3
    max_container_log_line_size = 16384
    netns_mounts_under_state_dir = false
    restrict_oom_score_adj = false
    sandbox_image = "k8s.gcr.io/pause:3.5"
    selinux_category_range = 1024
    stats_collect_period = 10
    stream_idle_timeout = "4h0m0s"
    stream_server_address = "127.0.0.1"
    stream_server_port = "0"
    systemd_cgroup = false
    tolerate_missing_hugetlb_controller = true
    unset_seccomp_profile = ""

    [plugins."io.containerd.grpc.v1.cri".cni]
      bin_dir = "/opt/cni/bin"
      conf_dir = "/etc/cni/net.d"
      conf_template = ""
      max_conf_num = 1

    [plugins."io.containerd.grpc.v1.cri".containerd]
      default_runtime_name = "runc"
      disable_snapshot_annotations = true
      discard_unpacked_layers = false
      no_pivot = false
      snapshotter = "overlayfs"

      [plugins."io.containerd.grpc.v1.cri".containerd.default_runtime]
        base_runtime_spec = ""
        container_annotations = []
        pod_annotations = []
        privileged_without_host_devices = false
        runtime_engine = ""
        runtime_root = ""
        runtime_type = ""

        [plugins."io.containerd.grpc.v1.cri".containerd.default_runtime.options]

      [plugins."io.containerd.grpc.v1.cri".containerd.runtimes]

        [plugins."io.containerd.grpc.v1.cri".containerd.runtimes.runc]
          base_runtime_spec = ""
          container_annotations = []
          pod_annotations = []
          privileged_without_host_devices = false
          runtime_engine = ""
          runtime_root = ""
          runtime_type = "io.containerd.runc.v2"

          [plugins."io.containerd.grpc.v1.cri".containerd.runtimes.runc.options]
            BinaryName = ""
            CriuImagePath = ""
            CriuPath = ""
            CriuWorkPath = ""
            IoGid = 0
            IoUid = 0
            NoNewKeyring = false
            NoPivotRoot = false
            Root = ""
            ShimCgroup = ""
            SystemdCgroup = false

      [plugins."io.containerd.grpc.v1.cri".containerd.untrusted_workload_runtime]
        base_runtime_spec = ""
        container_annotations = []
        pod_annotations = []
        privileged_without_host_devices = false
        runtime_engine = ""
        runtime_root = ""
        runtime_type = ""

        [plugins."io.containerd.grpc.v1.cri".containerd.untrusted_workload_runtime.options]

    [plugins."io.containerd.grpc.v1.cri".image_decryption]
      key_model = "node"

    [plugins."io.containerd.grpc.v1.cri".registry]
      config_path = ""

      [plugins."io.containerd.grpc.v1.cri".registry.auths]

      [plugins."io.containerd.grpc.v1.cri".registry.configs]

      [plugins."io.containerd.grpc.v1.cri".registry.headers]

      [plugins."io.containerd.grpc.v1.cri".registry.mirrors]

    [plugins."io.containerd.grpc.v1.cri".x509_key_pair_streaming]
      tls_cert_file = ""
      tls_key_file = ""

  [plugins."io.containerd.internal.v1.opt"]
    path = "/opt/containerd"

  [plugins."io.containerd.internal.v1.restart"]
    interval = "10s"

  [plugins."io.containerd.metadata.v1.bolt"]
    content_sharing_policy = "shared"

  [plugins."io.containerd.monitor.v1.cgroups"]
    no_prometheus = false

  [plugins."io.containerd.runtime.v1.linux"]
    no_shim = false
    runtime = "runc"
    runtime_root = ""
    shim = "containerd-shim"
    shim_debug = false

  [plugins."io.containerd.runtime.v2.task"]
    platforms = ["linux/amd64"]

  [plugins."io.containerd.service.v1.diff-service"]
    default = ["walking"]

  [plugins."io.containerd.snapshotter.v1.aufs"]
    root_path = ""

  [plugins."io.containerd.snapshotter.v1.btrfs"]
    root_path = ""

  [plugins."io.containerd.snapshotter.v1.devmapper"]
    async_remove = false
    base_image_size = ""
    pool_name = ""
    root_path = ""

  [plugins."io.containerd.snapshotter.v1.native"]
    root_path = ""

  [plugins."io.containerd.snapshotter.v1.overlayfs"]
    root_path = ""

  [plugins."io.containerd.snapshotter.v1.zfs"]
    root_path = ""

[proxy_plugins]

[stream_processors]

  [stream_processors."io.containerd.ocicrypt.decoder.v1.tar"]
    accepts = ["application/vnd.oci.image.layer.v1.tar+encrypted"]
    args = ["--decryption-keys-path", "/etc/containerd/ocicrypt/keys"]
    env = ["OCICRYPT_KEYPROVIDER_CONFIG=/etc/containerd/ocicrypt/ocicrypt_keyprovider.conf"]
    path = "ctd-decoder"
    returns = "application/vnd.oci.image.layer.v1.tar"

  [stream_processors."io.containerd.ocicrypt.decoder.v1.tar.gzip"]
    accepts = ["application/vnd.oci.image.layer.v1.tar+gzip+encrypted"]
    args = ["--decryption-keys-path", "/etc/containerd/ocicrypt/keys"]
    env = ["OCICRYPT_KEYPROVIDER_CONFIG=/etc/containerd/ocicrypt/ocicrypt_keyprovider.conf"]
    path = "ctd-decoder"
    returns = "application/vnd.oci.image.layer.v1.tar+gzip"

[timeouts]
  "io.containerd.timeout.shim.cleanup" = "5s"
  "io.containerd.timeout.shim.load" = "5s"
  "io.containerd.timeout.shim.shutdown" = "3s"
  "io.containerd.timeout.task.state" = "2s"

[ttrpc]
  address = ""
  gid = 0
  uid = 0
`

// startContainerd will start a rootless containerd in a subdirectory of dataDir
// This should start containerd with a sock file in dataDir/containerd/containerd.sock
func startContainerd(ctx context.Context, dataDir string) error {
	// extract containerd into a new directory
	installDir := filepath.Join(dataDir, "containerd")
	if _, err := os.Stat(installDir); err == nil {
		if err := os.RemoveAll(installDir); err != nil {
			return errors.Wrap(err, "remove previous containerd")
		}
	}

	// TODO make this not always download, it should only download when needed
	packageURI := `https://github.com/containerd/containerd/releases/download/v1.5.1/containerd-1.5.1-linux-amd64.tar.gz`
	resp, err := http.Get(packageURI)
	if err != nil {
		return errors.Wrap(err, "download containerd")
	}
	defer resp.Body.Close()

	if err := os.MkdirAll(installDir, 0755); err != nil {
		return errors.Wrap(err, "mkdir")
	}

	if err := extractArchiveStreamToDir(resp.Body, installDir); err != nil {
		return errors.Wrap(err, "extract")
	}

	// if err := generateDefaultConfig(dataDir, installDir); err != nil {
	// 	return errors.Wrap(err, "generate default config")
	// }

	if err := writeContainerdConfig(installDir); err != nil {
		return errors.Wrap(err, "write containerd config")
	}

	if err := spwanContainerd(installDir); err != nil {
		return errors.Wrap(err, "spawn containerd")
	}

	return nil
}

func spwanContainerd(installDir string) error {
	go func() {
		args := []string{
			fmt.Sprintf("--config=%s", filepath.Join(installDir, "config.toml")),
		}

		cmd := exec.Command(filepath.Join(installDir, "bin", "containerd"), args...)
		cmd.Env = os.Environ()

		// TODO stream the output of stdout and stderr to files
		var stdout bytes.Buffer
		var stderr bytes.Buffer

		cmd.Stdout = &stdout
		cmd.Stderr = &stderr

		err := cmd.Run()
		if err != nil {
			fmt.Printf("%s\n", stderr.String())
			panic(err)
		}

		fmt.Printf("%s\n", stdout.String())
	}()

	return nil
}

// writeContainerdConfig will write execute the config template
// and store it in a well known location (installDir/config.toml)
func writeContainerdConfig(installDir string) error {
	currentUser, err := user.Current()
	if err != nil {
		return errors.Wrap(err, "get current user")
	}

	gid, err := strconv.Atoi(currentUser.Gid)
	if err != nil {
		return errors.Wrap(err, "convert gid to int")
	}
	uid, err := strconv.Atoi(currentUser.Uid)
	if err != nil {
		return errors.Wrap(err, "convert uid to int")
	}

	b := fmt.Sprintf(containerdConfig,
		installDir,
		filepath.Join(installDir, "state"),
		filepath.Join(installDir, "containerd.sock"),
		gid,
		uid,
	)

	if err := ioutil.WriteFile(filepath.Join(installDir, "config.toml"), []byte(b), 0644); err != nil {
		return errors.Wrap(err, "write config")
	}

	return nil
}
