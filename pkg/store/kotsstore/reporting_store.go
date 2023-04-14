package kotsstore

import (
	"fmt"
	"time"

	"github.com/pkg/errors"
	reportingtypes "github.com/replicatedhq/kots/pkg/api/reporting/types"
	"github.com/replicatedhq/kots/pkg/logger"
	"github.com/replicatedhq/kots/pkg/persistence"
	"github.com/rqlite/gorqlite"
)

func (s *KOTSStore) SavePreflightReport(licenseID string, preflightStatus *reportingtypes.PreflightStatus) error {
	db := persistence.MustGetDBSession()

	createdAt := time.Now().UTC()

	query := `
	INSERT INTO preflight_report (
		created_at,
		license_id,
		instance_id,
		cluster_id,
		sequence,
		skip_preflights,
		install_status,
		is_cli,
		preflight_status,
		app_status,
		kots_version)
	VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	ON CONFLICT(created_at) DO UPDATE SET
		license_id = EXCLUDED.license_id,
		instance_id = EXCLUDED.instance_id,
		cluster_id = EXCLUDED.cluster_id,
		sequence = EXCLUDED.sequence,
		skip_preflights = EXCLUDED.skip_preflights,
		install_status = EXCLUDED.install_status,
		is_cli = EXCLUDED.is_cli,
		preflight_status = EXCLUDED.preflight_status,
		app_status = EXCLUDED.app_status,
		kots_version = EXCLUDED.kots_version`

	statement := gorqlite.ParameterizedStatement{
		Query: query,
		Arguments: []interface{}{
			createdAt.UnixMilli(),
			licenseID,
			preflightStatus.InstanceID,
			preflightStatus.ClusterID,
			preflightStatus.Sequence,
			preflightStatus.SkipPreflights,
			preflightStatus.InstallStatus,
			preflightStatus.IsCLI,
			preflightStatus.PreflightStatus,
			preflightStatus.AppStatus,
			preflightStatus.KOTSVersion,
		},
	}

	wr, err := db.WriteOneParameterized(statement)
	if err != nil {
		return fmt.Errorf("failed to write preflight report: %v: %v", err, wr.Err)
	}

	go func() {
		err := s.removeOldReportingData("preflight_report")
		if err != nil {
			logger.Warnf("failed to delete old data from preflight_report: %v", err)
		}
	}()

	return nil
}

