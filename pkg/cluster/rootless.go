package cluster

import (
	"context"
	"fmt"
	"io/ioutil"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/opencontainers/runc/libcontainer/cgroups"
	"github.com/pkg/errors"
	"github.com/rootless-containers/rootlesskit/pkg/child"
	"github.com/rootless-containers/rootlesskit/pkg/copyup/tmpfssymlink"
	"github.com/rootless-containers/rootlesskit/pkg/network/slirp4netns"
	"github.com/rootless-containers/rootlesskit/pkg/parent"
	portbuiltin "github.com/rootless-containers/rootlesskit/pkg/port/builtin"
	"golang.org/x/sys/unix"
)

var (
	pipeFD             = "_KOTS_ROOTLESS_FD"
	childEnv           = "_KOTS_ROOTLESS_SOCK"
	evacuateCgroup2Env = "_KOTS_ROOTLESS_EVACUATE_CGROUP2" // boolean
	Sock               = ""
)

func InitUserNamespace(ctx context.Context, dataDir string) error {
	defer func() {
		os.Unsetenv(pipeFD)
		os.Unsetenv(childEnv)
	}()

	stateDir := filepath.Join(dataDir, "rootless")

	// this code was inspired (and some of it copied) from https://github.com/k3s-io/k3s/tree/master/pkg/rootless
	if os.Getenv(pipeFD) != "" {
		childOpt, err := createChildOpt()
		if err != nil {
			return errors.Wrap(err, "create child opt")
		}

		if err := child.Child(*childOpt); err != nil {
			return errors.Wrap(err, "child")
		}
	}

	if err := validateSysctl(); err != nil {
		return errors.Wrap(err, "validate sysctl")
	}

	parentOpt, err := createParentOpt(dataDir, stateDir)
	if err != nil {
		return errors.Wrap(err, "create parent opt")
	}

	os.Setenv(childEnv, filepath.Join(parentOpt.StateDir, parent.StateFileAPISock))
	if parentOpt.EvacuateCgroup2 != "" {
		os.Setenv(evacuateCgroup2Env, "1")
	}

	if err := parent.Parent(*parentOpt); err != nil {
		return errors.Wrap(err, "parent")
	}
	os.Exit(0)

	return nil
}

func validateSysctl() error {
	expected := map[string]string{
		// kernel.unprivileged_userns_clone needs to be 1 to allow userns on some distros.
		"kernel.unprivileged_userns_clone": "1",

		// net.ipv4.ip_forward should not need to be 1 in the parent namespace.
		// However, the current k3s implementation has a bug that requires net.ipv4.ip_forward=1
		// https://github.com/rancher/k3s/issues/2420#issuecomment-715051120
		"net.ipv4.ip_forward": "1",
	}
	for key, expectedValue := range expected {
		if actualValue, err := readSysctl(key); err == nil {
			if expectedValue != actualValue {
				return errors.Errorf("expected sysctl value %q to be %q, got %q; try adding \"%s=%s\" to /etc/sysctl.conf and running `sudo sysctl --system`",
					key, expectedValue, actualValue, key, expectedValue)
			}
		}
	}
	return nil
}

func readSysctl(key string) (string, error) {
	p := "/proc/sys/" + strings.ReplaceAll(key, ".", "/")
	b, err := ioutil.ReadFile(p)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(b)), nil
}

func createParentOpt(dataDir string, stateDir string) (*parent.Opt, error) {
	if err := os.MkdirAll(stateDir, 0755); err != nil {
		return nil, errors.Wrapf(err, "failed to mkdir %s", stateDir)
	}

	stateDir, err := ioutil.TempDir("", "rootless")
	if err != nil {
		return nil, err
	}

	opt := &parent.Opt{
		StateDir:       stateDir,
		CreatePIDNS:    true,
		CreateCgroupNS: true,
		CreateUTSNS:    true,
		CreateIPCNS:    true,
	}

	selfCgroupMap, err := cgroups.ParseCgroupFile("/proc/self/cgroup")
	if err != nil {
		return nil, err
	}
	selfCgroup2 := selfCgroupMap[""]
	if selfCgroup2 == "" {
		fmt.Printf("enabling cgroup2 is highly recommended, see https://rootlesscontaine.rs/getting-started/common/cgroup2/")
	} else {
		selfCgroup2Dir := filepath.Join("/sys/fs/cgroup", selfCgroup2)
		fmt.Printf("%s\n", selfCgroup2Dir)
		if err := unix.Access(selfCgroup2Dir, unix.W_OK); err == nil {
			opt.EvacuateCgroup2 = "kots_evac"
		} else {
			// return nil, errors.Wrap(err, "unix access")
			fmt.Printf("cgroup2 is not enabled, see https://rootlesscontaine.rs/getting-started/common/cgroup2/")
		}
	}

	mtu := 65520
	ipnet, err := parseCIDR("10.41.0.0/16")
	if err != nil {
		return nil, err
	}
	disableHostLoopback := false
	binary := filepath.Join(BinRoot(dataDir), "slirp4netns")
	opt.NetworkDriver, err = slirp4netns.NewParentDriver(os.Stdout, binary, mtu, ipnet, "tap0", disableHostLoopback, "", false, false, false)
	if err != nil {
		return nil, err
	}

	// TODO remove this next line, right now the slirp4netns configuration isn't right and prevents outbound traffic!
	opt.NetworkDriver = nil

	opt.PortDriver, err = portbuiltin.NewParentDriver(os.Stdout, stateDir)
	if err != nil {
		return nil, err
	}

	opt.PipeFDEnvKey = pipeFD

	return opt, nil
}

func parseCIDR(s string) (*net.IPNet, error) {
	if s == "" {
		return nil, nil
	}
	ip, ipnet, err := net.ParseCIDR(s)
	if err != nil {
		return nil, err
	}
	if !ip.Equal(ipnet.IP) {
		return nil, errors.Errorf("cidr must be like 10.0.2.0/24, not like 10.0.2.100/24")
	}
	return ipnet, nil
}

func createChildOpt() (*child.Opt, error) {
	opt := &child.Opt{}
	opt.TargetCmd = os.Args
	opt.PipeFDEnvKey = pipeFD
	opt.NetworkDriver = slirp4netns.NewChildDriver()
	opt.PortDriver = portbuiltin.NewChildDriver(os.Stdout)
	opt.CopyUpDirs = []string{"/etc", "/var/run", "/run", "/var/lib"}
	opt.CopyUpDriver = tmpfssymlink.NewChildDriver()
	opt.MountProcfs = true
	opt.Reaper = true
	if v := os.Getenv(evacuateCgroup2Env); v != "" {
		var err error
		opt.EvacuateCgroup2, err = strconv.ParseBool(v)
		if err != nil {
			return nil, err
		}
	}
	return opt, nil
}
