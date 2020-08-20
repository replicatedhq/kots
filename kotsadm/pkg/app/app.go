package app

import (
	"time"

	"github.com/pkg/errors"
	"github.com/replicatedhq/kots/kotsadm/pkg/app/types"
	"github.com/replicatedhq/kots/kotsadm/pkg/logger"
	"github.com/replicatedhq/kots/kotsadm/pkg/persistence"
	"github.com/replicatedhq/kots/kotsadm/pkg/store"
	"go.uber.org/zap"
)

func GetLicenseDataFromDatabase(id string) (string, error) {
	logger.Debug("getting app license from database",
		zap.String("id", id))

	db := persistence.MustGetPGSession()
	query := `select license from app where id = $1`
	row := db.QueryRow(query, id)

	license := ""

	if err := row.Scan(&license); err != nil {
		return "", errors.Wrap(err, "failed to scan license")
	}

	return license, nil
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

func ResetRestore(appID string) error {
	db := persistence.MustGetPGSession()
	query := `update app set restore_in_progress_name = NULL, restore_undeploy_status = '' where id = $1`
	_, err := db.Exec(query, appID)
	if err != nil {
		return errors.Wrap(err, "failed to exec")
	}

	return nil
}

func SetRestoreUndeployStatus(appID string, undeployStatus types.UndeployStatus) error {
	db := persistence.MustGetPGSession()
	query := `update app set restore_undeploy_status = $1 where id = $2`
	_, err := db.Exec(query, undeployStatus, appID)
	if err != nil {
		return errors.Wrap(err, "failed to exec")
	}

	return nil
}

func ListInstalledForDownstream(clusterID string) ([]*types.App, error) {
	db := persistence.MustGetPGSession()
	query := `select ad.app_id from app_downstream ad inner join app a on ad.app_id = a.id where ad.cluster_id = $1 and a.install_state = 'installed'`
	rows, err := db.Query(query, clusterID)
	if err != nil {
		return nil, errors.Wrap(err, "failed to query db")
	}
	defer rows.Close()

	apps := []*types.App{}
	for rows.Next() {
		var appID string
		if err := rows.Scan(&appID); err != nil {
			return nil, errors.Wrap(err, "failed to scan")
		}
		app, err := store.GetStore().GetApp(appID)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to get app %s", appID)
		}
		apps = append(apps, app)
	}

	return apps, nil
}
