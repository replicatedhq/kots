package app

import (
	"database/sql"
	"time"

	"github.com/pkg/errors"
	"github.com/replicatedhq/kotsadm/pkg/logger"
	"github.com/replicatedhq/kotsadm/pkg/persistence"
	"go.uber.org/zap"
)

type App struct {
	ID              string
	Slug            string
	Name            string
	IsAirgap        bool
	CurrentSequence int64

	// Additional fields will be added here as implementation is moved from node to go
	RestoreInProgressName string
}

func Get(id string) (*App, error) {
	logger.Debug("getting app from id",
		zap.String("id", id))

	db := persistence.MustGetPGSession()
	query := `select id, slug, name, current_sequence, is_airgap, restore_in_progress_name from app where id = $1`
	row := db.QueryRow(query, id)

	app := App{}

	var currentSequence sql.NullInt64
	var restoreInProgressName sql.NullString

	if err := row.Scan(&app.ID, &app.Slug, &app.Name, &currentSequence, &app.IsAirgap, &restoreInProgressName); err != nil {
		return nil, errors.Wrap(err, "failed to scan app")
	}

	if currentSequence.Valid {
		app.CurrentSequence = currentSequence.Int64
	} else {
		app.CurrentSequence = -1
	}

	app.RestoreInProgressName = restoreInProgressName.String

	return &app, nil
}

func GetFromSlug(slug string) (*App, error) {
	logger.Debug("getting app from slug",
		zap.String("slug", slug))

	db := persistence.MustGetPGSession()
	query := `select id from app where slug = $1`
	row := db.QueryRow(query, slug)

	id := ""

	if err := row.Scan(&id); err != nil {
		return nil, errors.Wrap(err, "failed to scan id")
	}

	return Get(id)
}

// LastUpdateAtTime sets the time that the client last checked for an update to now
func LastUpdateAtTime(appID string) error {
	db := persistence.MustGetPGSession()
	query := `update app set last_update_check_at = $1 where id = $2`
	_, err := db.Exec(query, time.Now(), appID)
	if err != nil {
		return errors.Wrap(err, "failed to update last_update_check_at")
	}

	return nil
}

func InitiateRestore(snapshotName string, appID string) error {
	db := persistence.MustGetPGSession()
	query := `update app set restore_in_progress_name = $1 where id = $2`
	_, err := db.Exec(query, snapshotName, appID)
	if err != nil {
		return errors.Wrap(err, "failed to update restore_in_progress_name")
	}

	return nil
}
