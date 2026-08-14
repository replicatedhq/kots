package persistence

import (
	"os"
	"path/filepath"

	"github.com/pkg/errors"
	schemasv1alpha4 "github.com/schemahero/schemahero/pkg/apis/schemas/v1alpha4"
	"github.com/schemahero/schemahero/pkg/database/interfaces"
	postgreslib "github.com/schemahero/schemahero/plugins/postgres/lib"
	rqlitelib "github.com/schemahero/schemahero/plugins/rqlite/lib"
	"gopkg.in/yaml.v2"
)

// As of schemahero v0.25 the DB drivers live in separate go-plugin modules and
// schemahero's own Database/PlanSyncFromFile wrapper can only reach them by
// launching plugin binaries as subprocesses. We instead import the driver's lib
// package and drive its connection directly, so migrations run in-process with
// no plugin binary to ship or discover.
func UpdateDBSchema(driver string, uri string, schemaDir string) error {
	conn, err := connectSchemahero(driver, uri)
	if err != nil {
		return errors.Wrapf(err, "failed to connect to %s", driver)
	}
	if conn == nil {
		return errors.Errorf("no database connection returned for driver %q", driver)
	}
	defer conn.Close()

	statements := []string{}

	err = filepath.Walk(schemaDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() {
			return nil
		}

		if ext := filepath.Ext(path); ext != ".yaml" && ext != ".yml" {
			return nil
		}

		specContents, err := os.ReadFile(path)
		if err != nil {
			return errors.Wrap(err, "failed to read spec")
		}

		table := schemasv1alpha4.Table{}
		if err := yaml.Unmarshal(specContents, &table); err != nil {
			return errors.Wrapf(err, "failed to unmarshal %s", path)
		}

		if table.Spec.Name == "" {
			return errors.Errorf("table spec %s has no name", path)
		}

		schema, err := driverTableSchema(driver, table.Spec.Schema)
		if err != nil {
			return errors.Wrapf(err, "failed to resolve %s schema for %s", driver, path)
		}

		stmnts, err := conn.PlanTableSchema(table.Spec.Name, schema, nil)
		if err != nil {
			return errors.Wrapf(err, "failed to plan table %s", table.Spec.Name)
		}
		statements = append(statements, stmnts...)

		return nil
	})
	if err != nil {
		return errors.Wrap(err, "failed to walk")
	}

	if err := conn.DeployStatements(statements); err != nil {
		return errors.Wrap(err, "failed to apply sync")
	}

	return nil
}

func connectSchemahero(driver string, uri string) (interfaces.SchemaHeroDatabaseConnection, error) {
	switch driver {
	case "postgres":
		return postgreslib.Connect(uri)
	case "rqlite":
		return rqlitelib.Connect(uri)
	default:
		return nil, errors.Errorf("unsupported schemahero driver %q", driver)
	}
}

func driverTableSchema(driver string, schema *schemasv1alpha4.TableSchema) (any, error) {
	if schema == nil {
		return nil, errors.New("table spec has no schema")
	}

	switch driver {
	case "postgres":
		if schema.Postgres == nil {
			return nil, errors.New("table spec has no postgres schema")
		}
		return schema.Postgres, nil
	case "rqlite":
		if schema.RQLite == nil {
			return nil, errors.New("table spec has no rqlite schema")
		}
		return schema.RQLite, nil
	default:
		return nil, errors.Errorf("unsupported schemahero driver %q", driver)
	}
}
