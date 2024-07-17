package persistence

import (
	"database/sql"
	"encoding/base64"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	_ "github.com/lib/pq"
	"github.com/pkg/errors"
	"github.com/rqlite/gorqlite"
)

const (
	RQLITE_MIGRATION_SUCCESS_KEY   = "rqlite.migration.success"
	RQLITE_MIGRATION_SUCCESS_VALUE = "true"
)

func MigrateFromPostgresToRqlite() error {
	if os.Getenv("POSTGRES_URI") == "" {
		return nil
	}

	// check if we already migrated
	rqliteDB := MustGetDBSession()
	alreadyMigrated, err := isAlreadyMigrated(rqliteDB)
	if err != nil {
		return errors.Wrap(err, "failed to check if already migrated")
	}
	if alreadyMigrated {
		log.Println("Postgres instance detected but it's already been migrated to rqlite. Skipping migration...")
		return nil
	}

	log.Println("Postgres instance detected, migrating to rqlite...")
	log.Println("Updating Postgres schema...")

	// we need to bring postgres schema up to date before we can migrate
	if err := updatePostgresSchema(); err != nil {
		return errors.Wrap(err, "failed to update postgres schema")
	}

	log.Println("Migrating data to rqlite...")

	pgDB := mustGetPostgresSession()
	statements := []gorqlite.ParameterizedStatement{}

	apiTaskStatusStatements, err := apiTaskStatusTableStatements(pgDB)
	if err != nil {
		return errors.Wrap(err, "failed to construct api_task_status table statements")
	}
	statements = append(statements, apiTaskStatusStatements...)

	appDownstreamOutputStatements, err := appDownstreamOutputTableStatements(pgDB)
	if err != nil {
		return errors.Wrap(err, "failed to construct app_downstream_output table statements")
	}
	statements = append(statements, appDownstreamOutputStatements...)

	appDownstreamVersionStatements, err := appDownstreamVersionTableStatements(pgDB)
	if err != nil {
		return errors.Wrap(err, "failed to construct app_downstream_version table statements")
	}
	statements = append(statements, appDownstreamVersionStatements...)

	appDownstreamStatements, err := appDownstreamTableStatements(pgDB)
	if err != nil {
		return errors.Wrap(err, "failed to construct app_downstream table statements")
	}
	statements = append(statements, appDownstreamStatements...)

	appStatusStatements, err := appStatusTableStatements(pgDB)
	if err != nil {
		return errors.Wrap(err, "failed to construct app_status table statements")
	}
	statements = append(statements, appStatusStatements...)

	appVersionStatements, err := appVersionTableStatements(pgDB)
	if err != nil {
		return errors.Wrap(err, "failed to construct app_version table statements")
	}
	statements = append(statements, appVersionStatements...)

	appStatements, err := appTableStatements(pgDB)
	if err != nil {
		return errors.Wrap(err, "failed to construct app table statements")
	}
	statements = append(statements, appStatements...)

	clusterStatements, err := clusterTableStatements(pgDB)
	if err != nil {
		return errors.Wrap(err, "failed to construct cluster table statements")
	}
	statements = append(statements, clusterStatements...)

	initialBrandingStatements, err := initialBrandingTableStatements(pgDB)
	if err != nil {
		return errors.Wrap(err, "failed to construct initial_branding table statements")
	}
	statements = append(statements, initialBrandingStatements...)

	kotsadmParamsStatements, err := kotsadmParamsTableStatements(pgDB)
	if err != nil {
		return errors.Wrap(err, "failed to construct kotsadm_params table statements")
	}
	statements = append(statements, kotsadmParamsStatements...)

	objectStoreStatements, err := objectStoreTableStatements(pgDB)
	if err != nil {
		return errors.Wrap(err, "failed to construct object_store table statements")
	}
	statements = append(statements, objectStoreStatements...)

	pendingSupportBundleStatements, err := pendingSupportBundleTableStatements(pgDB)
	if err != nil {
		return errors.Wrap(err, "failed to construct pending_supportbundle table statements")
	}
	statements = append(statements, pendingSupportBundleStatements...)

	preflightResultStatements, err := preflightResultTableStatements(pgDB)
	if err != nil {
		return errors.Wrap(err, "failed to construct preflight_result table statements")
	}
	statements = append(statements, preflightResultStatements...)

	preflightSpecStatements, err := preflightSpecTableStatements(pgDB)
	if err != nil {
		return errors.Wrap(err, "failed to construct preflight_spec table statements")
	}
	statements = append(statements, preflightSpecStatements...)

	scheduledInstanceSnapshotsStatements, err := scheduledInstanceSnapshotsTableStatements(pgDB)
	if err != nil {
		return errors.Wrap(err, "failed to construct scheduled_instance_snapshots table statements")
	}
	statements = append(statements, scheduledInstanceSnapshotsStatements...)

	scheduledSnapshotsStatements, err := scheduledSnapshotsTableStatements(pgDB)
	if err != nil {
		return errors.Wrap(err, "failed to construct scheduled_snapshots table statements")
	}
	statements = append(statements, scheduledSnapshotsStatements...)

	sessionStatements, err := sessionTableStatements(pgDB)
	if err != nil {
		return errors.Wrap(err, "failed to construct session table statements")
	}
	statements = append(statements, sessionStatements...)

	shipUserLocalStatements, err := shipUserLocalTableStatements(pgDB)
	if err != nil {
		return errors.Wrap(err, "failed to construct ship_user_local table statements")
	}
	statements = append(statements, shipUserLocalStatements...)

	shipUserStatements, err := shipUserTableStatements(pgDB)
	if err != nil {
		return errors.Wrap(err, "failed to construct ship_user table statements")
	}
	statements = append(statements, shipUserStatements...)

	supportBundleAnalysisStatements, err := supportBundleAnalysisTableStatements(pgDB)
	if err != nil {
		return errors.Wrap(err, "failed to construct supportbundle_analysis table statements")
	}
	statements = append(statements, supportBundleAnalysisStatements...)

	supportBundleStatements, err := supportBundleTableStatements(pgDB)
	if err != nil {
		return errors.Wrap(err, "failed to construct supportbundle table statements")
	}
	statements = append(statements, supportBundleStatements...)

	userAppStatements, err := userAppTableStatements(pgDB)
	if err != nil {
		return errors.Wrap(err, "failed to construct user_app table statements")
	}
	statements = append(statements, userAppStatements...)

	userClusterStatements, err := userClusterTableStatements(pgDB)
	if err != nil {
		return errors.Wrap(err, "failed to construct user_cluster table statements")
	}
	statements = append(statements, userClusterStatements...)

	// record a successful migration
	statements = append(statements, gorqlite.ParameterizedStatement{
		Query:     "REPLACE INTO kotsadm_params (key, value) VALUES (?, ?)",
		Arguments: []interface{}{RQLITE_MIGRATION_SUCCESS_KEY, RQLITE_MIGRATION_SUCCESS_VALUE},
	})

	if wrs, err := rqliteDB.WriteParameterized(statements); err != nil {
		wrErrs := []error{}
		for _, wr := range wrs {
			wrErrs = append(wrErrs, wr.Err)
		}
		return fmt.Errorf("failed to write data to rqlite: %v: %v", err, wrErrs)
	}

	log.Println("Migrated from Postgres to rqlite successfully!")

	return nil
}

