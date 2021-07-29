package kotsstore

import (
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/gosimple/slug"
	"github.com/pkg/errors"
	downstreamtypes "github.com/replicatedhq/kots/pkg/api/downstream/types"
	apptypes "github.com/replicatedhq/kots/pkg/app/types"
	"github.com/replicatedhq/kots/pkg/gitops"
	"github.com/replicatedhq/kots/pkg/kotsutil"
	"github.com/replicatedhq/kots/pkg/logger"
	"github.com/replicatedhq/kots/pkg/persistence"
	"github.com/segmentio/ksuid"
	"go.uber.org/zap"
)

func (s *KOTSStore) AddAppToAllDownstreams(appID string) error {
	db := persistence.MustGetPGSession()

	clusters, err := s.ListClusters()
	if err != nil {
		return errors.Wrap(err, "failed to list clusters")
	}
	for _, cluster := range clusters {
		query := `insert into app_downstream (app_id, cluster_id, downstream_name) values ($1, $2, $3) ON CONFLICT DO NOTHING`
		_, err = db.Exec(query, appID, cluster.ClusterID, cluster.Name)
		if err != nil {
			return errors.Wrap(err, "failed to create app downstream")
		}
	}

	return nil
}

func (s *KOTSStore) SetAppInstallState(appID string, state string) error {
	db := persistence.MustGetPGSession()

	query := `update app set install_state = $2 where id = $1`
	_, err := db.Exec(query, appID, state)
	if err != nil {
		return errors.Wrap(err, "failed to update app install state")
	}

	return nil
}

func (s *KOTSStore) ListInstalledApps() ([]*apptypes.App, error) {
	db := persistence.MustGetPGSession()
	query := `select id from app where install_state = 'installed'`
	rows, err := db.Query(query)
	if err != nil {
		return nil, errors.Wrap(err, "failed to query db")
	}
	defer rows.Close()

	apps := []*apptypes.App{}
	for rows.Next() {
		var appID string
		if err := rows.Scan(&appID); err != nil {
			return nil, errors.Wrap(err, "failed to scan")
		}
		app, err := s.GetApp(appID)
		if err != nil {
			return nil, errors.Wrap(err, "failed to get app")
		}
		apps = append(apps, app)
	}

	return apps, nil
}

func (s *KOTSStore) ListInstalledAppSlugs() ([]string, error) {
	db := persistence.MustGetPGSession()
	query := `select slug from app where install_state = 'installed'`
	rows, err := db.Query(query)
	if err != nil {
		return nil, errors.Wrap(err, "failed to query db")
	}
	defer rows.Close()

	appSlugs := []string{}
	for rows.Next() {
		var appSlug string
		if err := rows.Scan(&appSlug); err != nil {
			return nil, errors.Wrap(err, "failed to scan")
		}
		appSlugs = append(appSlugs, appSlug)
	}
	return appSlugs, nil
}

func (s *KOTSStore) GetAppIDFromSlug(slug string) (string, error) {
	db := persistence.MustGetPGSession()
	query := `select id from app where slug = $1`
	row := db.QueryRow(query, slug)

	id := ""

	if err := row.Scan(&id); err != nil {
		return "", errors.Wrap(err, "failed to scan id")
	}

	return id, nil
}

