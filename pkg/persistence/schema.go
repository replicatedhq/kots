package persistence

import (
	"os"
	"path/filepath"

	"github.com/pkg/errors"
	schemaherodb "github.com/schemahero/schemahero/pkg/database"
)

func UpdateDBSchema(driver string, uri string, schemaDir string) error {
	statements := []string{}

	schemaheroDB := schemaherodb.Database{
		Driver: driver,
		URI:    uri,
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