func isAlreadyMigrated(rqliteDB *gorqlite.Connection) (bool, error) {
	rows, err := rqliteDB.QueryOneParameterized(gorqlite.ParameterizedStatement{
		Query:     `SELECT value FROM kotsadm_params WHERE key = ?`,
		Arguments: []interface{}{RQLITE_MIGRATION_SUCCESS_KEY},
	})
	if err != nil {
		return false, fmt.Errorf("failed to query: %v: %v", err, rows.Err)
	}
	if !rows.Next() {
		return false, nil
	}

	var value string
	if err := rows.Scan(&value); err != nil {
		return false, errors.Wrap(err, "failed to scan")
	}

	return value == "true", nil
}

func updatePostgresSchema() error {
	return UpdateDBSchema("postgres", os.Getenv("POSTGRES_URI"), os.Getenv("POSTGRES_SCHEMA_DIR"))
}

func apiTaskStatusTableStatements(pgDB *sql.DB) ([]gorqlite.ParameterizedStatement, error) {
	statements := []gorqlite.ParameterizedStatement{}

	query := `
SELECT
	id,
	updated_at,
	current_message,
	status
FROM api_task_status`
	rows, err := pgDB.Query(query)
	if err != nil {
		return nil, errors.Wrap(err, "failed to query")
	}

	for rows.Next() {
		var id string
		var updatedAt sql.NullTime
		var message sql.NullString
		var status sql.NullString

		if err := rows.Scan(
			&id,
			&updatedAt,
			&message,
			&status,
		); err != nil {
			return nil, errors.Wrap(err, "failed to scan")
		}

		columns := []string{
			// non-nullable columns
			"id",
		}

		args := []interface{}{
			// non-nullable columns
			id,
		}

		// exclude columns with null values.
		// timestamps are converted to unix and stored as integers.
		// byte arrays are converted to base64 and stored as strings.

		if updatedAt.Valid {
			columns = append(columns, "updated_at")
			args = append(args, updatedAt.Time.Unix())
		}
		if message.Valid {
			columns = append(columns, "current_message")
			args = append(args, message.String)
		}
		if status.Valid {
			columns = append(columns, "status")
			args = append(args, status.String)
		}

		query := fmt.Sprintf(`REPLACE INTO api_task_status (%s) VALUES (%s)`, strings.Join(columns, ","), strings.Join(strings.Split(strings.Repeat("?", len(columns)), ""), ", "))
		statement := gorqlite.ParameterizedStatement{
			Query:     query,
			Arguments: args,
		}

		statements = append(statements, statement)
	}

	return statements, nil
}

func appDownstreamOutputTableStatements(pgDB *sql.DB) ([]gorqlite.ParameterizedStatement, error) {
	statements := []gorqlite.ParameterizedStatement{}

	query := `
SELECT
	app_id,
	cluster_id,
	downstream_sequence,
	dryrun_stdout,
	dryrun_stderr,
	apply_stdout,
	apply_stderr,
	helm_stdout,
	helm_stderr,
	is_error
FROM app_downstream_output`
	rows, err := pgDB.Query(query)
	if err != nil {
		return nil, errors.Wrap(err, "failed to query")
	}

	for rows.Next() {
		var appID string
		var clusterID string
		var downstreamSequence int64
		var dryrunStdout sql.NullString
		var dryrunStderr sql.NullString
		var applyStdout sql.NullString
		var applyStderr sql.NullString
		var helmStdout sql.NullString
		var helmStderr sql.NullString
		var isError sql.NullBool

		if err := rows.Scan(
			&appID,
			&clusterID,
			&downstreamSequence,
			&dryrunStdout,
			&dryrunStderr,
			&applyStdout,
			&applyStderr,
			&helmStdout,
			&helmStderr,
			&isError,
		); err != nil {
			return nil, errors.Wrap(err, "failed to scan")
		}

		columns := []string{
			// non-nullable columns
			"app_id",
			"cluster_id",
			"downstream_sequence",
		}

		args := []interface{}{
			// non-nullable columns
			appID,
			clusterID,
			downstreamSequence,
		}

		// exclude columns with null values.
		// timestamps are converted to unix and stored as integers.
		// byte arrays are converted to base64 and stored as strings.

		if dryrunStdout.Valid {
			columns = append(columns, "dryrun_stdout")
			args = append(args, dryrunStdout.String)
		}
		if dryrunStderr.Valid {
			columns = append(columns, "dryrun_stderr")
			args = append(args, dryrunStderr.String)
		}
		if applyStdout.Valid {
			columns = append(columns, "apply_stdout")
			args = append(args, applyStdout.String)
		}
		if applyStderr.Valid {
			columns = append(columns, "apply_stderr")
			args = append(args, applyStderr.String)
		}
		if helmStdout.Valid {
			columns = append(columns, "helm_stdout")
			args = append(args, helmStdout.String)
		}
		if helmStderr.Valid {
			columns = append(columns, "helm_stderr")
			args = append(args, helmStderr.String)
		}
		if isError.Valid {
			columns = append(columns, "is_error")
			args = append(args, isError.Bool)
		}

		query := fmt.Sprintf(`REPLACE INTO app_downstream_output (%s) VALUES (%s)`, strings.Join(columns, ","), strings.Join(strings.Split(strings.Repeat("?", len(columns)), ""), ", "))
		statement := gorqlite.ParameterizedStatement{
			Query:     query,
			Arguments: args,
		}

		statements = append(statements, statement)
	}

	return statements, nil
}

