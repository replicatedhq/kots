package s3pg

import (
	"time"

	"github.com/pkg/errors"
	"github.com/replicatedhq/kots/kotsadm/pkg/logger"
	"github.com/replicatedhq/kots/kotsadm/pkg/persistence"
	sessiontypes "github.com/replicatedhq/kots/kotsadm/pkg/session/types"
	usertypes "github.com/replicatedhq/kots/kotsadm/pkg/user/types"
	"github.com/segmentio/ksuid"
	"go.uber.org/zap"
)

func (s S3PGStore) CreateSession(forUser *usertypes.User) (*sessiontypes.Session, error) {
	logger.Debug("creating session")

	randomID, err := ksuid.NewRandom()
	if err != nil {
		return nil, errors.Wrap(err, "failed to generate random session id")
	}

	id := randomID.String()

	db := persistence.MustGetPGSession()
	query := `insert into session (id, user_id, metadata, expire_at) values ($1, $2, $3, $4)`
	_, err = db.Exec(query, id, forUser.ID, "", time.Now().AddDate(0, 0, 14))
	if err != nil {
		return nil, errors.Wrap(err, "failed to create session")
	}

	return s.GetSession(id)
}

func (s S3PGStore) GetSession(id string) (*sessiontypes.Session, error) {
	logger.Debug("getting session from database",
		zap.String("id", id))

	db := persistence.MustGetPGSession()
	query := `select id, expire_at from session where id = $1`
	row := db.QueryRow(query, id)
	session := sessiontypes.Session{}

	var expiresAt time.Time
	if err := row.Scan(&session.ID, &expiresAt); err != nil {
		return nil, errors.Wrap(err, "failed to get session")
	}

	session.ExpiresAt = expiresAt

	return &session, nil
}

func (s S3PGStore) DeleteSession(id string) error {
	db := persistence.MustGetPGSession()
	query := `delete from session where id = $1`

	_, err := db.Exec(query, id)
	if err != nil {
		return errors.Wrap(err, "failed to exec")
	}

	return nil
}
