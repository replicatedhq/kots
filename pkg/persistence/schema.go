package persistence

import (
	"os"
	"path/filepath"
	"slices"

	"github.com/pkg/errors"
	schemaherodb "github.com/schemahero/schemahero/pkg/database"
	"github.com/schemahero/schemahero/pkg/database/plugin"
)

// defaultSchemaheroPluginDir is schemahero's default plugin discovery directory,
// and where our images install the driver plugin binaries at build time (see the
// apko image definitions).
const defaultSchemaheroPluginDir = "/var/lib/schemahero/plugins"

func UpdateDBSchema(driver string, uri string, schemaDir string) error {
	statements := []string{}

	// As of schemahero v0.25 the database drivers (postgres, rqlite, ...) are no
	// longer compiled into the schemahero library: each is a hashicorp/go-plugin
	// binary that schemahero launches as a subprocess and discovers on disk.
	// InitializePluginSystem discovers the schemahero-<driver> binaries shipped in
	// the image at /var/lib/schemahero/plugins. Discovery is local-only; the
	// binaries are installed at build time, so no network fetch happens at runtime
	// (airgap-safe).
	pluginManager := plugin.InitializePluginSystem()
	defer pluginManager.Cleanup()

	// Fail fast if the driver's plugin wasn't discovered. Otherwise schemahero
	// falls through to a network (ORAS) download that, in an air-gapped install,
	// only surfaces as a confusing ~30s timeout rather than a clear "missing
	// plugin" error.
	if !schemaheroPluginDiscovered(pluginManager, driver) {
		return errors.Errorf("schemahero %q driver plugin not found: expected the plugin binary at %s (installed into the image at build time; set SCHEMAHERO_PLUGIN_PATH to override its location). It is required to apply the schema and must be present for air-gapped installs.",
			driver, filepath.Join(defaultSchemaheroPluginDir, "schemahero-"+driver))
	}

	schemaheroDB := schemaherodb.Database{
		Driver: driver,
		URI:    uri,
	}
	schemaheroDB.SetPluginManager(pluginManager)

	err := filepath.Walk(schemaDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() {
			return nil
		}

		if ext := filepath.Ext(path); ext != ".yaml" && ext != ".yml" {
			return nil
		}

		stmnts, err := schemaheroDB.PlanSyncFromFile(path, "table")
		if err != nil {
			return err
		}
		statements = append(statements, stmnts...)

		return nil
	})
	if err != nil {
		return errors.Wrap(err, "failed to walk")
	}

	if err := schemaheroDB.ApplySync(statements); err != nil {
		return errors.Wrap(err, "failed to apply sync")
	}

	return nil
}

// schemaheroPluginDiscovered reports whether InitializePluginSystem discovered a
// local plugin supporting the given driver. It mirrors how schemahero resolves a
// driver before falling back to a network download, so it honors every discovery
// location schemahero checks (including $SCHEMAHERO_PLUGIN_PATH).
func schemaheroPluginDiscovered(pluginManager *plugin.PluginManager, driver string) bool {
	for _, info := range pluginManager.ListPlugins() {
		if slices.Contains(info.Engines, driver) {
			return true
		}
	}
	return false
}
