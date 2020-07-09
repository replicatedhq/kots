package app

import (
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/gosimple/slug"
	"github.com/pkg/errors"
	"github.com/replicatedhq/kots/kotsadm/pkg/downstream"
	"github.com/replicatedhq/kots/kotsadm/pkg/gitops"
	"github.com/replicatedhq/kots/kotsadm/pkg/logger"
	"github.com/replicatedhq/kots/kotsadm/pkg/persistence"
	"github.com/segmentio/ksuid"
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
	UpdateCheckerSpec     string
	IsGitOps              bool
}

type RegistryInfo struct {
	Hostname    string
	Username    string
	Password    string
	PasswordEnc string
	Namespace   string
}

func Get(id string) (*App, error) {
	logger.Debug("getting app from id",
		zap.String("id", id))

	db := persistence.MustGetPGSession()
	query := `select id, slug, name, current_sequence, is_airgap, restore_in_progress_name, update_checker_spec from app where id = $1`
	row := db.QueryRow(query, id)

	app := App{}

	var currentSequence sql.NullInt64
	var restoreInProgressName sql.NullString
	var updateCheckerSpec sql.NullString

	if err := row.Scan(&app.ID, &app.Slug, &app.Name, &currentSequence, &app.IsAirgap, &restoreInProgressName, &updateCheckerSpec); err != nil {
		return nil, errors.Wrap(err, "failed to scan app")
	}

	if currentSequence.Valid {
		app.CurrentSequence = currentSequence.Int64
	} else {
		app.CurrentSequence = -1
	}

	app.RestoreInProgressName = restoreInProgressName.String
	app.UpdateCheckerSpec = updateCheckerSpec.String

	isGitOps, err := IsGitOpsEnabled(id)
	if err != nil {
		return nil, errors.Wrap(err, "failed to check if gitops is enabled")
	}
	app.IsGitOps = isGitOps

	return &app, nil
}

func ListInstalled() ([]*App, error) {
	logger.Debug("getting all users apps")

	db := persistence.MustGetPGSession()
	query := `select id from app where install_state = 'installed'`
	rows, err := db.Query(query)
	if err != nil {
		return nil, errors.Wrap(err, "failed to query db")
	}

	apps := []*App{}
	for rows.Next() {
		var appID string
		if err := rows.Scan(&appID); err != nil {
			return nil, errors.Wrap(err, "failed to scan")
		}
		app, err := Get(appID)
		if err != nil {
			return nil, errors.Wrap(err, "failed to get app")
		}
		apps = append(apps, app)
	}

	return apps, nil
}

func IsGitOpsEnabled(appID string) (bool, error) {
	downstreams, err := downstream.ListDownstreamsForApp(appID)
	if err != nil {
		return false, errors.Wrap(err, "failed to list downstreams")
	}

	for _, d := range downstreams {
		downstreamGitOps, err := gitops.GetDownstreamGitOps(appID, d.ClusterID)
		if err != nil {
			return false, errors.Wrap(err, "failed to get downstream gitops")
		}
		if downstreamGitOps != nil {
			return true, nil
		}
	}

	return false, nil
}

func SetUpdateCheckerSpec(appID string, updateCheckerSpec string) error {
	logger.Debug("setting update checker spec",
		zap.String("appID", appID))

	db := persistence.MustGetPGSession()
	query := `update app set update_checker_spec = $1 where id = $2`
	_, err := db.Exec(query, updateCheckerSpec, appID)
	if err != nil {
		return errors.Wrap(err, "failed to exec db query")
	}

	return nil
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

func Create(name string, upstreamURI string, licenseData string, isAirgapEnabled bool) (*App, error) {
	logger.Debug("creating app",
		zap.String("name", name),
		zap.String("upstreamURI", upstreamURI))

	db := persistence.MustGetPGSession()
	tx, err := db.Begin()
	if err != nil {
		return nil, errors.Wrap(err, "failed to begin transaction")
	}
	defer tx.Rollback()

	titleForSlug := strings.Replace(name, ".", "-", 0)
	slugProposal := slug.Make(titleForSlug)

	foundUniqueSlug := false
	i := 0
	for !foundUniqueSlug {
		if i > 0 {
			slugProposal = fmt.Sprintf("%s-%d", titleForSlug, i)
		}

		query := `select count(1) as count from app where slug = $1`
		row := tx.QueryRow(query, slugProposal)
		exists := 0
		if err := row.Scan(&exists); err != nil {
			return nil, errors.Wrap(err, "failed to scan existing slug")
		}

		if exists == 0 {
			foundUniqueSlug = true
		} else {
			i++
		}
	}

	installState := ""
	if strings.HasPrefix(upstreamURI, "replicated://") == false {
		installState = "installed"
	} else {
		if isAirgapEnabled {
			installState = "airgap_upload_pending"
		} else {
			installState = "online_upload_pending"
		}
	}

	id := ksuid.New().String()

	query := `insert into app (id, name, icon_uri, created_at, slug, upstream_uri, license, is_all_users, install_state)
values ($1, $2, $3, $4, $5, $6, $7, $8, $9)`
	_, err = tx.Exec(query, id, name, "", time.Now(), slugProposal, upstreamURI, licenseData, true, installState)
	if err != nil {
		return nil, errors.Wrap(err, "failed to insert app")
	}

	if err := tx.Commit(); err != nil {
		return nil, errors.Wrap(err, "failerd to commit transaction")
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
