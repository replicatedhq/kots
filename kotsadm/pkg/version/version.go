package version

import (
	"encoding/json"
	"time"

	"github.com/pkg/errors"
	"github.com/replicatedhq/kots/kotsadm/pkg/app"
	"github.com/replicatedhq/kots/kotsadm/pkg/config"
	"github.com/replicatedhq/kots/kotsadm/pkg/downstream"
	"github.com/replicatedhq/kots/kotsadm/pkg/gitops"
	"github.com/replicatedhq/kots/kotsadm/pkg/kotsutil"
	"github.com/replicatedhq/kots/kotsadm/pkg/persistence"
	"github.com/replicatedhq/kots/kotsadm/pkg/version/types"
)

// GetNextAppSequence determines next available sequence for this app
// we shouldn't assume that a.CurrentSequence is accurate. Returns 0 if currentSequence is nil
func GetNextAppSequence(appID string, currentSequence *int64) (int64, error) {
	newSequence := 0
	if currentSequence != nil {
		db := persistence.MustGetPGSession()
		row := db.QueryRow(`select max(sequence) from app_version where app_id = $1`, appID)
		if err := row.Scan(&newSequence); err != nil {
			return 0, errors.Wrap(err, "failed to find current max sequence in row")
		}
		newSequence++
	}
	return int64(newSequence), nil
}

// CreateFirstVersion works much likst CreateVersion except that it assumes version 0
// and never attempts to calculate a diff, or look at previous versions
func CreateFirstVersion(appID string, filesInDir string, source string) (int64, error) {
	return createVersion(appID, filesInDir, source, nil)
}

// CreateVersion creates a new version of the app in the database, but the caller
// is responsible for uploading the archive to s3
func CreateVersion(appID string, filesInDir string, source string, currentSequence int64) (int64, error) {
	return createVersion(appID, filesInDir, source, &currentSequence)
}