func appDownstreamVersionTableStatements(pgDB *sql.DB) ([]gorqlite.ParameterizedStatement, error) {
	statements := []gorqlite.ParameterizedStatement{}

	query := `
SELECT
	app_id,
	cluster_id,
	sequence,
	version_label,
	parent_sequence,
	created_at,
	applied_at,
	status,
	status_info,
	source,
	diff_summary,
	diff_summary_error,
	preflight_progress,
	preflight_result,
	preflight_result_created_at,
	preflight_ignore_permissions,
	preflight_skipped,
	git_commit_url,
	git_deployable
FROM app_downstream_version`
	rows, err := pgDB.Query(query)
	if err != nil {
		return nil, errors.Wrap(err, "failed to query")
	}

	for rows.Next() {
		var appID string
		var clusterID string
		var sequence int64
		var versionLabel string
		var parentSequence sql.NullInt64
		var createdAt NullStringTime
		var appliedAt NullStringTime
		var status sql.NullString
		var statusInfo sql.NullString
		var source sql.NullString
		var diffSummary sql.NullString
		var diffSummaryError sql.NullString
		var preflightProgress sql.NullString
		var preflightResult sql.NullString
		var preflightResultCreatedAt NullStringTime
		var preflightIgnorePermissions sql.NullBool
		var preflightSkipped sql.NullBool
		var gitCommitURL sql.NullString
		var gitDeployable sql.NullBool

		if err := rows.Scan(
			&appID,
			&clusterID,
			&sequence,
			&versionLabel,
			&parentSequence,
			&createdAt,
			&appliedAt,
			&status,
			&statusInfo,
			&source,
			&diffSummary,
			&diffSummaryError,
			&preflightProgress,
			&preflightResult,
			&preflightResultCreatedAt,
			&preflightIgnorePermissions,
			&preflightSkipped,
			&gitCommitURL,
			&gitDeployable,
		); err != nil {
			return nil, errors.Wrap(err, "failed to scan")
		}

		columns := []string{
			// non-nullable columns
			"app_id",
			"cluster_id",
			"sequence",
			"version_label",
		}

		args := []interface{}{
			// non-nullable columns
			appID,
			clusterID,
			sequence,
			versionLabel,
		}

		// exclude columns with null values.
		// timestamps are converted to unix and stored as integers.
		// byte arrays are converted to base64 and stored as strings.

		if parentSequence.Valid {
			columns = append(columns, "parent_sequence")
			args = append(args, parentSequence.Int64)
		}
		if createdAt.Valid {
			columns = append(columns, "created_at")
			args = append(args, createdAt.Time.Unix())
		}
		if appliedAt.Valid {
			columns = append(columns, "applied_at")
			args = append(args, appliedAt.Time.Unix())
		}
		if status.Valid {
			columns = append(columns, "status")
			args = append(args, status.String)
		}
		if statusInfo.Valid {
			columns = append(columns, "status_info")
			args = append(args, statusInfo.String)
		}
		if source.Valid {
			columns = append(columns, "source")
			args = append(args, source.String)
		}
		if diffSummary.Valid {
			columns = append(columns, "diff_summary")
			args = append(args, diffSummary.String)
		}
		if diffSummaryError.Valid {
			columns = append(columns, "diff_summary_error")
			args = append(args, diffSummaryError.String)
		}
		if preflightProgress.Valid {
			columns = append(columns, "preflight_progress")
			args = append(args, preflightProgress.String)
		}
		if preflightResult.Valid {
			columns = append(columns, "preflight_result")
			args = append(args, preflightResult.String)
		}
		if preflightResultCreatedAt.Valid {
			columns = append(columns, "preflight_result_created_at")
			args = append(args, preflightResultCreatedAt.Time.Unix())
		}
		if preflightIgnorePermissions.Valid {
			columns = append(columns, "preflight_ignore_permissions")
			args = append(args, preflightIgnorePermissions.Bool)
		}
		if preflightSkipped.Valid {
			columns = append(columns, "preflight_skipped")
			args = append(args, preflightSkipped.Bool)
		}
		if gitCommitURL.Valid {
			columns = append(columns, "git_commit_url")
			args = append(args, gitCommitURL.String)
		}
		if gitDeployable.Valid {
			columns = append(columns, "git_deployable")
			args = append(args, gitDeployable.Bool)
		}

		query := fmt.Sprintf(`REPLACE INTO app_downstream_version (%s) VALUES (%s)`, strings.Join(columns, ","), strings.Join(strings.Split(strings.Repeat("?", len(columns)), ""), ", "))
		statement := gorqlite.ParameterizedStatement{
			Query:     query,
			Arguments: args,
		}

		statements = append(statements, statement)
	}

	return statements, nil
}

func appDownstreamTableStatements(pgDB *sql.DB) ([]gorqlite.ParameterizedStatement, error) {
	statements := []gorqlite.ParameterizedStatement{}

	query := `
SELECT
	app_id,
	cluster_id,
	downstream_name,
	current_sequence
FROM app_downstream`
	rows, err := pgDB.Query(query)
	if err != nil {
		return nil, errors.Wrap(err, "failed to query")
	}

	for rows.Next() {
		var appID string
		var clusterID string
		var downstreamName string
		var currentSequence sql.NullInt64

		if err := rows.Scan(
			&appID,
			&clusterID,
			&downstreamName,
			&currentSequence,
		); err != nil {
			return nil, errors.Wrap(err, "failed to scan")
		}

		columns := []string{
			// non-nullable columns
			"app_id",
			"cluster_id",
			"downstream_name",
		}

		args := []interface{}{
			// non-nullable columns
			appID,
			clusterID,
			downstreamName,
		}

		// exclude columns with null values.
		// timestamps are converted to unix and stored as integers.
		// byte arrays are converted to base64 and stored as strings.

		if currentSequence.Valid {
			columns = append(columns, "current_sequence")
			args = append(args, currentSequence.Int64)
		}

		query := fmt.Sprintf(`REPLACE INTO app_downstream (%s) VALUES (%s)`, strings.Join(columns, ","), strings.Join(strings.Split(strings.Repeat("?", len(columns)), ""), ", "))
		statement := gorqlite.ParameterizedStatement{
			Query:     query,
			Arguments: args,
		}

		statements = append(statements, statement)
	}

	return statements, nil
}

func appStatusTableStatements(pgDB *sql.DB) ([]gorqlite.ParameterizedStatement, error) {
	statements := []gorqlite.ParameterizedStatement{}

	query := `
SELECT
	app_id,
	resource_states,
	updated_at,
	sequence
FROM app_status`
	rows, err := pgDB.Query(query)
	if err != nil {
		return nil, errors.Wrap(err, "failed to query")
	}

	for rows.Next() {
		var appID string
		var resourceStates sql.NullString
		var updatedAt sql.NullTime
		var sequence sql.NullInt64

		if err := rows.Scan(
			&appID,
			&resourceStates,
			&updatedAt,
			&sequence,
		); err != nil {
			return nil, errors.Wrap(err, "failed to scan")
		}

		columns := []string{
			// non-nullable columns
			"app_id",
		}

		args := []interface{}{
			// non-nullable columns
			appID,
		}

		// exclude columns with null values.
		// timestamps are converted to unix and stored as integers.
		// byte arrays are converted to base64 and stored as strings.

		if resourceStates.Valid {
			columns = append(columns, "resource_states")
			args = append(args, resourceStates.String)
		}
		if updatedAt.Valid {
			columns = append(columns, "updated_at")
			args = append(args, updatedAt.Time.Unix())
		}
		if sequence.Valid {
			columns = append(columns, "sequence")
			args = append(args, sequence.Int64)
		}

		query := fmt.Sprintf(`REPLACE INTO app_status (%s) VALUES (%s)`, strings.Join(columns, ","), strings.Join(strings.Split(strings.Repeat("?", len(columns)), ""), ", "))
		statement := gorqlite.ParameterizedStatement{
			Query:     query,
			Arguments: args,
		}

		statements = append(statements, statement)
	}

	return statements, nil
}