func (s *KOTSStore) SaveReportingInfo(licenseID string, reportingInfo *reportingtypes.ReportingInfo) error {
	db := persistence.MustGetDBSession()

	createdAt := time.Now().UTC()

	query := `
	INSERT INTO instance_report (
		created_at,
		license_id,
		instance_id,
		cluster_id,
		app_status,
		is_kurl,
		kurl_node_count_total,
		kurl_node_count_ready,
		k8s_version,
		kots_version,
		kots_install_id,
		kurl_install_id,
		is_gitops_enabled,
		gitops_provider,
		downstream_channel_sequence,
		downstream_channel_id,
		downstream_channel_name,
		downstream_sequence,
		downstream_source,
		install_status,
		preflight_state,
		skip_preflights,
		repl_helm_installs,
		native_helm_installs)
	VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	ON CONFLICT(created_at) DO UPDATE SET
		license_id = EXCLUDED.license_id,
		instance_id = EXCLUDED.instance_id,
		cluster_id = EXCLUDED.cluster_id,
		app_status = EXCLUDED.app_status,
		is_kurl = EXCLUDED.is_kurl,
		kurl_node_count_total = EXCLUDED.kurl_node_count_total,
		kurl_node_count_ready = EXCLUDED.kurl_node_count_ready,
		k8s_version = EXCLUDED.k8s_version,
		kots_version = EXCLUDED.kots_version,
		kots_install_id = EXCLUDED.kots_install_id,
		kurl_install_id = EXCLUDED.kurl_install_id,
		is_gitops_enabled = EXCLUDED.is_gitops_enabled,
		gitops_provider = EXCLUDED.gitops_provider,
		downstream_channel_sequence = EXCLUDED.downstream_channel_sequence,
		downstream_channel_id = EXCLUDED.downstream_channel_id,
		downstream_channel_name = EXCLUDED.downstream_channel_name,
		downstream_sequence = EXCLUDED.downstream_sequence,
		downstream_source = EXCLUDED.downstream_source,
		install_status = EXCLUDED.install_status,
		preflight_state = EXCLUDED.preflight_state,
		skip_preflights = EXCLUDED.skip_preflights,
		repl_helm_installs = EXCLUDED.repl_helm_installs,
		native_helm_installs = EXCLUDED.native_helm_installs`

	statement := gorqlite.ParameterizedStatement{
		Query: query,
		Arguments: []interface{}{
			createdAt.UnixMilli(),
			licenseID,
			reportingInfo.InstanceID,
			reportingInfo.ClusterID,
			reportingInfo.AppStatus,
			reportingInfo.IsKurl,
			reportingInfo.KurlNodeCountTotal,
			reportingInfo.KurlNodeCountReady,
			reportingInfo.K8sVersion,
			reportingInfo.KOTSVersion,
			reportingInfo.KOTSInstallID,
			reportingInfo.KURLInstallID,
			reportingInfo.IsGitOpsEnabled,
			reportingInfo.GitOpsProvider,
			reportingInfo.Downstream.Cursor,
			reportingInfo.Downstream.ChannelID,
			reportingInfo.Downstream.ChannelName,
			reportingInfo.Downstream.Sequence,
			reportingInfo.Downstream.Source,
			reportingInfo.Downstream.Status,
			reportingInfo.Downstream.PreflightState,
			reportingInfo.Downstream.SkipPreflights,
			reportingInfo.Downstream.ReplHelmInstalls,
			reportingInfo.Downstream.NativeHelmInstalls,
		},
	}

	wr, err := db.WriteOneParameterized(statement)
	if err != nil {
		return fmt.Errorf("failed to write instance report: %v: %v", err, wr.Err)
	}

	go func() {
		err := s.removeOldReportingData("instance_report")
		if err != nil {
			logger.Warnf("failed to delete old data from instance_report: %v", err)
		}
	}()

	return nil
}

func (s *KOTSStore) removeOldReportingData(reportingTable string) error {
	db := persistence.MustGetDBSession()

	query := fmt.Sprintf(`select count(1) from %s`, reportingTable)
	rows, err := db.QueryOneParameterized(gorqlite.ParameterizedStatement{
		Query: query,
	})
	if err != nil {
		return fmt.Errorf("failed to query number of rows: %v: %v", err, rows.Err)
	}

	if !rows.Next() {
		return ErrNotFound
	}

	var numRows int64
	if err := rows.Scan(&numRows); err != nil {
		return errors.Wrap(err, "failed to scan number of rows")
	}

	reportingMaxRows := int64(4000) // at 10 records per day, this is more than a year of data
	if numRows <= reportingMaxRows {
		logger.Debugf("no old data to delete from %s", reportingTable)
		return nil
	}

	query = fmt.Sprintf(`select created_at from %s order by created_at desc limit ?, 1`, reportingTable)
	rows, err = db.QueryOneParameterized(gorqlite.ParameterizedStatement{
		Query: query,
		Arguments: []interface{}{
			reportingMaxRows,
		},
	})
	if err != nil {
		return fmt.Errorf("failed to query timestamp: %v: %v", err, rows.Err)
	}

	if !rows.Next() {
		return ErrNotFound
	}

	// timestamps are stored with millisecond precision, but scanning directly into a Time variable assumes second precision
	var timeMs int64
	if err := rows.Scan(&timeMs); err != nil {
		return errors.Wrap(err, "failed to scan timestamp")
	}
	oldestCreatedAt := time.UnixMilli(timeMs)

	query = fmt.Sprintf(`delete from %s where created_at <= ?`, reportingTable)
	wr, err := db.WriteOneParameterized(gorqlite.ParameterizedStatement{
		Query: query,
		Arguments: []interface{}{
			oldestCreatedAt.UnixMilli(),
		},
	})
	if err != nil {
		return fmt.Errorf("failed to delete: %v: %v", err, wr.Err)
	}

	return nil
}
