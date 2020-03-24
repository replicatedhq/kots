package version

import (
	"database/sql"
	"log"
	"time"

	"github.com/pkg/errors"
	"github.com/replicatedhq/kotsadm/pkg/downstream"
	"github.com/replicatedhq/kotsadm/pkg/persistence"
)

func PopulateMissingDownstreamVersions() error {
	db := persistence.MustGetPGSession()
	tx, err := db.Begin()
	if err != nil {
		return errors.Wrap(err, "failed to begin transaction")
	}
	defer tx.Rollback()

	rows, err := tx.Query(`select id from app`)
	if err != nil {
		return errors.Wrap(err, "failed query app IDs")
	}

	appIDs := make([]string, 0)
	for rows.Next() {
		var appID string
		if err := rows.Scan(&appID); err != nil {
			if err == sql.ErrNoRows {
				return nil
			}
			return errors.Wrap(err, "failed to scan app ID")
		}
		appIDs = append(appIDs, appID)
	}

	for _, appID := range appIDs {
		if err := populateMissingDownstreamVersionsForApp(tx, appID); err != nil {
			return errors.Wrapf(err, "failed to populate missing data for app %s", appID)
		}
	}

	if err = tx.Commit(); err != nil {
		return errors.Wrap(err, "failed to commit")
	}

	return nil
}

func populateMissingDownstreamVersionsForApp(tx *sql.Tx, appID string) error {
	var maxAppVersionSequenceSQL sql.NullInt64
	row := tx.QueryRow(`select max(sequence) from app_version where app_id = $1`, appID)
	if err := row.Scan(&maxAppVersionSequenceSQL); err != nil {
		return errors.Wrap(err, "failed to find current appversion max sequence in row")
	}
	if !maxAppVersionSequenceSQL.Valid {
		log.Printf("No app version found for %s; skipping migration.", appID)
		return nil
	}
	maxAppVersionSequence := maxAppVersionSequenceSQL.Int64

	downstreams, err := downstream.ListDownstreamsForApp(appID)
	if err != nil {
		if err == sql.ErrNoRows {
			log.Printf("No downstreams found for app %s; skipping migration.", appID)
			return nil
		}
		return errors.Wrap(err, "failed to list downstreams")
	}

	for _, d := range downstreams {
		query := `select max(sequence) from app_downstream_version where app_id = $1 and cluster_id = $2`
		row := tx.QueryRow(query, appID, d.ClusterID)

		var maxDownstreamSequenceSQL sql.NullInt64
		if err := row.Scan(&maxDownstreamSequenceSQL); err != nil {
			return errors.Wrapf(err, "failed to find max sequence for app %s, cluster %s", appID, d.ClusterID)
		}

		maxDownstreamSequence := int64(-1)
		if maxDownstreamSequenceSQL.Valid {
			maxDownstreamSequence = maxDownstreamSequenceSQL.Int64
		}

		if maxAppVersionSequence <= maxDownstreamSequence {
			// all sequences are present in app_downstream_version. nothing to do.
			continue
		}

		commitURL := ""
		diffSummary := ""
		downstreamStatus := "failed"
		source := "Generated"

		for maxDownstreamSequence < maxAppVersionSequence {
			var createdAt time.Time
			var versionLabel string
			maxDownstreamSequence = maxDownstreamSequence + 1

			query = `select version_label, created_at from app_version where app_id = $1 and sequence = $2`
			row := tx.QueryRow(query, appID, maxDownstreamSequence)

			var versionLabelSQL sql.NullString
			if err := row.Scan(&versionLabelSQL, &createdAt); err != nil {
				// this could be optional but because this is in a transaction,
				// no other queries can be executed after this error
				return errors.Wrap(err, "failed to load version label")
			}

			versionLabel = versionLabelSQL.String

			query = `insert into app_downstream_version (app_id, cluster_id, sequence, parent_sequence, created_at, version_label, status, source, diff_summary, git_commit_url, git_deployable) values ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)`
			_, err = tx.Exec(query, appID, d.ClusterID, maxDownstreamSequence, maxDownstreamSequence, createdAt,
				versionLabel, downstreamStatus, source,
				diffSummary, commitURL, commitURL != "")
			if err != nil {
				return errors.Wrap(err, "failed to create downstream version")
			}
		}
	}

	return nil
}