func appVersionTableStatements(pgDB *sql.DB) ([]gorqlite.ParameterizedStatement, error) {
	statements := []gorqlite.ParameterizedStatement{}

	query := `
SELECT
	app_id,
	sequence,
	version_label,
	is_required,
	update_cursor,
	channel_id,
	channel_name,
	upstream_released_at,
	created_at,
	release_notes,
	supportbundle_spec,
	preflight_spec,
	analyzer_spec,
	app_spec,
	kots_app_spec,
	kots_installation_spec,
	kots_license,
	config_spec,
	applied_at,
	status,
	encryption_key,
	backup_spec,
	identity_spec,
	branding_archive
FROM app_version`
	rows, err := pgDB.Query(query)
	if err != nil {
		return nil, errors.Wrap(err, "failed to query")
	}

	for rows.Next() {
		var appID string
		var sequence int64
		var versionLabel string
		var isRequired bool
		var updateCursor sql.NullString
		var channelID sql.NullString
		var channelName sql.NullString
		var upstreamReleasedAt NullStringTime
		var createdAt NullStringTime
		var releaseNotes sql.NullString
		var supportbundleSpec sql.NullString
		var preflightSpec sql.NullString
		var analyzerSpec sql.NullString
		var appSpec sql.NullString
		var kotsAppSpec sql.NullString
		var kotsInstallationSpec sql.NullString
		var kotsLicense sql.NullString
		var configSpec sql.NullString
		var appliedAt NullStringTime
		var status sql.NullString
		var encryptionKey sql.NullString
		var backupSpec sql.NullString
		var identitySpec sql.NullString
		var brandingArchive []byte

		if err := rows.Scan(
			&appID,
			&sequence,
			&versionLabel,
			&isRequired,
			&updateCursor,
			&channelID,
			&channelName,
			&upstreamReleasedAt,
			&createdAt,
			&releaseNotes,
			&supportbundleSpec,
			&preflightSpec,
			&analyzerSpec,
			&appSpec,
			&kotsAppSpec,
			&kotsInstallationSpec,
			&kotsLicense,
			&configSpec,
			&appliedAt,
			&status,
			&encryptionKey,
			&backupSpec,
			&identitySpec,
			&brandingArchive,
		); err != nil {
			return nil, errors.Wrap(err, "failed to scan")
		}

		columns := []string{
			// non-nullable columns
			"app_id",
			"sequence",
			"version_label",
			"is_required",
		}

		args := []interface{}{
			// non-nullable columns
			appID,
			sequence,
			versionLabel,
			isRequired,
		}

		// exclude columns with null values.
		// timestamps are converted to unix and stored as integers.
		// byte arrays are converted to base64 and stored as strings.

		if updateCursor.Valid {
			columns = append(columns, "update_cursor")
			args = append(args, updateCursor.String)
		}
		if channelID.Valid {
			columns = append(columns, "channel_id")
			args = append(args, channelID.String)
		}
		if channelName.Valid {
			columns = append(columns, "channel_name")
			args = append(args, channelName.String)
		}
		if upstreamReleasedAt.Valid {
			columns = append(columns, "upstream_released_at")
			args = append(args, upstreamReleasedAt.Time.Unix())
		}
		if createdAt.Valid {
			columns = append(columns, "created_at")
			args = append(args, createdAt.Time.Unix())
		}
		if releaseNotes.Valid {
			columns = append(columns, "release_notes")
			args = append(args, releaseNotes.String)
		}
		if supportbundleSpec.Valid {
			columns = append(columns, "supportbundle_spec")
			args = append(args, supportbundleSpec.String)
		}
		if preflightSpec.Valid {
			columns = append(columns, "preflight_spec")
			args = append(args, preflightSpec.String)
		}
		if analyzerSpec.Valid {
			columns = append(columns, "analyzer_spec")
			args = append(args, analyzerSpec.String)
		}
		if appSpec.Valid {
			columns = append(columns, "app_spec")
			args = append(args, appSpec.String)
		}
		if kotsAppSpec.Valid {
			columns = append(columns, "kots_app_spec")
			args = append(args, kotsAppSpec.String)
		}
		if kotsInstallationSpec.Valid {
			columns = append(columns, "kots_installation_spec")
			args = append(args, kotsInstallationSpec.String)
		}
		if kotsLicense.Valid {
			columns = append(columns, "kots_license")
			args = append(args, kotsLicense.String)
		}
		if configSpec.Valid {
			columns = append(columns, "config_spec")
			args = append(args, configSpec.String)
		}
		if appliedAt.Valid {
			columns = append(columns, "applied_at")
			args = append(args, appliedAt.Time.Unix())
		}
		if status.Valid {
			columns = append(columns, "status")
			args = append(args, status.String)
		}
		if encryptionKey.Valid {
			columns = append(columns, "encryption_key")
			args = append(args, encryptionKey.String)
		}
		if backupSpec.Valid {
			columns = append(columns, "backup_spec")
			args = append(args, backupSpec.String)
		}
		if identitySpec.Valid {
			columns = append(columns, "identity_spec")
			args = append(args, identitySpec.String)
		}
		if brandingArchive != nil {
			columns = append(columns, "branding_archive")
			args = append(args, base64.StdEncoding.EncodeToString(brandingArchive))
		}

		query := fmt.Sprintf(`REPLACE INTO app_version (%s) VALUES (%s)`, strings.Join(columns, ","), strings.Join(strings.Split(strings.Repeat("?", len(columns)), ""), ", "))
		statement := gorqlite.ParameterizedStatement{
			Query:     query,
			Arguments: args,
		}

		statements = append(statements, statement)
	}

	return statements, nil
}

