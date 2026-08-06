package persistence

import (
	"os"
	"path/filepath"

	"github.com/pkg/errors"
	schemaherodb "github.com/schemahero/schemahero/pkg/database"
	"github.com/schemahero/schemahero/pkg/database/plugin"
)

func UpdateDBSchema(driver string, uri string, schemaDir string) error {
	statements := []string{}

	// As of schemahero v0.23+ the database drivers (postgres, rqlite, ...) are no
	// longer in-tree: they are hashicorp/go-plugin binaries that schemahero launches
	// as subprocesses. Initialize the plugin system so schemahero discovers the
	// schemahero-<driver> binaries shipped in the image at /var/lib/schemahero/plugins
	// (see the apko image definitions). Discovery is local-only; the binaries are
	// installed at build time, so no network fetch happens at runtime (airgap-safe).
	plugin.InitializePluginSystem()

	schemaheroDB := schemaherodb.Database{
		Driver: driver,
		URI:    uri,
	}

	if pluginManager := plugin.GetGlobalPluginManager(); pluginManager != nil {
		schemaheroDB.SetPluginManager(pluginManager)
	}

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
