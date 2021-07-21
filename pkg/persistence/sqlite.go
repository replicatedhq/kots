package persistence

import (
	"database/sql"
	"fmt"

	_ "github.com/mattn/go-sqlite3"
	"github.com/pkg/errors"
	schemasv1alpha4 "github.com/schemahero/schemahero/pkg/apis/schemas/v1alpha4"
	schemaheroscheme "github.com/schemahero/schemahero/pkg/client/schemaheroclientset/scheme"
	"github.com/schemahero/schemahero/pkg/database"
	"k8s.io/client-go/kubernetes/scheme"
)

var sqliteDB *sql.DB

func mustGetSQLiteSession() *sql.DB {
	if sqliteDB != nil {
		return sqliteDB
	}

	db, err := sql.Open("sqlite3", SQLiteURI)
	if err != nil {
		fmt.Printf("error connecting to sqlite: %v\n", err)
		panic(err)
	}

	// apply the schema
	if err := applySQLiteSchema(db); err != nil {
		fmt.Printf("error applying schema: %v\n", err)
		panic(err)
	}

	sqliteDB = db
	return db
}

func applySQLiteSchema(db *sql.DB) error {
	schemaheroscheme.AddToScheme(scheme.Scheme)
	decode := scheme.Codecs.UniversalDeserializer().Decode

	schemahero := database.Database{
		Driver: "sqlite",
		URI:    SQLiteURI,
	}

	for _, table := range tables {
		// we need to use the k8s unmarshaler because these structs only have json tags on them
		decoded, _, err := decode([]byte(table), nil, nil)
		if err != nil {
			return errors.Wrap(err, "decode table")
		}

		t := decoded.(*schemasv1alpha4.Table)
		statements, err := schemahero.PlanSyncTableSpec(&t.Spec)
		if err != nil {
			return errors.Wrap(err, "plan table")
		}

		for _, statement := range statements {
			if _, err := db.Exec(statement); err != nil {
				return errors.Wrap(err, "execute plan")
			}
		}
	}

	return nil
}