func appTableStatements(pgDB *sql.DB) ([]gorqlite.ParameterizedStatement, error) {
	statements := []gorqlite.ParameterizedStatement{}

	query := `
SELECT
	id,
	name,
	created_at,
	slug,
	upstream_uri,
	channel_changed,
	icon_uri,
	updated_at,
	license,
	current_sequence,
	last_update_check_at,
	is_all_users,
	registry_hostname,
	registry_username,
	registry_password,
	registry_password_enc,
	namespace,
	registry_is_readonly,
	last_registry_sync,
	last_license_sync,
	install_state,
	is_airgap,
	snapshot_ttl_new,
	snapshot_schedule,
	restore_in_progress_name,
	restore_undeploy_status,
	update_checker_spec,
	semver_auto_deploy
FROM app`
	rows, err := pgDB.Query(query)
	if err != nil {
		return nil, errors.Wrap(err, "failed to query")
	}

	for rows.Next() {
		var id string
		var name string
		var createdAt StringTime
		var slug string
		var upstreamURI string
		var channelChanged bool
		var iconURI sql.NullString
		var updatedAt NullStringTime
		var license sql.NullString
		var currentSequence sql.NullInt64
		var lastUpdateCheckAt NullStringTime
		var isAllUsers sql.NullBool
		var registryHostname sql.NullString
		var registryUsername sql.NullString
		var registryPassword sql.NullString
		var registryPasswordEnc sql.NullString
		var namespace sql.NullString
		var registryIsReadonly sql.NullBool
		var lastRegistrySync NullStringTime
		var lastLicenseSync NullStringTime
		var installState sql.NullString
		var isAirgap sql.NullBool
		var snapshotTTLNew sql.NullString
		var snapshotSchedule sql.NullString
		var restoreInProgressName sql.NullString
		var restoreUndeployStatus sql.NullString
		var updateCheckerSpec sql.NullString
		var semverAutoDeploy sql.NullString

		if err := rows.Scan(
			&id,
			&name,
			&createdAt,
			&slug,
			&upstreamURI,
			&channelChanged,
			&iconURI,
			&updatedAt,
			&license,
			&currentSequence,
			&lastUpdateCheckAt,
			&isAllUsers,
			&registryHostname,
			&registryUsername,
			&registryPassword,
			&registryPasswordEnc,
			&namespace,
			&registryIsReadonly,
			&lastRegistrySync,
			&lastLicenseSync,
			&installState,
			&isAirgap,
			&snapshotTTLNew,
			&snapshotSchedule,
			&restoreInProgressName,
			&restoreUndeployStatus,
			&updateCheckerSpec,
			&semverAutoDeploy,
		); err != nil {
			return nil, errors.Wrap(err, "failed to scan")
		}

		columns := []string{
			// non-nullable columns
			"id",
			"name",
			"created_at",
			"slug",
			"upstream_uri",
			"channel_changed",
		}

		args := []interface{}{
			// non-nullable columns
			id,
			name,
			createdAt.Time.Unix(),
			slug,
			upstreamURI,
			channelChanged,
		}

		// exclude columns with null values.
		// timestamps are converted to unix and stored as integers.
		// byte arrays are converted to base64 and stored as strings.

		if iconURI.Valid {
			columns = append(columns, "icon_uri")
			args = append(args, iconURI.String)
		}
		if updatedAt.Valid {
			columns = append(columns, "updated_at")
			args = append(args, updatedAt.Time.Unix())
		}
		if license.Valid {
			columns = append(columns, "license")
			args = append(args, license.String)
		}
		if currentSequence.Valid {
			columns = append(columns, "current_sequence")
			args = append(args, currentSequence.Int64)
		}
		if lastUpdateCheckAt.Valid {
			columns = append(columns, "last_update_check_at")
			args = append(args, lastUpdateCheckAt.Time.Unix())
		}
		if isAllUsers.Valid {
			columns = append(columns, "is_all_users")
			args = append(args, isAllUsers.Bool)
		}
		if registryHostname.Valid {
			columns = append(columns, "registry_hostname")
			args = append(args, registryHostname.String)
		}
		if registryUsername.Valid {
			columns = append(columns, "registry_username")
			args = append(args, registryUsername.String)
		}
		if registryPassword.Valid {
			columns = append(columns, "registry_password")
			args = append(args, registryPassword.String)
		}
		if registryPasswordEnc.Valid {
			columns = append(columns, "registry_password_enc")
			args = append(args, registryPasswordEnc.String)
		}
		if namespace.Valid {
			columns = append(columns, "namespace")
			args = append(args, namespace.String)
		}
		if registryIsReadonly.Valid {
			columns = append(columns, "registry_is_readonly")
			args = append(args, registryIsReadonly.Bool)
		}
		if lastRegistrySync.Valid {
			columns = append(columns, "last_registry_sync")
			args = append(args, lastRegistrySync.Time.Unix())
		}
		if lastLicenseSync.Valid {
			columns = append(columns, "last_license_sync")
			args = append(args, lastLicenseSync.Time.Unix())
		}
		if installState.Valid {
			columns = append(columns, "install_state")
			args = append(args, installState.String)
		}
		if isAirgap.Valid {
			columns = append(columns, "is_airgap")
			args = append(args, isAirgap.Bool)
		}
		if snapshotTTLNew.Valid {
			columns = append(columns, "snapshot_ttl_new")
			args = append(args, snapshotTTLNew.String)
		}
		if snapshotSchedule.Valid {
			columns = append(columns, "snapshot_schedule")
			args = append(args, snapshotSchedule.String)
		}
		if restoreInProgressName.Valid {
			columns = append(columns, "restore_in_progress_name")
			args = append(args, restoreInProgressName.String)
		}
		if restoreUndeployStatus.Valid {
			columns = append(columns, "restore_undeploy_status")
			args = append(args, restoreUndeployStatus.String)
		}
		if updateCheckerSpec.Valid {
			columns = append(columns, "update_checker_spec")
			args = append(args, updateCheckerSpec.String)
		}
		if semverAutoDeploy.Valid {
			columns = append(columns, "semver_auto_deploy")
			args = append(args, semverAutoDeploy.String)
		}

		query := fmt.Sprintf(`REPLACE INTO app (%s) VALUES (%s)`, strings.Join(columns, ","), strings.Join(strings.Split(strings.Repeat("?", len(columns)), ""), ", "))
		statement := gorqlite.ParameterizedStatement{
			Query:     query,
			Arguments: args,
		}

		statements = append(statements, statement)
	}

	return statements, nil
}

func clusterTableStatements(pgDB *sql.DB) ([]gorqlite.ParameterizedStatement, error) {
	statements := []gorqlite.ParameterizedStatement{}

	query := `
SELECT
	id,
	title,
	slug,
	created_at,
	cluster_type,
	is_all_users,
	snapshot_ttl,
	updated_at,
	token,
	snapshot_schedule
FROM cluster`
	rows, err := pgDB.Query(query)
	if err != nil {
		return nil, errors.Wrap(err, "failed to query")
	}

	for rows.Next() {
		var id string
		var title string
		var slug string
		var createdAt StringTime
		var clusterType string
		var isAllUsers bool
		var snapshotTTL string
		var updatedAt sql.NullTime
		var token sql.NullString
		var snapshotSchedule sql.NullString

		if err := rows.Scan(
			&id,
			&title,
			&slug,
			&createdAt,
			&clusterType,
			&isAllUsers,
			&snapshotTTL,
			&updatedAt,
			&token,
			&snapshotSchedule,
		); err != nil {
			return nil, errors.Wrap(err, "failed to scan")
		}

		columns := []string{
			// non-nullable columns
			"id",
			"title",
			"slug",
			"created_at",
			"cluster_type",
			"is_all_users",
			"snapshot_ttl",
		}

		args := []interface{}{
			// non-nullable columns
			id,
			title,
			slug,
			createdAt.Time.Unix(),
			clusterType,
			isAllUsers,
			snapshotTTL,
		}

		// exclude columns with null values.
		// timestamps are converted to unix and stored as integers.
		// byte arrays are converted to base64 and stored as strings.

		if updatedAt.Valid {
			columns = append(columns, "updated_at")
			args = append(args, updatedAt.Time.Unix())
		}
		if token.Valid {
			columns = append(columns, "token")
			args = append(args, token.String)
		}
		if snapshotSchedule.Valid {
			columns = append(columns, "snapshot_schedule")
			args = append(args, snapshotSchedule.String)
		}

		query := fmt.Sprintf(`REPLACE INTO cluster (%s) VALUES (%s)`, strings.Join(columns, ","), strings.Join(strings.Split(strings.Repeat("?", len(columns)), ""), ", "))
		statement := gorqlite.ParameterizedStatement{
			Query:     query,
			Arguments: args,
		}

		statements = append(statements, statement)
	}

	return statements, nil
}

func initialBrandingTableStatements(pgDB *sql.DB) ([]gorqlite.ParameterizedStatement, error) {
	statements := []gorqlite.ParameterizedStatement{}

	query := `
SELECT
	id,
	contents,
	created_at
FROM initial_branding`
	rows, err := pgDB.Query(query)
	if err != nil {
		return nil, errors.Wrap(err, "failed to query")
	}

	for rows.Next() {
		var id string
		var contents []byte
		var createdAt StringTime

		if err := rows.Scan(
			&id,
			&contents,
			&createdAt,
		); err != nil {
			return nil, errors.Wrap(err, "failed to scan")
		}

		columns := []string{
			// non-nullable columns
			"id",
			"contents",
			"created_at",
		}

		args := []interface{}{
			// non-nullable columns
			id,
			base64.StdEncoding.EncodeToString(contents),
			createdAt.Time.Unix(),
		}

		query := fmt.Sprintf(`REPLACE INTO initial_branding (%s) VALUES (%s)`, strings.Join(columns, ","), strings.Join(strings.Split(strings.Repeat("?", len(columns)), ""), ", "))
		statement := gorqlite.ParameterizedStatement{
			Query:     query,
			Arguments: args,
		}

		statements = append(statements, statement)
	}

	return statements, nil
}

