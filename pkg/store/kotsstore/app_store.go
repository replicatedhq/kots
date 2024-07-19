package kotsstore

import (
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
	troubleshootanalyze "github.com/replicatedhq/troubleshoot/pkg/analyze"
	"github.com/rqlite/gorqlite"
	"github.com/segmentio/ksuid"
	"go.uber.org/zap"
)

func (s *KOTSStore) AddAppToAllDownstreams(appID string) error {
	db := persistence.MustGetDBSession()

	clusters, err := s.ListClusters()
	if err != nil {
		return errors.Wrap(err, "failed to list clusters")
	}
	for _, cluster := range clusters {
		query := `insert into app_downstream (app_id, cluster_id, downstream_name) values (?, ?, ?) ON CONFLICT DO NOTHING`
		wr, err := db.WriteOneParameterized(gorqlite.ParameterizedStatement{
			Query:     query,
			Arguments: []interface{}{appID, cluster.ClusterID, cluster.Name},
		})
		if err != nil {
			return fmt.Errorf("failed to create app downstream: %v: %v", err, wr.Err)
		}
	}

	return nil
}

func (s *KOTSStore) SetAppInstallState(appID string, state string) error {
	db := persistence.MustGetDBSession()

	query := `update app set install_state = ? where id = ?`
	wr, err := db.WriteOneParameterized(gorqlite.ParameterizedStatement{
		Query:     query,
		Arguments: []interface{}{state, appID},
	})
	if err != nil {
		return fmt.Errorf("failed to update app install state: %v: %v", err, wr.Err)
	}

	return nil
}

