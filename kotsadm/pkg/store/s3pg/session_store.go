package s3pg

import (
	"encoding/json"
	"time"

	"github.com/pkg/errors"
	"github.com/replicatedhq/kots/kotsadm/pkg/logger"
	"github.com/replicatedhq/kots/kotsadm/pkg/persistence"
	sessiontypes "github.com/replicatedhq/kots/kotsadm/pkg/session/types"
	usertypes "github.com/replicatedhq/kots/kotsadm/pkg/user/types"
	"github.com/segmentio/ksuid"
)

type SessionMetadata struct {
	Roles []string
}

func (s S3PGStore) CreateSession(forUser *usertypes.User, expiresAt *time.Time, roles []string) (*sessiontypes.Session, error) {
	logger.Debug("creating session")

	randomID, err := ksuid.NewRandom()
	if err != nil {
		return nil, errors.Wrap(err, "failed to generate random session id")
	}

	id := randomID.String()

	metadata, err := json.Marshal(SessionMetadata{Roles: roles})
	if err != nil {
		return nil, errors.Wrap(err, "failed to marshal session metadata")
	}

	if expiresAt == nil {
		e := time.Now().AddDate(0, 0, 14)
		expiresAt = &e
	}

	db := persistence.MustGetPGSession()
	query := `insert into session (id, user_id, metadata, expire_at) values ($1, $2, $3, $4)`
	_, err = db.Exec(query, id, forUser.ID, string(metadata), *expiresAt)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create session")
	}

	return s.GetSession(id)
}

func (s S3PGStore) GetSession(id string) (*sessiontypes.Session, error) {
	// too noisy
	// logger.Debug("getting session from database",
	// 	zap.String("id", id))

	db := persistence.MustGetPGSession()
	query := `select id, metadata, expire_at from session where id = $1`
	row := db.QueryRow(query, id)
	session := sessiontypes.Session{}

	var expiresAt time.Time
	var metadataStr string
	if err := row.Scan(&session.ID, &metadataStr, &expiresAt); err != nil {
		return nil, errors.Wrap(err, "failed to get session")
	}

	if metadataStr != "" {
		metadata := SessionMetadata{}
		if err := json.Unmarshal([]byte(metadataStr), &metadata); err != nil {
			return nil, errors.Wrap(err, "failed to unmarshal session metadata")
		}
		session.HasRBAC = true
		session.Roles = metadata.Roles
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