func kotsadmParamsTableStatements(pgDB *sql.DB) ([]gorqlite.ParameterizedStatement, error) {
	statements := []gorqlite.ParameterizedStatement{}

	query := `
SELECT
	key,
	value
FROM kotsadm_params`
	rows, err := pgDB.Query(query)
	if err != nil {
		return nil, errors.Wrap(err, "failed to query")
	}

	for rows.Next() {
		var key string
		var value string

		if err := rows.Scan(
			&key,
			&value,
		); err != nil {
			return nil, errors.Wrap(err, "failed to scan")
		}

		columns := []string{
			// non-nullable columns
			"key",
			"value",
		}

		args := []interface{}{
			// non-nullable columns
			key,
			value,
		}

		query := fmt.Sprintf(`REPLACE INTO kotsadm_params (%s) VALUES (%s)`, strings.Join(columns, ","), strings.Join(strings.Split(strings.Repeat("?", len(columns)), ""), ", "))
		statement := gorqlite.ParameterizedStatement{
			Query:     query,
			Arguments: args,
		}

		statements = append(statements, statement)
	}

	return statements, nil
}

func objectStoreTableStatements(pgDB *sql.DB) ([]gorqlite.ParameterizedStatement, error) {
	statements := []gorqlite.ParameterizedStatement{}

	query := `
SELECT
	filepath,
	encoded_block
FROM object_store`
	rows, err := pgDB.Query(query)
	if err != nil {
		return nil, errors.Wrap(err, "failed to query")
	}

	for rows.Next() {
		var filepath string
		var encodedBlock string

		if err := rows.Scan(
			&filepath,
			&encodedBlock,
		); err != nil {
			return nil, errors.Wrap(err, "failed to scan")
		}

		columns := []string{
			// non-nullable columns
			"filepath",
			"encoded_block",
		}

		args := []interface{}{
			// non-nullable columns
			filepath,
			encodedBlock,
		}

		query := fmt.Sprintf(`REPLACE INTO object_store (%s) VALUES (%s)`, strings.Join(columns, ","), strings.Join(strings.Split(strings.Repeat("?", len(columns)), ""), ", "))
		statement := gorqlite.ParameterizedStatement{
			Query:     query,
			Arguments: args,
		}

		statements = append(statements, statement)
	}

	return statements, nil
}

func pendingSupportBundleTableStatements(pgDB *sql.DB) ([]gorqlite.ParameterizedStatement, error) {
	statements := []gorqlite.ParameterizedStatement{}

	query := `
SELECT
	id,
	app_id,
	cluster_id,
	created_at
FROM pending_supportbundle`
	rows, err := pgDB.Query(query)
	if err != nil {
		return nil, errors.Wrap(err, "failed to query")
	}

	for rows.Next() {
		var id string
		var appID string
		var clusterID string
		var createdAt NullStringTime

		if err := rows.Scan(
			&id,
			&appID,
			&clusterID,
			&createdAt,
		); err != nil {
			return nil, errors.Wrap(err, "failed to scan")
		}

		columns := []string{
			// non-nullable columns
			"id",
			"app_id",
			"cluster_id",
		}

		args := []interface{}{
			// non-nullable columns
			id,
			appID,
			clusterID,
		}

		// exclude columns with null values.
		// timestamps are converted to unix and stored as integers.
		// byte arrays are converted to base64 and stored as strings.

		if createdAt.Valid {
			columns = append(columns, "created_at")
			args = append(args, createdAt.Time.Unix())
		}

		query := fmt.Sprintf(`REPLACE INTO pending_supportbundle (%s) VALUES (%s)`, strings.Join(columns, ","), strings.Join(strings.Split(strings.Repeat("?", len(columns)), ""), ", "))
		statement := gorqlite.ParameterizedStatement{
			Query:     query,
			Arguments: args,
		}

		statements = append(statements, statement)
	}

	return statements, nil
}

func preflightResultTableStatements(pgDB *sql.DB) ([]gorqlite.ParameterizedStatement, error) {
	statements := []gorqlite.ParameterizedStatement{}

	query := `
SELECT
	id,
	watch_id,
	result,
	created_at
FROM preflight_result`
	rows, err := pgDB.Query(query)
	if err != nil {
		return nil, errors.Wrap(err, "failed to query")
	}

	for rows.Next() {
		var id string
		var watchID string
		var result string
		var createdAt StringTime

		if err := rows.Scan(
			&id,
			&watchID,
			&result,
			&createdAt,
		); err != nil {
			return nil, errors.Wrap(err, "failed to scan")
		}

		columns := []string{
			// non-nullable columns
			"id",
			"watch_id",
			"result",
			"created_at",
		}

		args := []interface{}{
			// non-nullable columns
			id,
			watchID,
			result,
			createdAt.Time.Unix(),
		}

		query := fmt.Sprintf(`REPLACE INTO preflight_result (%s) VALUES (%s)`, strings.Join(columns, ","), strings.Join(strings.Split(strings.Repeat("?", len(columns)), ""), ", "))
		statement := gorqlite.ParameterizedStatement{
			Query:     query,
			Arguments: args,
		}

		statements = append(statements, statement)
	}

	return statements, nil
}

func preflightSpecTableStatements(pgDB *sql.DB) ([]gorqlite.ParameterizedStatement, error) {
	statements := []gorqlite.ParameterizedStatement{}

	query := `
SELECT
	watch_id,
	sequence,
	spec
FROM preflight_spec`
	rows, err := pgDB.Query(query)
	if err != nil {
		return nil, errors.Wrap(err, "failed to query")
	}

	for rows.Next() {
		var watchID string
		var sequence int64
		var spec string

		if err := rows.Scan(
			&watchID,
			&sequence,
			&spec,
		); err != nil {
			return nil, errors.Wrap(err, "failed to scan")
		}

		columns := []string{
			// non-nullable columns
			"watch_id",
			"sequence",
			"spec",
		}

		args := []interface{}{
			// non-nullable columns
			watchID,
			sequence,
			spec,
		}

		query := fmt.Sprintf(`REPLACE INTO preflight_spec (%s) VALUES (%s)`, strings.Join(columns, ","), strings.Join(strings.Split(strings.Repeat("?", len(columns)), ""), ", "))
		statement := gorqlite.ParameterizedStatement{
			Query:     query,
			Arguments: args,
		}

		statements = append(statements, statement)
	}

	return statements, nil
}

func scheduledInstanceSnapshotsTableStatements(pgDB *sql.DB) ([]gorqlite.ParameterizedStatement, error) {
	statements := []gorqlite.ParameterizedStatement{}

	query := `
SELECT
	id,
	cluster_id,
	scheduled_timestamp,
	backup_name
FROM scheduled_instance_snapshots`
	rows, err := pgDB.Query(query)
	if err != nil {
		return nil, errors.Wrap(err, "failed to query")
	}

	for rows.Next() {
		var id string
		var clusterID string
		var scheduledTimestamp time.Time
		var backupName sql.NullString

		if err := rows.Scan(
			&id,
			&clusterID,
			&scheduledTimestamp,
			&backupName,
		); err != nil {
			return nil, errors.Wrap(err, "failed to scan")
		}

		columns := []string{
			// non-nullable columns
			"id",
			"cluster_id",
			"scheduled_timestamp",
		}

		args := []interface{}{
			// non-nullable columns
			id,
			clusterID,
			scheduledTimestamp.Unix(),
		}

		// exclude columns with null values.
		// timestamps are converted to unix and stored as integers.
		// byte arrays are converted to base64 and stored as strings.

		if backupName.Valid {
			columns = append(columns, "backup_name")
			args = append(args, backupName.String)
		}

		query := fmt.Sprintf(`REPLACE INTO scheduled_instance_snapshots (%s) VALUES (%s)`, strings.Join(columns, ","), strings.Join(strings.Split(strings.Repeat("?", len(columns)), ""), ", "))
		statement := gorqlite.ParameterizedStatement{
			Query:     query,
			Arguments: args,
		}

		statements = append(statements, statement)
	}

	return statements, nil
}

