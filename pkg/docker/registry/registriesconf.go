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
// If the host file is missing, a minimal empty v2 file is used instead. If the
// v1 config cannot be converted, an error is returned so that blocked-registry
// and insecure-registry policy is not silently discarded.
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

		if _, statErr := os.Stat(hostPath); statErr != nil {
			if os.IsNotExist(statErr) {
				// File does not exist; fall back to an empty v2 file.
				return writeDefaultRegistriesConf()
			}
			return "", errors.Wrap(statErr, "failed to stat host registries.conf")
		}

		// Host file exists but is not valid v2. Determine whether it is a legacy
		// v1 config before attempting conversion, so an invalid v2 file is not
		// silently replaced with an empty v2 config.
		isV1, err := isV1RegistriesConf(hostPath)
		if err != nil {
			return "", errors.Wrap(err, "host registries.conf is not valid v2 and cannot be parsed as v1")
		}
		if !isV1 {
			return "", errors.Errorf("host registries.conf %q is not a valid v2 file and does not contain v1 markers; refusing to install an empty v2 config and discard registry policy", hostPath)
		}

		convertedPath, err := convertHostRegistriesConfToV2(hostPath)
		if err == nil {
			return convertedPath, nil
		}
		if errors.Is(err, errV1ConfigEmpty) {
			// Nothing to preserve; fall back to an empty v2 file.
			return writeDefaultRegistriesConf()
		}
		return "", errors.Wrap(err, "failed to convert host v1 registries.conf to v2; refusing to install an empty v2 config and discard registry policy")
	}

	return writeDefaultRegistriesConf()
}

// errV1ConfigEmpty indicates a legacy v1 registries.conf file was parsed
// successfully but contained no blocked, insecure, or search registry entries.
// In this case it is safe to fall back to an empty v2 configuration.
var errV1ConfigEmpty = errors.New("v1 registries.conf has no blocked, insecure, or search registry entries")

func isValidV2RegistriesConf(path string) bool {
	ctx := &types.SystemContext{SystemRegistriesConfPath: path}
	_, err := sysregistriesv2.TryUpdatingCache(ctx)
	return err == nil
}

func isV1RegistriesConf(path string) (bool, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return false, err
	}
	var raw map[string]interface{}
	if err := toml.Unmarshal(data, &raw); err != nil {
		return false, err
	}
	registries, ok := raw["registries"].(map[string]interface{})
	if !ok {
		return false, nil
	}
	_, hasSearch := registries["search"]
	_, hasInsecure := registries["insecure"]
	_, hasBlock := registries["block"]
	return hasSearch || hasInsecure || hasBlock, nil
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
		return "", errV1ConfigEmpty
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