func (s *KOTSStore) GetApp(id string) (*apptypes.App, error) {
	// too noisy
	// logger.Debug("getting app from id",
	// 	zap.String("id", id))

	db := persistence.MustGetPGSession()
	query := `select id, name, license, upstream_uri, icon_uri, created_at, updated_at, slug, current_sequence, last_update_check_at, is_airgap, snapshot_ttl_new, snapshot_schedule, restore_in_progress_name, restore_undeploy_status, update_checker_spec, install_state from app where id = $1`
	row := db.QueryRow(query, id)

	app := apptypes.App{}

	var licenseStr sql.NullString
	var upstreamURI sql.NullString
	var iconURI sql.NullString
	var updatedAt sql.NullTime
	var currentSequence sql.NullInt64
	var lastUpdateCheckAt sql.NullString
	var snapshotTTLNew sql.NullString
	var snapshotSchedule sql.NullString
	var restoreInProgressName sql.NullString
	var restoreUndeployStatus sql.NullString
	var updateCheckerSpec sql.NullString

	if err := row.Scan(&app.ID, &app.Name, &licenseStr, &upstreamURI, &iconURI, &app.CreatedAt, &updatedAt, &app.Slug, &currentSequence, &lastUpdateCheckAt, &app.IsAirgap, &snapshotTTLNew, &snapshotSchedule, &restoreInProgressName, &restoreUndeployStatus, &updateCheckerSpec, &app.InstallState); err != nil {
		return nil, errors.Wrap(err, "failed to scan app")
	}

	app.License = licenseStr.String
	app.UpstreamURI = upstreamURI.String
	app.IconURI = iconURI.String
	app.LastUpdateCheckAt = lastUpdateCheckAt.String
	app.SnapshotTTL = snapshotTTLNew.String
	app.SnapshotSchedule = snapshotSchedule.String
	app.RestoreInProgressName = restoreInProgressName.String
	app.RestoreUndeployStatus = apptypes.UndeployStatus(restoreUndeployStatus.String)
	app.UpdateCheckerSpec = updateCheckerSpec.String

	if updatedAt.Valid {
		app.UpdatedAt = &updatedAt.Time
	}

	if currentSequence.Valid {
		app.CurrentSequence = currentSequence.Int64
	} else {
		app.CurrentSequence = -1
	}

	if app.CurrentSequence != -1 {
		query = `select preflight_spec, config_spec from app_version where app_id = $1 AND sequence = $2`
		row = db.QueryRow(query, id, app.CurrentSequence)

		var preflightSpec sql.NullString
		var configSpec sql.NullString

		if err := row.Scan(&preflightSpec, &configSpec); err != nil {
			return nil, errors.Wrap(err, "failed to scan app_version")
		}

		if preflightSpec.String != "" {
			preflight, err := kotsutil.LoadPreflightFromContents([]byte(preflightSpec.String))
			if err != nil {
				return nil, errors.Wrap(err, "failed to load preflights from spec")
			}
			if len(preflight.Spec.Analyzers) > 0 {
				app.HasPreflight = true
			}
		}
		if configSpec.String != "" {
			config, err := kotsutil.LoadConfigFromBytes([]byte(configSpec.String))
			if err != nil {
				return nil, errors.Wrap(err, "failed to load config from spec")
			}
			if len(config.Spec.Groups) > 0 {
				app.IsConfigurable = true
			}
		}
	}

	isGitOps, err := s.IsGitOpsEnabledForApp(id)
	if err != nil {
		return nil, errors.Wrap(err, "failed to check if gitops is enabled")
	}
	app.IsGitOps = isGitOps

	return &app, nil
}

func (s *KOTSStore) GetAppFromSlug(slug string) (*apptypes.App, error) {
	id, err := s.GetAppIDFromSlug(slug)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get id from slug")
	}

	return s.GetApp(id)
}

func (s *KOTSStore) CreateApp(name string, upstreamURI string, licenseData string, isAirgapEnabled bool, skipImagePush bool, registryIsReadOnly bool) (*apptypes.App, error) {
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
			if skipImagePush {
				installState = "installed"
			} else {
				installState = "airgap_upload_pending"
			}
		} else {
			installState = "online_upload_pending"
		}
	}

	id := ksuid.New().String()

	query := `insert into app (id, name, icon_uri, created_at, slug, upstream_uri, license, is_all_users, install_state, registry_is_readonly)
values ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)`
	_, err = tx.Exec(query, id, name, "", time.Now(), slugProposal, upstreamURI, licenseData, true, installState, registryIsReadOnly)
	if err != nil {
		return nil, errors.Wrap(err, "failed to insert app")
	}

	if err := tx.Commit(); err != nil {
		return nil, errors.Wrap(err, "failed to commit transaction")
	}

	return s.GetApp(id)
}

func (s *KOTSStore) ListDownstreamsForApp(appID string) ([]downstreamtypes.Downstream, error) {
	db := persistence.MustGetPGSession()
	query := `select c.id from app_downstream d inner join cluster c on d.cluster_id = c.id where app_id = $1`
	rows, err := db.Query(query, appID)
	if err != nil {
		return nil, errors.Wrap(err, "failed to query")
	}
	defer rows.Close()

	downstreams := []downstreamtypes.Downstream{}
	for rows.Next() {
		var clusterID string
		if err := rows.Scan(&clusterID); err != nil {
			return nil, errors.Wrap(err, "failed to scan")
		}
		downstream, err := s.GetDownstream(clusterID)
		if err != nil {
			return nil, errors.Wrap(err, "failed to get downstream")
		}
		if downstream != nil {
			downstreams = append(downstreams, *downstream)
		}
	}

	return downstreams, nil
}

