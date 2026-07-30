package registry

import (
	"os"
	"path/filepath"
	"sync"

	"github.com/pkg/errors"
	"go.podman.io/image/v5/types"
)

var (
	registriesConfPathOnce sync.Once
	registriesConfPath     string
	registriesConfPathErr  error
)

// SetSystemRegistriesConfPath sets SystemRegistriesConfPath on the provided
// SystemContext to a valid v2-format registries.conf file. This avoids failures
// when the host's /etc/containers/registries.conf is in the legacy v1 format,
// which newer versions of containers/image reject.
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
		registriesConfPath, registriesConfPathErr = writeDefaultRegistriesConf()
	})
	return registriesConfPath, registriesConfPathErr
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
