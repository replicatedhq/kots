package registry

import (
	"os"
	"path/filepath"
	"sync"

	"github.com/BurntSushi/toml"
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
// is in the legacy v1 format (which newer containers/image rejects), its
// blocked, insecure, and search registry settings are converted to a v2 file.
// If the host file is missing or the v1 config cannot be converted, a minimal
// empty v2 file is used instead.
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
		// Host has a v1 or otherwise invalid config. Try to preserve its
		// blocked/insecure/search registry settings by converting to v2.
		convertedPath, err := convertHostRegistriesConfToV2(hostPath)
		if err == nil {
			return convertedPath, nil
		}
		// If conversion fails or the v1 file has no settings, fall back to an
		// empty v2 file rather than letting the operation fail entirely.
	}

	return writeDefaultRegistriesConf()
}

func isValidV2RegistriesConf(path string) bool {
	ctx := &types.SystemContext{SystemRegistriesConfPath: path}
	_, err := sysregistriesv2.TryUpdatingCache(ctx)
	return err == nil
}

func convertHostRegistriesConfToV2(hostPath string) (string, error) {
	data, err := os.ReadFile(hostPath)
	if err != nil {
		return "", errors.Wrap(err, "failed to read host registries.conf")
	}

	var v1 sysregistriesv2.V1RegistriesConf
	if err := toml.Unmarshal(data, &v1); err != nil {
		return "", errors.Wrap(err, "failed to parse v1 registries.conf")
	}
	if !v1.Nonempty() {
		// Nothing to preserve; use the empty fallback.
		return "", errors.New("v1 registries.conf is empty")
	}

	v2, err := v1.ConvertToV2()
	if err != nil {
		return "", errors.Wrap(err, "failed to convert v1 registries.conf to v2")
	}

	dir, err := os.MkdirTemp("", "kots-registries-conf")
	if err != nil {
		return "", errors.Wrap(err, "failed to create registries.conf temp dir")
	}
	path := filepath.Join(dir, "registries.conf")

	f, err := os.Create(path)
	if err != nil {
		return "", errors.Wrap(err, "failed to create converted registries.conf")
	}
	defer f.Close()

	enc := toml.NewEncoder(f)
	if err := enc.Encode(v2); err != nil {
		return "", errors.Wrap(err, "failed to encode converted registries.conf")
	}

	// Validate that the generated v2 file can be loaded by the library.
	ctx := &types.SystemContext{SystemRegistriesConfPath: path}
	if _, err := sysregistriesv2.TryUpdatingCache(ctx); err != nil {
		return "", errors.Wrap(err, "generated v2 registries.conf is invalid")
	}

	return path, nil
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