func scheduledSnapshotsTableStatements(pgDB *sql.DB) ([]gorqlite.ParameterizedStatement, error) {
	statements := []gorqlite.ParameterizedStatement{}

	query := `
SELECT
	id,
	app_id,
	scheduled_timestamp,
	backup_name
FROM scheduled_snapshots`
	rows, err := pgDB.Query(query)
	if err != nil {
		return nil, errors.Wrap(err, "failed to query")
	}

	for rows.Next() {
		var id string
		var appID string
		var scheduledTimestamp time.Time
		var backupName sql.NullString

		if err := rows.Scan(
			&id,
			&appID,
			&scheduledTimestamp,
			&backupName,
		); err != nil {
			return nil, errors.Wrap(err, "failed to scan")
		}

		columns := []string{
			// non-nullable columns
			"id",
			"app_id",
			"scheduled_timestamp",
		}

		args := []interface{}{
			// non-nullable columns
			id,
			appID,
			scheduledTimestamp.Unix(),
		}

		// exclude columns with null values.
		// timestamps are converted to unix and stored as integers.
		// byte arrays are converted to base64 and stored as strings.

		if backupName.Valid {
			columns = append(columns, "backup_name")
			args = append(args, backupName.String)
		}

		query := fmt.Sprintf(`REPLACE INTO scheduled_snapshots (%s) VALUES (%s)`, strings.Join(columns, ","), strings.Join(strings.Split(strings.Repeat("?", len(columns)), ""), ", "))
		statement := gorqlite.ParameterizedStatement{
			Query:     query,
			Arguments: args,
		}

		statements = append(statements, statement)
	}

	return statements, nil
}

func sessionTableStatements(pgDB *sql.DB) ([]gorqlite.ParameterizedStatement, error) {
	statements := []gorqlite.ParameterizedStatement{}

	query := `
SELECT
	id,
	user_id,
	metadata,
	expire_at,
	issued_at
FROM session`
	rows, err := pgDB.Query(query)
	if err != nil {
		return nil, errors.Wrap(err, "failed to query")
	}

	for rows.Next() {
		var id string
		var userID string
		var metadata string
		var expireAt time.Time
		var issuedAt sql.NullTime

		if err := rows.Scan(
			&id,
			&userID,
			&metadata,
			&expireAt,
			&issuedAt,
		); err != nil {
			return nil, errors.Wrap(err, "failed to scan")
		}

		columns := []string{
			// non-nullable columns
			"id",
			"user_id",
			"metadata",
			"expire_at",
		}

		args := []interface{}{
			// non-nullable columns
			id,
			userID,
			metadata,
			expireAt.Unix(),
		}

		// exclude columns with null values.
		// timestamps are converted to unix and stored as integers.
		// byte arrays are converted to base64 and stored as strings.

		if issuedAt.Valid {
			columns = append(columns, "issued_at")
			args = append(args, issuedAt.Time.Unix())
		}

		query := fmt.Sprintf(`REPLACE INTO session (%s) VALUES (%s)`, strings.Join(columns, ","), strings.Join(strings.Split(strings.Repeat("?", len(columns)), ""), ", "))
		statement := gorqlite.ParameterizedStatement{
			Query:     query,
			Arguments: args,
		}

		statements = append(statements, statement)
	}

	return statements, nil
}

func shipUserLocalTableStatements(pgDB *sql.DB) ([]gorqlite.ParameterizedStatement, error) {
	statements := []gorqlite.ParameterizedStatement{}

	query := `
SELECT
	user_id,
	password_bcrypt,
	email,
	first_name,
	last_name
FROM ship_user_local`
	rows, err := pgDB.Query(query)
	if err != nil {
		return nil, errors.Wrap(err, "failed to query")
	}

	for rows.Next() {
		var userID string
		var passwordBcrypt string
		var email string
		var firstName sql.NullString
		var lastName sql.NullString

		if err := rows.Scan(
			&userID,
			&passwordBcrypt,
			&email,
			&firstName,
			&lastName,
		); err != nil {
			return nil, errors.Wrap(err, "failed to scan")
		}

		columns := []string{
			// non-nullable columns
			"user_id",
			"password_bcrypt",
			"email",
		}

		args := []interface{}{
			// non-nullable columns
			userID,
			passwordBcrypt,
			email,
		}

		// exclude columns with null values.
		// timestamps are converted to unix and stored as integers.
		// byte arrays are converted to base64 and stored as strings.

		if firstName.Valid {
			columns = append(columns, "first_name")
			args = append(args, firstName.String)
		}
		if lastName.Valid {
			columns = append(columns, "last_name")
			args = append(args, lastName.String)
		}

		query := fmt.Sprintf(`REPLACE INTO ship_user_local (%s) VALUES (%s)`, strings.Join(columns, ","), strings.Join(strings.Split(strings.Repeat("?", len(columns)), ""), ", "))
		statement := gorqlite.ParameterizedStatement{
			Query:     query,
			Arguments: args,
		}

		statements = append(statements, statement)
	}

	return statements, nil
}

func shipUserTableStatements(pgDB *sql.DB) ([]gorqlite.ParameterizedStatement, error) {
	statements := []gorqlite.ParameterizedStatement{}

	query := `
SELECT
	id,
	created_at,
	github_id,
	last_login
FROM ship_user`
	rows, err := pgDB.Query(query)
	if err != nil {
		return nil, errors.Wrap(err, "failed to query")
	}

	for rows.Next() {
		var id string
		var createdAt sql.NullTime
		var githubID sql.NullString
		var lastLogin sql.NullTime

		if err := rows.Scan(
			&id,
			&createdAt,
			&githubID,
			&lastLogin,
		); err != nil {
			return nil, errors.Wrap(err, "failed to scan")
		}

		columns := []string{
			// non-nullable columns
			"id",
		}

		args := []interface{}{
			// non-nullable columns
			id,
		}

		// exclude columns with null values.
		// timestamps are converted to unix and stored as integers.
		// byte arrays are converted to base64 and stored as strings.

		if createdAt.Valid {
			columns = append(columns, "created_at")
			args = append(args, createdAt.Time.Unix())
		}
		if githubID.Valid {
			columns = append(columns, "github_id")
			args = append(args, githubID.String)
		}
		if lastLogin.Valid {
			columns = append(columns, "last_login")
			args = append(args, lastLogin.Time.Unix())
		}

		query := fmt.Sprintf(`REPLACE INTO ship_user (%s) VALUES (%s)`, strings.Join(columns, ","), strings.Join(strings.Split(strings.Repeat("?", len(columns)), ""), ", "))
		statement := gorqlite.ParameterizedStatement{
			Query:     query,
			Arguments: args,
		}

		statements = append(statements, statement)
	}

	return statements, nil
}

