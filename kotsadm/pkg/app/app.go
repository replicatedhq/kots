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
	ID                    string `json:"id"`
	Slug                  string `json:"slug"`
	Name                  string `json:"name"`
	License               string `json:"license"`
	IsAirgap              bool   `json:"isAirgap"`
	CurrentSequence       int64  `json:"currentSequence"`
	UpstreamURI           string `json:"upstreamUri"`
	IconURI               string `json:"iconUri"`
	UpdatedAt             string `json:"createdAt"`
	CreatedAt             string `json:"updatedAt"`
	LastUpdateCheckAt     string `json:"lastUpdateCheckAt"`
	BundleCommand         string `json:"bundleCommand"`
	HasPreflight          bool   `json:"hasPreflight"`
	IsConfigurable        bool   `json:"isConfigurable"`
	SnapshotTTL           string `json:"snapshotTtl"`
	SnapshotSchedule      string `json:"snapshotSchedule"`
	RestoreInProgressName string `json:"restoreInProgressName"`
	RestoreUndeployStatus string `json:"restoreUndeloyStatus"`
	UpdateCheckerSpec     string `json:"updateCheckerSpec"`
	IsGitOps              bool   `json:"isGitOps"`
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
	query := `select id, name, license, upstream_uri, icon_uri, created_at, updated_at, slug, current_sequence, last_update_check_at, is_airgap, snapshot_ttl_new, snapshot_schedule, restore_in_progress_name, restore_undeploy_status, update_checker_spec from app where id = $1`
	row := db.QueryRow(query, id)

	app := App{}

	var licenseStr sql.NullString
	var upstreamURI sql.NullString
	var iconURI sql.NullString
	var updatedAt sql.NullString
	var currentSequence sql.NullInt64
	var lastUpdateCheckAt sql.NullString
	var snapshotTTLNew sql.NullString
	var snapshotSchedule sql.NullString
	var restoreInProgressName sql.NullString
	var restoreUndeployStatus sql.NullString
	var updateCheckerSpec sql.NullString

	if err := row.Scan(&app.ID, &app.Name, &licenseStr, &upstreamURI, &iconURI, &app.CreatedAt, &updatedAt, &app.Slug, &currentSequence, &lastUpdateCheckAt, &app.IsAirgap, &snapshotTTLNew, &snapshotSchedule, &restoreInProgressName, &restoreUndeployStatus, &updateCheckerSpec); err != nil {
		return nil, errors.Wrap(err, "failed to scan app")
	}

	app.License = licenseStr.String
	app.UpstreamURI = upstreamURI.String
	app.IconURI = iconURI.String
	app.UpdatedAt = updatedAt.String
	app.LastUpdateCheckAt = lastUpdateCheckAt.String
	app.SnapshotTTL = snapshotTTLNew.String
	app.SnapshotSchedule = snapshotSchedule.String
	app.RestoreInProgressName = restoreInProgressName.String
	app.RestoreUndeployStatus = restoreUndeployStatus.String
	app.UpdateCheckerSpec = updateCheckerSpec.String

	if currentSequence.Valid {
		app.CurrentSequence = currentSequence.Int64
	} else {
		app.CurrentSequence = -1
	}

	query = `select preflight_spec, config_spec from app_version where app_id = $1 AND sequence = $2`
	row = db.QueryRow(query, id, app.CurrentSequence)

	var preflightSpec sql.NullString
	var configSpec sql.NullString

	if err := row.Scan(&preflightSpec, &configSpec); err != nil {
		return nil, errors.Wrap(err, "failed to scan app_version")
	}

	if preflightSpec.Valid && preflightSpec.String != "" {
		app.HasPreflight = true
	}
	if configSpec.Valid && configSpec.String != "" {
		app.IsConfigurable = true
	}

	bundleCommand := fmt.Sprintf(`
	curl https://krew.sh/support-bundle | bash
      kubectl support-bundle API_ADDRESS/api/v1/troubleshoot/%s
	`, app.Slug)
	app.BundleCommand = bundleCommand

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