// this is the common, internal function to create an app version, used in both
// new and updates to apps
func createVersion(appID string, filesInDir string, source string, currentSequence *int64) (int64, error) {
	kotsKinds, err := kotsutil.LoadKotsKindsFromPath(filesInDir)
	if err != nil {
		return int64(0), errors.Wrap(err, "failed to read kots kinds")
	}

	supportBundleSpec, err := kotsKinds.Marshal("troubleshoot.replicated.com", "v1beta1", "Collector")
	if err != nil {
		return int64(0), errors.Wrap(err, "failed to marshal support bundle spec")
	}
	analyzersSpec, err := kotsKinds.Marshal("troubleshoot.replicated.com", "v1beta1", "Analyzer")
	if err != nil {
		return int64(0), errors.Wrap(err, "failed to marshal analyzer spec")
	}
	preflightSpec, err := kotsKinds.Marshal("troubleshoot.replicated.com", "v1beta1", "Preflight")
	if err != nil {
		return int64(0), errors.Wrap(err, "failed to marshal preflight spec")
	}

	appSpec, err := kotsKinds.Marshal("app.k8s.io", "v1beta1", "Application")
	if err != nil {
		return int64(0), errors.Wrap(err, "failed to marshal app spec")
	}
	kotsAppSpec, err := kotsKinds.Marshal("kots.io", "v1beta1", "Application")
	if err != nil {
		return int64(0), errors.Wrap(err, "failed to marshal kots app spec")
	}
	kotsInstallationSpec, err := kotsKinds.Marshal("kots.io", "v1beta1", "Installation")
	if err != nil {
		return int64(0), errors.Wrap(err, "failed to marshal kots installation spec")
	}
	backupSpec, err := kotsKinds.Marshal("velero.io", "v1", "Backup")
	if err != nil {
		return int64(0), errors.Wrap(err, "failed to marshal backup spec")
	}

	licenseSpec, err := kotsKinds.Marshal("kots.io", "v1beta1", "License")
	if err != nil {
		return int64(0), errors.Wrap(err, "failed to marshal license spec")
	}
	configSpec, err := kotsKinds.Marshal("kots.io", "v1beta1", "Config")
	if err != nil {
		return int64(0), errors.Wrap(err, "failed to marshal config spec")
	}
	configValuesSpec, err := kotsKinds.Marshal("kots.io", "v1beta1", "ConfigValues")
	if err != nil {
		return int64(0), errors.Wrap(err, "failed to marshal configvalues spec")
	}

	db := persistence.MustGetPGSession()

	tx, err := db.Begin()
	if err != nil {
		return 0, errors.Wrap(err, "failed to begin")
	}
	defer tx.Rollback()

	n, err := GetNextAppSequence(appID, currentSequence)
	if err != nil {
		return 0, errors.Wrap(err, "failed to get new app sequence")
	}
	newSequence := int(n)

	query := `insert into app_version (app_id, sequence, created_at, version_label, release_notes, update_cursor, channel_name, encryption_key,
supportbundle_spec, analyzer_spec, preflight_spec, app_spec, kots_app_spec, kots_installation_spec, kots_license, config_spec, config_values, backup_spec)
values ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18)
ON CONFLICT(app_id, sequence) DO UPDATE SET
created_at = EXCLUDED.created_at,
version_label = EXCLUDED.version_label,
release_notes = EXCLUDED.release_notes,
update_cursor = EXCLUDED.update_cursor,
channel_name = EXCLUDED.channel_name,
encryption_key = EXCLUDED.encryption_key,
supportbundle_spec = EXCLUDED.supportbundle_spec,
analyzer_spec = EXCLUDED.analyzer_spec,
preflight_spec = EXCLUDED.preflight_spec,
app_spec = EXCLUDED.app_spec,
kots_app_spec = EXCLUDED.kots_app_spec,
kots_installation_spec = EXCLUDED.kots_installation_spec,
kots_license = EXCLUDED.kots_license,
config_spec = EXCLUDED.config_spec,
config_values = EXCLUDED.config_values,
backup_spec = EXCLUDED.backup_spec`
	_, err = tx.Exec(query, appID, newSequence, time.Now(),
		kotsKinds.Installation.Spec.VersionLabel,
		kotsKinds.Installation.Spec.ReleaseNotes,
		kotsKinds.Installation.Spec.UpdateCursor,
		kotsKinds.Installation.Spec.ChannelName,
		kotsKinds.Installation.Spec.EncryptionKey,
		supportBundleSpec,
		analyzersSpec,
		preflightSpec,
		appSpec,
		kotsAppSpec,
		kotsInstallationSpec,
		licenseSpec,
		configSpec,
		configValuesSpec,
		backupSpec)
	if err != nil {
		return int64(0), errors.Wrap(err, "failed to insert app version")
	}

	appName := kotsKinds.KotsApplication.Spec.Title
	if appName == "" {
		a, err := app.Get(appID)
		if err != nil {
			return int64(0), errors.Wrap(err, "failed to get app")
		}

		appName = a.Name
	}

	appIcon := kotsKinds.KotsApplication.Spec.Icon

	query = "update app set current_sequence = $1, name = $2, icon_uri = $3 where id = $4"
	_, err = tx.Exec(query, int64(newSequence), appName, appIcon, appID)
	if err != nil {
		return int64(0), errors.Wrap(err, "failed to update app")
	}

	previousArchiveDir := ""
	if currentSequence != nil {
		// Get the previous archive, we need this to calculate the diff
		previousDir, err := GetAppVersionArchive(appID, *currentSequence)
		if err != nil {
			return int64(0), errors.Wrap(err, "failed to get previous archive")
		}

		previousArchiveDir = previousDir
	}

	downstreams, err := downstream.ListDownstreamsForApp(appID)
	if err != nil {
		return int64(0), errors.Wrap(err, "failed to list downstreams")
	}

	for _, d := range downstreams {
		downstreamStatus := "pending"
		if currentSequence == nil && kotsKinds.Config != nil {
			configSpec, err := kotsKinds.Marshal("kots.io", "v1beta1", "Config")
			if err != nil {
				return int64(0), errors.Wrap(err, "failed to render config")
			}

			configValues := ""
			if kotsKinds.ConfigValues != nil {
				configValuesSpec, err := kotsKinds.Marshal("kots.io", "v1beta1", "ConfigValues")
				if err != nil {
					return int64(0), errors.Wrap(err, "failed to render config values")
				}

				configValues = string(configValuesSpec)
			}

			license := ""
			if kotsKinds.License != nil {
				licenseSpec, err := kotsKinds.Marshal("kots.io", "v1beta1", "License")
				if err != nil {
					return int64(0), errors.Wrap(err, "failed to render license")
				}

				license = string(licenseSpec)
			}
			needsConfig, err := config.NeedsConfiguration(string(configSpec), configValues, license)
			if err != nil {
				return int64(0), errors.Wrap(err, "failed to check if app needs configuration")
			}

			if needsConfig || configValues == "" {
				downstreamStatus = "pending_config"
			} else if kotsKinds.Preflight != nil {
				downstreamStatus = "pending_preflight"
			}
		} else if kotsKinds.Preflight != nil {
			downstreamStatus = "pending_preflight"
		}

		diffSummary := ""
		if currentSequence != nil {
			// diff this release from the last release
			diff, err := downstream.DiffAppVersionsForDownstream(d.Name, filesInDir, previousArchiveDir, kotsKinds.KustomizeVersion())
			if err != nil {
				return int64(0), errors.Wrap(err, "failed to diff")
			}
			b, err := json.Marshal(diff)
			if err != nil {
				return int64(0), errors.Wrap(err, "failed to marshal diff")
			}
			diffSummary = string(b)

			// check if version needs additional configuration
			t, err := config.NeedsConfiguration(configSpec, configValuesSpec, licenseSpec)
			if err != nil {
				return int64(0), errors.Wrap(err, "failed to check if version needs configuration")
			}
			if t {
				downstreamStatus = "pending_config"
			}
		}

		commitURL := ""

		downstreamGitOps, err := gitops.GetDownstreamGitOps(appID, d.ClusterID)
		if err != nil {
			return int64(0), errors.Wrap(err, "failed to get downstream gitops")
		}
		if downstreamGitOps != nil {
			a, err := app.Get(appID)
			if err != nil {
				return int64(0), errors.Wrap(err, "failed to get app")
			}
			createdCommitURL, err := gitops.CreateGitOpsCommit(downstreamGitOps, a.Slug, a.Name, int(newSequence), filesInDir, d.Name)
			if err != nil {
				return int64(0), errors.Wrap(err, "failed to create gitops commit")
			}

			commitURL = createdCommitURL
		}

		query = `insert into app_downstream_version (app_id, cluster_id, sequence, parent_sequence, created_at, version_label, status, source, diff_summary, git_commit_url, git_deployable) values ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)`
		_, err = tx.Exec(query, appID, d.ClusterID, newSequence, newSequence, time.Now(),
			kotsKinds.Installation.Spec.VersionLabel, downstreamStatus, source,
			diffSummary, commitURL, commitURL != "")
		if err != nil {
			return int64(0), errors.Wrap(err, "failed to create downstream version")
		}
	}

	if err = tx.Commit(); err != nil {
		return int64(0), errors.Wrap(err, "failed to commit")
	}

	return int64(newSequence), nil
}

