package version

import (
	"database/sql"
	"log"
	"time"

	"github.com/pkg/errors"
	"github.com/replicatedhq/kotsadm/pkg/downstream"
	"github.com/replicatedhq/kotsadm/pkg/downstream/types"
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
		downstreams, err := downstream.ListDownstreamsForApp(appID)
		if err != nil {
			return errors.Wrapf(err, "failed to get downstreams for app", appID)
		}
		for _, downstream := range downstreams {
			if err := populateMissingDownstreamVersionsForApp(tx, appID, downstream); err != nil {
				return errors.Wrapf(err, "failed to populate missing data for app %s", appID)
			}
		}
	}

	if err = tx.Commit(); err != nil {
		return errors.Wrap(err, "failed to commit")
	}

	return nil
}

type appDownstreamVersion struct {
	Sequence       int64
	ParentSequence int64
	CreatedAt      time.Time
	VersionLabel   string
	Status         sql.NullString
}

type appVersion struct {
	Sequence     int64
	UpdateCursor sql.NullString
	CreatedAt    time.Time
	VersionLabel string
}

// This will attempt to make sequences in app_version and app_downstream_version match.
// The descision will based on `created_at` timestamp in each table.
// Starting with the first record in app_version, the matching record in app_downstream_version should be (slightly) older.
// If for a app_version record we see a app_downstream_version record that's newer, we remove it.
// The record removed cannot be matching an earlier record in app_version because all previous records have been reconciled.
// Caveat: this is tailored to a specific database problem.
func populateMissingDownstreamVersionsForApp(tx *sql.Tx, appID string, downstream *types.Downstream) error {
	versions, err := getAppVersions(tx, appID)
	if err != nil {
		return errors.Wrapf(err, "failed to load versions for app %s", appID)
	}
	if len(versions) == 0 {
		return nil
	}

	downstreamVersions, err := getAppDownstreamVersions(tx, appID, downstream.ClusterID)
	if err != nil {
		return errors.Wrapf(err, "failed to load dowstream versions for app %s in cluster %s", appID, downstream.ClusterID)
	}
	if len(downstreamVersions) == 0 {
		return nil
	}

	newDownstreamSequence := downstream.CurrentSequence
VERSIONS:
	for idx1, version := range versions {
		versionsRemoved := false
		for idx2 := idx1; idx2 < len(downstreamVersions); idx2++ {
			downstreamVersion := downstreamVersions[idx2]

			if downstreamVersion.CreatedAt.After(version.CreatedAt) {
				if !versionsRemoved {
					continue VERSIONS
				}

				log.Printf("Will change downstream version sequence: app=%s, cluster=%s, sequence=%d, new sequence=%d", appID, downstream.ClusterID, downstreamVersion.Sequence, version.Sequence)
				err := setDownstreamVersionSequence(tx, appID, downstream.ClusterID, downstreamVersion.Sequence, version.Sequence)
				if err != nil {
					return errors.Wrapf(err, "failed to update dowstream versions for app %s in cluster %s, current sequence %d, new sequence %d", appID, downstream.ClusterID, downstreamVersion.Sequence, version.Sequence)
				}
				if downstream.CurrentSequence == downstreamVersion.Sequence {
					newDownstreamSequence = version.Sequence
				}
				continue VERSIONS
			}

			log.Printf("Will delete downstream version: app=%s, cluster=%s, sequence=%d", appID, downstream.ClusterID, downstreamVersion.Sequence)
			err := deleteAppDownstreamVersions(tx, appID, downstream.ClusterID, downstreamVersion.Sequence)
			if err != nil {
				return errors.Wrapf(err, "failed to delete dowstream versions for app %s in cluster %s, sequence %d", appID, downstream.ClusterID, downstreamVersion.Sequence)
			}
			versionsRemoved = true
		}
	}

	if newDownstreamSequence == downstream.CurrentSequence {
		return nil
	}

	log.Printf("Will change current downstream %s sequence from %d to %d", downstream.ClusterID, downstream.CurrentSequence, newDownstreamSequence)
	if err := updateDownstreamSequence(tx, appID, downstream.ClusterID, newDownstreamSequence); err != nil {
		return errors.Wrapf(err, "failed to change dowstream %s sequence from %d to %d", downstream.ClusterID, downstream.CurrentSequence, newDownstreamSequence)
	}

	return nil
}

func getAppDownstreamVersions(tx *sql.Tx, appID string, clusterID string) ([]appDownstreamVersion, error) {
	query := `SELECT sequence, parent_sequence, created_at, version_label, status
	FROM app_downstream_version
	WHERE app_id = $1 AND cluster_id = $2
	ORDER BY sequence ASC`
	rows, err := tx.Query(query, appID, clusterID)
	if err != nil {
		return nil, errors.Wrap(err, "failed to query dowstream versions")
	}
	defer rows.Close()

	downstreamVersions := make([]appDownstreamVersion, 0)
	for rows.Next() {
		v := appDownstreamVersion{}
		if err := rows.Scan(&v.Sequence, &v.ParentSequence, &v.CreatedAt, &v.VersionLabel, &v.Status); err != nil {
			if err == sql.ErrNoRows {
				return downstreamVersions, nil
			}
			return nil, errors.Wrap(err, "failed to query dowstream versions")
		}
		downstreamVersions = append(downstreamVersions, v)
	}

	return downstreamVersions, nil
}

func getAppVersions(tx *sql.Tx, appID string) ([]appVersion, error) {
	query := `SELECT sequence, update_cursor, created_at, version_label
	FROM app_version
	WHERE app_id = $1
	ORDER BY sequence ASC`
	rows, err := tx.Query(query, appID)
	if err != nil {
		return nil, errors.Wrap(err, "failed to query app versions")
	}
	defer rows.Close()

	versions := make([]appVersion, 0)
	for rows.Next() {
		v := appVersion{}
		if err := rows.Scan(&v.Sequence, &v.UpdateCursor, &v.CreatedAt, &v.VersionLabel); err != nil {
			if err == sql.ErrNoRows {
				return versions, nil
			}
			return nil, errors.Wrap(err, "failed to query app versions")
		}
		versions = append(versions, v)
	}
	return versions, nil
}

func deleteAppDownstreamVersions(tx *sql.Tx, appID string, clusterID string, sequence int64) error {
	query := `DELETE FROM app_downstream_version WHERE app_id = $1 AND cluster_id = $2 AND sequence = $3`
	_, err := tx.Exec(query, appID, clusterID, sequence)
	if err != nil {
		return errors.Wrap(err, "failed to delete downstream version")
	}
	return nil
}

func setDownstreamVersionSequence(tx *sql.Tx, appID string, clusterID string, sequence int64, newSequence int64) error {
	query := `UPDATE app_downstream_version
	SET sequence = $4, parent_sequence = $4
	WHERE app_id = $1 AND cluster_id = $2 AND sequence = $3`
	_, err := tx.Exec(query, appID, clusterID, sequence, newSequence)
	if err != nil {
		return errors.Wrap(err, "failed to update downstream version")
	}
	return nil
}

func updateDownstreamSequence(tx *sql.Tx, appID string, clusterID string, newSequence int64) error {
	if newSequence == -1 {
		return nil
	}

	query := `UPDATE app_downstream SET current_sequence = $3 WHERE app_id = $1 AND cluster_id = $2`
	_, err := tx.Exec(query, appID, clusterID, newSequence)
	if err != nil {
		return errors.Wrap(err, "failed to update downstream sequence")
	}
	return nil
}