func (s *KOTSStore) ListAppsForDownstream(clusterID string) ([]*apptypes.App, error) {
	db := persistence.MustGetPGSession()
	query := `select ad.app_id from app_downstream ad inner join app a on ad.app_id = a.id where ad.cluster_id = $1 and a.install_state = 'installed'`
	rows, err := db.Query(query, clusterID)
	if err != nil {
		return nil, errors.Wrap(err, "failed to query db")
	}
	defer rows.Close()

	apps := []*apptypes.App{}
	for rows.Next() {
		var appID string
		if err := rows.Scan(&appID); err != nil {
			return nil, errors.Wrap(err, "failed to scan")
		}
		app, err := s.GetApp(appID)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to get app %s", appID)
		}
		apps = append(apps, app)
	}

	return apps, nil
}

func (s *KOTSStore) GetDownstream(clusterID string) (*downstreamtypes.Downstream, error) {
	db := persistence.MustGetPGSession()
	query := `select c.id, c.slug, d.downstream_name, d.current_sequence from app_downstream d inner join cluster c on d.cluster_id = c.id where c.id = $1`
	row := db.QueryRow(query, clusterID)

	downstream := downstreamtypes.Downstream{
		CurrentSequence: -1,
	}
	var sequence sql.NullInt64
	if err := row.Scan(&downstream.ClusterID, &downstream.ClusterSlug, &downstream.Name, &sequence); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, errors.Wrap(err, "failed to scan downstream")
	}
	if sequence.Valid {
		downstream.CurrentSequence = sequence.Int64
	}

	return &downstream, nil
}

func (s *KOTSStore) IsGitOpsEnabledForApp(appID string) (bool, error) {
	downstreams, err := s.ListDownstreamsForApp(appID)
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

func (s *KOTSStore) SetUpdateCheckerSpec(appID string, updateCheckerSpec string) error {
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

func (s *KOTSStore) SetSnapshotTTL(appID string, snapshotTTL string) error {
	logger.Debug("Setting snapshot TTL",
		zap.String("appID", appID))
	db := persistence.MustGetPGSession()
	query := `update app set snapshot_ttl_new = $1 where id = $2`
	_, err := db.Exec(query, snapshotTTL, appID)
	if err != nil {
		return errors.Wrap(err, "failed to exec db query")
	}

	return nil
}

func (s *KOTSStore) SetSnapshotSchedule(appID string, snapshotSchedule string) error {
	logger.Debug("Setting snapshot Schedule",
		zap.String("appID", appID))
	db := persistence.MustGetPGSession()
	query := `update app set snapshot_schedule = $1 where id = $2`
	_, err := db.Exec(query, snapshotSchedule, appID)
	if err != nil {
		return errors.Wrap(err, "failed to exec db query")
	}

	return nil
}

func (s *KOTSStore) RemoveApp(appID string) error {
	logger.Debug("Removing app",
		zap.String("appID", appID))

	db := persistence.MustGetPGSession()
	tx, err := db.Begin()
	if err != nil {
		return errors.Wrap(err, "failed to begin transaction")
	}
	defer tx.Rollback()

	var query string

	// TODO: api_task_status needs app ID

	query = "delete from app_status where app_id = $1"
	_, err = tx.Exec(query, appID)
	if err != nil {
		return errors.Wrap(err, "failed to delete from app_status")
	}

	query = "delete from app_downstream_output where app_id = $1"
	_, err = tx.Exec(query, appID)
	if err != nil {
		return errors.Wrap(err, "failed to delete from app_downstream_output")
	}

	query = "delete from app_downstream_version where app_id = $1"
	_, err = tx.Exec(query, appID)
	if err != nil {
		return errors.Wrap(err, "failed to delete from app_downstream_version")
	}

	query = "delete from app_downstream where app_id = $1"
	_, err = tx.Exec(query, appID)
	if err != nil {
		return errors.Wrap(err, "failed to delete from app_downstream")
	}

	query = "delete from app_version where app_id = $1"
	_, err = tx.Exec(query, appID)
	if err != nil {
		return errors.Wrap(err, "failed to delete from app_version")
	}

	query = "delete from user_app where app_id = $1"
	_, err = tx.Exec(query, appID)
	if err != nil {
		return errors.Wrap(err, "failed to delete from user_app")
	}

	query = "delete from pending_supportbundle where app_id = $1"
	_, err = tx.Exec(query, appID)
	if err != nil {
		return errors.Wrap(err, "failed to delete from pending_supportbundle")
	}

	query = "delete from app where id = $1"
	_, err = tx.Exec(query, appID)
	if err != nil {
		return errors.Wrap(err, "failed to delete from app")
	}

	if err := tx.Commit(); err != nil {
		return errors.Wrap(err, "failed to commit transaction")
	}

	return nil
}
