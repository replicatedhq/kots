package identity

import (
	"github.com/replicatedhq/kots/kotsadm/pkg/persistence"
)

func createDexPostgresDatabase(database, user, password string) error {
	db := persistence.MustGetPGSession()

	query := `
	CREATE DATABASE $1;
	CREATE USER $2;
	ALTER USER $2 WITH PASSWORD '$3';
	GRANT ALL PRIVILEGES ON DATABASE $1 TO $2;`

	_, err := db.Exec(query, database, user, password)
	return err
}
