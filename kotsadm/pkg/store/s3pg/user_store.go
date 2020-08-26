package s3pg

import (
	"database/sql"
	"time"

	"github.com/pkg/errors"
	"github.com/replicatedhq/kots/kotsadm/pkg/persistence"
	"github.com/replicatedhq/kots/kotsadm/pkg/rand"
)

func (s S3PGStore) CreateAdminConsolePassword(passwordBcrypt string) (string, error) {
	db := persistence.MustGetPGSession()
	tx, err := db.Begin()
	if err != nil {
		return "", errors.Wrap(err, "failed to begin transaction")
	}
	defer tx.Rollback()

	query := `select user_id from ship_user_local where email = $1`
	row := tx.QueryRow(query, "default-user@none.com")

	var userID string
	if err := row.Scan(&userID); err != nil {
		if err != sql.ErrNoRows {
			return "", errors.Wrap(err, "failed lookup existing user")
		}

		userID = rand.StringWithCharset(32, rand.LOWER_CASE)
		query := `insert into ship_user (id, created_at, last_login) values ($1, $2, $3)`

		_, err := tx.Exec(query, userID, time.Now(), time.Now())
		if err != nil {
			return "", errors.Wrap(err, "failed to create ship user")
		}
	}

	query = `insert into ship_user_local (user_id, password_bcrypt, first_name, last_name, email)
	values ($1, $2, $3, $4, $5) ON CONFLICT (email) do update set password_bcrypt = EXCLUDED.password_bcrypt`
	_, err = tx.Exec(query, userID, passwordBcrypt, "Default", "User", "default-user@none.com")
	if err != nil {
		return "", errors.Wrap(err, "failed to create ship user local")
	}

	if err := tx.Commit(); err != nil {
		return "", errors.Wrap(err, "failed to commit transaction")
	}

	return userID, nil
}