func supportBundleAnalysisTableStatements(pgDB *sql.DB) ([]gorqlite.ParameterizedStatement, error) {
	statements := []gorqlite.ParameterizedStatement{}

	query := `
SELECT
	id,
	supportbundle_id,
	created_at,
	error,
	max_severity,
	insights
FROM supportbundle_analysis`
	rows, err := pgDB.Query(query)
	if err != nil {
		return nil, errors.Wrap(err, "failed to query")
	}

	for rows.Next() {
		var id string
		var supportbundleID string
		var createdAt time.Time
		var errorStr sql.NullString
		var maxSeverity sql.NullString
		var insights sql.NullString

		if err := rows.Scan(
			&id,
			&supportbundleID,
			&createdAt,
			&errorStr,
			&maxSeverity,
			&insights,
		); err != nil {
			return nil, errors.Wrap(err, "failed to scan")
		}

		columns := []string{
			// non-nullable columns
			"id",
			"supportbundle_id",
			"created_at",
		}

		args := []interface{}{
			// non-nullable columns
			id,
			supportbundleID,
			createdAt.Unix(),
		}

		// exclude columns with null values.
		// timestamps are converted to unix and stored as integers.
		// byte arrays are converted to base64 and stored as strings.

		if errorStr.Valid {
			columns = append(columns, "error")
			args = append(args, errorStr.String)
		}
		if maxSeverity.Valid {
			columns = append(columns, "max_severity")
			args = append(args, maxSeverity.String)
		}
		if insights.Valid {
			columns = append(columns, "insights")
			args = append(args, insights.String)
		}

		query := fmt.Sprintf(`REPLACE INTO supportbundle_analysis (%s) VALUES (%s)`, strings.Join(columns, ","), strings.Join(strings.Split(strings.Repeat("?", len(columns)), ""), ", "))
		statement := gorqlite.ParameterizedStatement{
			Query:     query,
			Arguments: args,
		}

		statements = append(statements, statement)
	}

	return statements, nil
}

func supportBundleTableStatements(pgDB *sql.DB) ([]gorqlite.ParameterizedStatement, error) {
	statements := []gorqlite.ParameterizedStatement{}

	query := `
SELECT
	id,
	slug,
	watch_id,
	status,
	created_at,
	name,
	size,
	tree_index,
	analysis_id,
	uploaded_at,
	shared_at,
	is_archived,
	redact_report
FROM supportbundle`
	rows, err := pgDB.Query(query)
	if err != nil {
		return nil, errors.Wrap(err, "failed to query")
	}

	for rows.Next() {
		var id string
		var slug string
		var watchID string
		var status string
		var createdAt time.Time
		var name sql.NullString
		var size sql.NullFloat64
		var treeIndex sql.NullString
		var analysisID sql.NullString
		var uploadedAt sql.NullTime
		var sharedAt sql.NullTime
		var isArchived sql.NullBool
		var redactReport sql.NullString

		if err := rows.Scan(
			&id,
			&slug,
			&watchID,
			&status,
			&createdAt,
			&name,
			&size,
			&treeIndex,
			&analysisID,
			&uploadedAt,
			&sharedAt,
			&isArchived,
			&redactReport,
		); err != nil {
			return nil, errors.Wrap(err, "failed to scan")
		}

		columns := []string{
			// non-nullable columns
			"id",
			"slug",
			"watch_id",
			"status",
			"created_at",
		}

		args := []interface{}{
			// non-nullable columns
			id,
			slug,
			watchID,
			status,
			createdAt.Unix(),
		}

		// exclude columns with null values.
		// timestamps are converted to unix and stored as integers.
		// byte arrays are converted to base64 and stored as strings.

		if name.Valid {
			columns = append(columns, "name")
			args = append(args, name.String)
		}
		if size.Valid {
			columns = append(columns, "size")
			args = append(args, size.Float64)
		}
		if treeIndex.Valid {
			columns = append(columns, "tree_index")
			args = append(args, treeIndex.String)
		}
		if analysisID.Valid {
			columns = append(columns, "analysis_id")
			args = append(args, analysisID.String)
		}
		if uploadedAt.Valid {
			columns = append(columns, "uploaded_at")
			args = append(args, uploadedAt.Time.Unix())
		}
		if sharedAt.Valid {
			columns = append(columns, "shared_at")
			args = append(args, sharedAt.Time.Unix())
		}
		if isArchived.Valid {
			columns = append(columns, "is_archived")
			args = append(args, isArchived.Bool)
		}
		if redactReport.Valid {
			columns = append(columns, "redact_report")
			args = append(args, redactReport.String)
		}

		query := fmt.Sprintf(`REPLACE INTO supportbundle (%s) VALUES (%s)`, strings.Join(columns, ","), strings.Join(strings.Split(strings.Repeat("?", len(columns)), ""), ", "))
		statement := gorqlite.ParameterizedStatement{
			Query:     query,
			Arguments: args,
		}

		statements = append(statements, statement)
	}

	return statements, nil
}

func userAppTableStatements(pgDB *sql.DB) ([]gorqlite.ParameterizedStatement, error) {
	statements := []gorqlite.ParameterizedStatement{}

	query := `
SELECT
	user_id,
	app_id
FROM user_app`
	rows, err := pgDB.Query(query)
	if err != nil {
		return nil, errors.Wrap(err, "failed to query")
	}

	for rows.Next() {
		var userID sql.NullString
		var appID sql.NullString

		if err := rows.Scan(
			&userID,
			&appID,
		); err != nil {
			return nil, errors.Wrap(err, "failed to scan")
		}

		columns := []string{
			// non-nullable columns
		}

		args := []interface{}{
			// non-nullable columns
		}

		// exclude columns with null values.
		// timestamps are converted to unix and stored as integers.
		// byte arrays are converted to base64 and stored as strings.

		if userID.Valid {
			columns = append(columns, "user_id")
			args = append(args, userID.String)
		}
		if appID.Valid {
			columns = append(columns, "app_id")
			args = append(args, appID.String)
		}

		query := fmt.Sprintf(`REPLACE INTO user_app (%s) VALUES (%s)`, strings.Join(columns, ","), strings.Join(strings.Split(strings.Repeat("?", len(columns)), ""), ", "))
		statement := gorqlite.ParameterizedStatement{
			Query:     query,
			Arguments: args,
		}

		statements = append(statements, statement)
	}

	return statements, nil
}

func userClusterTableStatements(pgDB *sql.DB) ([]gorqlite.ParameterizedStatement, error) {
	statements := []gorqlite.ParameterizedStatement{}

	query := `
SELECT
	user_id,
	cluster_id
FROM user_cluster`
	rows, err := pgDB.Query(query)
	if err != nil {
		return nil, errors.Wrap(err, "failed to query")
	}

	for rows.Next() {
		var userID string
		var clusterID string

		if err := rows.Scan(
			&userID,
			&clusterID,
		); err != nil {
			return nil, errors.Wrap(err, "failed to scan")
		}

		columns := []string{
			// non-nullable columns
			"user_id",
			"cluster_id",
		}

		args := []interface{}{
			// non-nullable columns
			userID,
			clusterID,
		}

		query := fmt.Sprintf(`REPLACE INTO user_cluster (%s) VALUES (%s)`, strings.Join(columns, ","), strings.Join(strings.Split(strings.Repeat("?", len(columns)), ""), ", "))
		statement := gorqlite.ParameterizedStatement{
			Query:     query,
			Arguments: args,
		}

		statements = append(statements, statement)
	}

	return statements, nil
}

func mustGetPostgresSession() *sql.DB {
	pgDB, err := sql.Open("postgres", os.Getenv("POSTGRES_URI"))
	if err != nil {
		fmt.Printf("error connecting to postgres: %v\n", err)
		panic(err)
	}
	return pgDB
}