// return the list of versions available for an app
func GetVersions(appID string) ([]types.AppVersion, error) {
	db := persistence.MustGetPGSession()
	query := `select sequence, update_cursor, version_label from app_version where app_id = $1 order by update_cursor asc, sequence asc`
	rows, err := db.Query(query, appID)
	if err != nil {
		return nil, errors.Wrap(err, "query app_version table")
	}

	versions := []types.AppVersion{}

	for rows.Next() {
		rowVersion := types.AppVersion{}
		err = rows.Scan(&rowVersion.Sequence, &rowVersion.UpdateCursor, &rowVersion.VersionLabel)
		if err != nil {
			return nil, errors.Wrap(err, "scan row from app_version table")
		}
		versions = append(versions, rowVersion)
	}

	return versions, nil
}

// DeployVersion deploys the version for the given sequence
func DeployVersion(appID string, sequence int64) error {
	db := persistence.MustGetPGSession()

	tx, err := db.Begin()
	if err != nil {
		return errors.Wrap(err, "failed to begin")
	}
	defer tx.Rollback()

	query := `update app_downstream set current_sequence = $1 where app_id = $2`
	_, err = tx.Exec(query, sequence, appID)
	if err != nil {
		return errors.Wrap(err, "failed to update app downstream current sequence")
	}

	query = `update app_downstream_version set status = 'deployed', applied_at = $3 where sequence = $1 and app_id = $2`
	_, err = tx.Exec(query, sequence, appID, time.Now())
	if err != nil {
		return errors.Wrap(err, "failed to update app downstream version status")
	}

	return nil
}