func (s *KOTSStore) ListInstalledApps() ([]*apptypes.App, error) {
	db := persistence.MustGetDBSession()
	query := `select id from app where install_state = 'installed'`
	rows, err := db.QueryOne(query)
	if err != nil {
		return nil, fmt.Errorf("failed to query: %v: %v", err, rows.Err)
	}

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

func (s *KOTSStore) ListFailedApps() ([]*apptypes.App, error) {
	db := persistence.MustGetDBSession()
	query := `select id from app where install_state != 'installed'`
	rows, err := db.QueryOne(query)
	if err != nil {
		return nil, fmt.Errorf("failed to query: %v: %v", err, rows.Err)
	}

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
	db := persistence.MustGetDBSession()
	query := `select slug from app where install_state = 'installed'`
	rows, err := db.QueryOne(query)
	if err != nil {
		return nil, fmt.Errorf("failed to query: %v: %v", err, rows.Err)
	}

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
	db := persistence.MustGetDBSession()
	query := `select id from app where slug = ?`
	rows, err := db.QueryOneParameterized(gorqlite.ParameterizedStatement{
		Query:     query,
		Arguments: []interface{}{slug},
	})
	if err != nil {
		return "", fmt.Errorf("failed to query: %v: %v", err, rows.Err)
	}
	if !rows.Next() {
		return "", ErrNotFound
	}

	id := ""
	if err := rows.Scan(&id); err != nil {
		return "", errors.Wrap(err, "failed to scan id")
	}

	return id, nil
}

func (s *KOTSStore) GetApp(id string) (*apptypes.App, error) {
	db := persistence.MustGetDBSession()
	query := `select id, name, license, upstream_uri, icon_uri, created_at, updated_at, slug, current_sequence, last_update_check_at, last_license_sync, is_airgap, snapshot_ttl_new, snapshot_schedule, restore_in_progress_name, restore_undeploy_status, update_checker_spec, semver_auto_deploy, install_state, channel_changed, channel_id from app where id = ?`
	rows, err := db.QueryOneParameterized(gorqlite.ParameterizedStatement{
		Query:     query,
		Arguments: []interface{}{id},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to query app: %v: %v", err, rows.Err)
	}
	if !rows.Next() {
		return nil, ErrNotFound
	}

	app := apptypes.App{}

	var licenseStr gorqlite.NullString
	var upstreamURI gorqlite.NullString
	var iconURI gorqlite.NullString
	var updatedAt gorqlite.NullTime
	var currentSequence gorqlite.NullInt64
	var lastUpdateCheckAt gorqlite.NullTime
	var lastLicenseSync gorqlite.NullTime
	var snapshotTTLNew gorqlite.NullString
	var snapshotSchedule gorqlite.NullString
	var restoreInProgressName gorqlite.NullString
	var restoreUndeployStatus gorqlite.NullString
	var updateCheckerSpec gorqlite.NullString
	var autoDeploy gorqlite.NullString
	var channelID gorqlite.NullString

	if err := rows.Scan(&app.ID, &app.Name, &licenseStr, &upstreamURI, &iconURI, &app.CreatedAt, &updatedAt, &app.Slug, &currentSequence, &lastUpdateCheckAt, &lastLicenseSync, &app.IsAirgap, &snapshotTTLNew, &snapshotSchedule, &restoreInProgressName, &restoreUndeployStatus, &updateCheckerSpec, &autoDeploy, &app.InstallState, &app.ChannelChanged, &channelID); err != nil {
		return nil, errors.Wrap(err, "failed to scan app")
	}

	app.License = licenseStr.String
	app.UpstreamURI = upstreamURI.String
	app.IconURI = iconURI.String
	app.SnapshotTTL = snapshotTTLNew.String
	app.SnapshotSchedule = snapshotSchedule.String
	app.RestoreInProgressName = restoreInProgressName.String
	app.RestoreUndeployStatus = apptypes.UndeployStatus(restoreUndeployStatus.String)
	app.UpdateCheckerSpec = updateCheckerSpec.String
	app.AutoDeploy = apptypes.AutoDeploy(autoDeploy.String)
	app.ChannelID = channelID.String

	if lastLicenseSync.Valid {
		app.LastLicenseSync = lastLicenseSync.Time.Format(time.RFC3339)
	}

	if lastUpdateCheckAt.Valid {
		app.LastUpdateCheckAt = &lastUpdateCheckAt.Time
	}

	if updatedAt.Valid {
		app.UpdatedAt = &updatedAt.Time
	}

	if currentSequence.Valid {
		app.CurrentSequence = currentSequence.Int64
	} else {
		app.CurrentSequence = -1
	}

	if app.CurrentSequence != -1 { // this means that there's at least 1 version available
		latestSequence, err := s.GetLatestAppSequence(app.ID, true)
		if err != nil {
			return nil, errors.Wrap(err, "failed to get latest app sequence")
		}

		query = `select preflight_spec, config_spec from app_version where app_id = ? AND sequence = ?`
		rows, err := db.QueryOneParameterized(gorqlite.ParameterizedStatement{
			Query:     query,
			Arguments: []interface{}{id, latestSequence},
		})
		if err != nil {
			return nil, fmt.Errorf("failed to query app_version: %v: %v", err, rows.Err)
		}
		if !rows.Next() {
			return nil, ErrNotFound
		}

		var preflightSpec gorqlite.NullString
		var configSpec gorqlite.NullString

		if err := rows.Scan(&preflightSpec, &configSpec); err != nil {
			return nil, errors.Wrap(err, "failed to scan app_version")
		}

		if preflightSpec.String != "" {
			preflight, err := kotsutil.LoadPreflightFromContents([]byte(preflightSpec.String))
			if err != nil {
				return nil, errors.Wrap(err, "failed to load preflights from spec")
			}

			// this spec has templates applied to it already
			numAnalyzers := 0
			for _, analyzer := range preflight.Spec.Analyzers {
				exclude := troubleshootanalyze.GetExcludeFlag(analyzer).BoolOrDefaultFalse()
				if !exclude {
					numAnalyzers += 1
				}
			}

			if numAnalyzers > 0 {
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

func (s *KOTSStore) CreateApp(name string, channelID string, upstreamURI string, licenseData string, isAirgapEnabled bool, skipImagePush bool, registryIsReadOnly bool) (*apptypes.App, error) {
	logger.Debug("creating app",
		zap.String("name", name),
		zap.String("upstreamURI", upstreamURI),
		zap.String("channelID", channelID),
	)

	db := persistence.MustGetDBSession()

	titleForSlug := strings.Replace(name, ".", "-", 0)
	slugProposal := slug.Make(titleForSlug)

	foundUniqueSlug := false
	i := 0
	for !foundUniqueSlug {
		if i > 0 {
			slugProposal = fmt.Sprintf("%s-%d", titleForSlug, i)
		}

		query := `select count(1) as count from app where slug = ?`
		rows, err := db.QueryOneParameterized(gorqlite.ParameterizedStatement{
			Query:     query,
			Arguments: []interface{}{slugProposal},
		})
		if err != nil {
			return nil, fmt.Errorf("failed to query app: %v: %v", err, rows.Err)
		}
		if !rows.Next() {
			return nil, ErrNotFound
		}

		exists := 0
		if err := rows.Scan(&exists); err != nil {
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

	query := `insert into app (id, name, icon_uri, created_at, slug, upstream_uri, license, is_all_users, install_state, registry_is_readonly, channel_id) values (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`
	wr, err := db.WriteOneParameterized(gorqlite.ParameterizedStatement{
		Query:     query,
		Arguments: []interface{}{id, name, "", time.Now().Unix(), slugProposal, upstreamURI, licenseData, true, installState, registryIsReadOnly, channelID},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to insert app: %v: %v", err, wr.Err)
	}

	return s.GetApp(id)
}

func (s *KOTSStore) ListDownstreamsForApp(appID string) ([]downstreamtypes.Downstream, error) {
	db := persistence.MustGetDBSession()
	query := `select c.id from app_downstream d inner join cluster c on d.cluster_id = c.id where app_id = ?`
	rows, err := db.QueryOneParameterized(gorqlite.ParameterizedStatement{
		Query:     query,
		Arguments: []interface{}{appID},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to query: %v: %v", err, rows.Err)
	}

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
	db := persistence.MustGetDBSession()
	query := `select ad.app_id from app_downstream ad inner join app a on ad.app_id = a.id where ad.cluster_id = ? and a.install_state = 'installed'`
	rows, err := db.QueryOneParameterized(gorqlite.ParameterizedStatement{
		Query:     query,
		Arguments: []interface{}{clusterID},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to query: %v: %v", err, rows.Err)
	}

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
	db := persistence.MustGetDBSession()
	query := `select c.id, c.slug, d.downstream_name, d.current_sequence from app_downstream d inner join cluster c on d.cluster_id = c.id where c.id = ?`
	rows, err := db.QueryOneParameterized(gorqlite.ParameterizedStatement{
		Query:     query,
		Arguments: []interface{}{clusterID},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to query: %v: %v", err, rows.Err)
	}
	if !rows.Next() {
		return nil, nil
	}

	downstream := downstreamtypes.Downstream{
		CurrentSequence: -1,
	}
	var sequence gorqlite.NullInt64
	if err := rows.Scan(&downstream.ClusterID, &downstream.ClusterSlug, &downstream.Name, &sequence); err != nil {
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

	db := persistence.MustGetDBSession()
	query := `update app set update_checker_spec = ? where id = ?`
	wr, err := db.WriteOneParameterized(gorqlite.ParameterizedStatement{
		Query:     query,
		Arguments: []interface{}{updateCheckerSpec, appID},
	})
	if err != nil {
		return fmt.Errorf("failed to write: %v: %v", err, wr.Err)
	}

	return nil
}

func (s *KOTSStore) SetAutoDeploy(appID string, autoDeploy apptypes.AutoDeploy) error {
	logger.Debug("setting auto deploy",
		zap.String("appID", appID))

	db := persistence.MustGetDBSession()
	query := `update app set semver_auto_deploy = ? where id = ?`
	wr, err := db.WriteOneParameterized(gorqlite.ParameterizedStatement{
		Query:     query,
		Arguments: []interface{}{autoDeploy, appID},
	})
	if err != nil {
		return fmt.Errorf("failed to write: %v: %v", err, wr.Err)
	}

	return nil
}

func (s *KOTSStore) SetSnapshotTTL(appID string, snapshotTTL string) error {
	logger.Debug("Setting snapshot TTL",
		zap.String("appID", appID))

	db := persistence.MustGetDBSession()
	query := `update app set snapshot_ttl_new = ? where id = ?`
	wr, err := db.WriteOneParameterized(gorqlite.ParameterizedStatement{
		Query:     query,
		Arguments: []interface{}{snapshotTTL, appID},
	})
	if err != nil {
		return fmt.Errorf("failed to write: %v: %v", err, wr.Err)
	}

	return nil
}

func (s *KOTSStore) SetSnapshotSchedule(appID string, snapshotSchedule string) error {
	logger.Debug("Setting snapshot Schedule",
		zap.String("appID", appID))

	db := persistence.MustGetDBSession()
	query := `update app set snapshot_schedule = ? where id = ?`
	wr, err := db.WriteOneParameterized(gorqlite.ParameterizedStatement{
		Query:     query,
		Arguments: []interface{}{snapshotSchedule, appID},
	})
	if err != nil {
		return fmt.Errorf("failed to write: %v: %v", err, wr.Err)
	}

	return nil
}

func (s *KOTSStore) RemoveApp(appID string) error {
	logger.Debug("Removing app",
		zap.String("appID", appID))

	db := persistence.MustGetDBSession()
	statements := []gorqlite.ParameterizedStatement{}

	// TODO: api_task_status needs app ID

	statements = append(statements, gorqlite.ParameterizedStatement{
		Query:     "delete from app_status where app_id = ?",
		Arguments: []interface{}{appID},
	})

	statements = append(statements, gorqlite.ParameterizedStatement{
		Query:     "delete from app_downstream_output where app_id = ?",
		Arguments: []interface{}{appID},
	})

	statements = append(statements, gorqlite.ParameterizedStatement{
		Query:     "delete from app_downstream_version where app_id = ?",
		Arguments: []interface{}{appID},
	})

	statements = append(statements, gorqlite.ParameterizedStatement{
		Query:     "delete from app_downstream where app_id = ?",
		Arguments: []interface{}{appID},
	})

	statements = append(statements, gorqlite.ParameterizedStatement{
		Query:     "delete from app_version where app_id = ?",
		Arguments: []interface{}{appID},
	})

	statements = append(statements, gorqlite.ParameterizedStatement{
		Query:     "delete from user_app where app_id = ?",
		Arguments: []interface{}{appID},
	})

	statements = append(statements, gorqlite.ParameterizedStatement{
		Query:     "delete from pending_supportbundle where app_id = ?",
		Arguments: []interface{}{appID},
	})

	statements = append(statements, gorqlite.ParameterizedStatement{
		Query:     "delete from app where id = ?",
		Arguments: []interface{}{appID},
	})

	if wrs, err := db.WriteParameterized(statements); err != nil {
		wrErrs := []error{}
		for _, wr := range wrs {
			wrErrs = append(wrErrs, wr.Err)
		}
		return fmt.Errorf("failed to write: %v: %v", err, wrErrs)
	}

	return nil
}

func (s *KOTSStore) SetAppChannelChanged(appID string, channelChanged bool) error {
	db := persistence.MustGetDBSession()

	query := `update app set channel_changed = ? where id = ?`
	wr, err := db.WriteOneParameterized(gorqlite.ParameterizedStatement{
		Query:     query,
		Arguments: []interface{}{channelChanged, appID},
	})
	if err != nil {
		return fmt.Errorf("failed to update app channel changed flag: %v: %v", err, wr.Err)
	}

	return nil
}

func (s *KOTSStore) GetAppChannelID(appID string) (string, error) {
	db := persistence.MustGetDBSession()
	query := `select channel_id from app where id = ?`
	rows, err := db.QueryOneParameterized(gorqlite.ParameterizedStatement{
		Query:     query,
		Arguments: []interface{}{appID},
	})
	if err != nil {
		return "", fmt.Errorf("failed to query: %v: %v", err, rows.Err)
	}
	if !rows.Next() {
		return "", ErrNotFound
	}

	var channelID gorqlite.NullString
	if err := rows.Scan(&channelID); err != nil {
		return "", errors.Wrap(err, "failed to scan channel id")
	}

	return channelID.String, nil
}

func (s *KOTSStore) SetAppChannelID(appID string, channelID string) error {
	db := persistence.MustGetDBSession()

	query := `update app set channel_id = ? where id = ?`
	wr, err := db.WriteOneParameterized(gorqlite.ParameterizedStatement{
		Query:     query,
		Arguments: []interface{}{channelID, appID},
	})
	if err != nil {
		return fmt.Errorf("failed to update app channel id: %v: %v", err, wr.Err)
	}

	return nil
}
