package registry

import (
	"os"
	"path/filepath"
	"sync"

	"github.com/pkg/errors"
	"go.podman.io/image/v5/pkg/sysregistriesv2"
	"go.podman.io/image/v5/types"
)

var (
	registriesConfPathOnce sync.Once
	registriesConfPath     string
	registriesConfPathErr  error
)

// SetSystemRegistriesConfPath sets SystemRegistriesConfPath on the provided
// SystemContext to a valid v2-format registries.conf file. If the host has a
// valid v2 registries.conf, that path is used so that configured mirrors,
// blocked registries, and insecure registries are preserved. If the host file
// is missing or is in the legacy v1 format (which newer containers/image rejects),
// a minimal empty v2 file is used instead.
func SetSystemRegistriesConfPath(sys *types.SystemContext) error {
	path, err := defaultRegistriesConfPath()
	if err != nil {
		return errors.Wrap(err, "failed to get default registries.conf path")
	}
	sys.SystemRegistriesConfPath = path
	return nil
}

func defaultRegistriesConfPath() (string, error) {
	registriesConfPathOnce.Do(func() {
		registriesConfPath, registriesConfPathErr = findDefaultRegistriesConfPath()
	})
	return registriesConfPath, registriesConfPathErr
}

// resetRegistriesConfPathOnce is used by tests to clear the cached path.
func resetRegistriesConfPathOnce() {
	registriesConfPathOnce = sync.Once{}
	registriesConfPath = ""
	registriesConfPathErr = nil
}

func findDefaultRegistriesConfPath() (string, error) {
	// Prefer the host's registries.conf when it is present and valid v2.
	hostPath := sysregistriesv2.ConfigPath(&types.SystemContext{})
	if hostPath != "" {
		if isValidV2RegistriesConf(hostPath) {
			return hostPath, nil
		}
	}

	return writeDefaultRegistriesConf()
}

func isValidV2RegistriesConf(path string) bool {
	ctx := &types.SystemContext{SystemRegistriesConfPath: path}
	_, err := sysregistriesv2.TryUpdatingCache(ctx)
	return err == nil
}

func writeDefaultRegistriesConf() (string, error) {
	dir, err := os.MkdirTemp("", "kots-registries-conf")
	if err != nil {
		return "", errors.Wrap(err, "failed to create registries.conf temp dir")
	}

	path := filepath.Join(dir, "registries.conf")
	// Empty file is valid v2 format and avoids loading a host v1 registries.conf.
	if err := os.WriteFile(path, []byte{}, 0644); err != nil {
		return "", errors.Wrap(err, "failed to write default registries.conf")
	}

	return path, nil
}
