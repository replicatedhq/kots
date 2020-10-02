package s3pg

import (
	"fmt"
	"os"

	"github.com/pkg/errors"
	"github.com/replicatedhq/kots/kotsadm/pkg/logger"
	"github.com/replicatedhq/kots/kotsadm/pkg/persistence"
	"github.com/replicatedhq/kots/pkg/kotsutil"
)

func (s S3PGStore) RunMigrations() {
	if err := s.migrateKotsAppSpec(); err != nil {
		logger.Error(errors.Wrap(err, "failed to migrate kots_app_spec"))
	}

	if err := s.migrateKotsInstallationSpec(); err != nil {
		logger.Error(errors.Wrap(err, "failed to migrate kots_installation_spec"))
	}

	if err := s.migrateSupportBundleSpec(); err != nil {
		logger.Error(errors.Wrap(err, "failed to migrate supportbundle_spec"))
	}

	if err := s.migratePreflightSpec(); err != nil {
		logger.Error(errors.Wrap(err, "failed to migrate preflight_spec"))
	}

	if err := s.migrateAnalyzerSpec(); err != nil {
		logger.Error(errors.Wrap(err, "failed to migrate analyzer_spec"))
	}

	if err := s.migrateAppSpec(); err != nil {
		logger.Error(errors.Wrap(err, "failed to migrate app_spec"))
	}
}

func (s S3PGStore) migrateKotsAppSpec() error {
	db := persistence.MustGetPGSession()
	query := `select app_id, sequence from app_version where kots_app_spec is null or not kots_app_spec like '%apiVersion%'`

	rows, err := db.Query(query)
	if err != nil {
		return errors.Wrap(err, "failed to query db")
	}
	defer rows.Close()

	type versionType struct {
		appID    string
		sequence int64
	}

	versions := make([]versionType, 0)
	for rows.Next() {
		var appID string
		var sequence int64

		if err := rows.Scan(&appID, &sequence); err != nil {
			return errors.Wrap(err, "failed to scan")
		}

		versions = append(versions, versionType{
			appID:    appID,
			sequence: sequence,
		})
	}

	for _, version := range versions {
		logger.Info(fmt.Sprintf("Migrating kots_app_spec for app %s sequence %d", version.appID, version.sequence))
		err := func() error {
			archiveDir, err := s.GetAppVersionArchive(version.appID, version.sequence)
			if err != nil {
				return errors.Wrap(err, "failed to get app version archive")
			}
			defer os.RemoveAll(archiveDir)

			kotsKinds, err := kotsutil.LoadKotsKindsFromPath(archiveDir)
			if err != nil {
				return errors.Wrap(err, "failed to load kots kinds from path")
			}

			spec, err := kotsKinds.Marshal("kots.io", "v1beta1", "Application")
			if err != nil {
				return errors.Wrap(err, "failed to marshal kots kinds")
			}

			query := `update app_version set kots_app_spec = $1 where app_id = $2 and sequence = $3`
			_, err = db.Exec(query, spec, version.appID, version.sequence)
			if err != nil {
				return errors.Wrap(err, "failed to set kots_app_spec")
			}

			return nil
		}()
		if err != nil {
			logger.Error(errors.Wrapf(err, "failed to migrate kots_app_spec, app %v sequece %v", version.appID, version.sequence))
		}
	}

	return nil
}

func (s S3PGStore) migrateKotsInstallationSpec() error {
	db := persistence.MustGetPGSession()
	query := `select app_id, sequence from app_version where kots_installation_spec is null or not kots_installation_spec like '%apiVersion%'`

	rows, err := db.Query(query)
	if err != nil {
		return errors.Wrap(err, "failed to query db")
	}
	defer rows.Close()

	type versionType struct {
		appID    string
		sequence int64
	}

	versions := make([]versionType, 0)
	for rows.Next() {
		var appID string
		var sequence int64

		if err := rows.Scan(&appID, &sequence); err != nil {
			return errors.Wrap(err, "failed to scan")
		}

		versions = append(versions, versionType{
			appID:    appID,
			sequence: sequence,
		})
	}

	for _, version := range versions {
		logger.Info(fmt.Sprintf("Migrating kots_installation_spec for app %s sequence %d", version.appID, version.sequence))
		err := func() error {
			archiveDir, err := s.GetAppVersionArchive(version.appID, version.sequence)
			if err != nil {
				return errors.Wrap(err, "failed to get app version archive")
			}
			defer os.RemoveAll(archiveDir)

			kotsKinds, err := kotsutil.LoadKotsKindsFromPath(archiveDir)
			if err != nil {
				return errors.Wrap(err, "failed to load kots kinds from path")
			}

			spec, err := kotsKinds.Marshal("kots.io", "v1beta1", "Installation")
			if err != nil {
				return errors.Wrap(err, "failed to marshal kots kinds")
			}

			query := `update app_version set kots_installation_spec = $1 where app_id = $2 and sequence = $3`
			_, err = db.Exec(query, spec, version.appID, version.sequence)
			if err != nil {
				return errors.Wrap(err, "failed to set kots_installation_spec")
			}

			return nil
		}()
		if err != nil {
			logger.Error(errors.Wrapf(err, "failed to migrate kots_installation_spec app %v sequece %v", version.appID, version.sequence))
		}
	}

	return nil
}

func (s S3PGStore) migrateSupportBundleSpec() error {
	db := persistence.MustGetPGSession()
	query := `select app_id, sequence from app_version where supportbundle_spec is null or not supportbundle_spec like '%apiVersion%'`

	rows, err := db.Query(query)
	if err != nil {
		return errors.Wrap(err, "failed to query db")
	}
	defer rows.Close()

	type versionType struct {
		appID    string
		sequence int64
	}

	versions := make([]versionType, 0)
	for rows.Next() {
		var appID string
		var sequence int64

		if err := rows.Scan(&appID, &sequence); err != nil {
			return errors.Wrap(err, "failed to scan")
		}

		versions = append(versions, versionType{
			appID:    appID,
			sequence: sequence,
		})
	}

	for _, version := range versions {
		logger.Info(fmt.Sprintf("Migrating kots_installation_spec for app %s sequence %d", version.appID, version.sequence))
		err := func() error {
			archiveDir, err := s.GetAppVersionArchive(version.appID, version.sequence)
			if err != nil {
				return errors.Wrap(err, "failed to get app version archive")
			}
			defer os.RemoveAll(archiveDir)

			kotsKinds, err := kotsutil.LoadKotsKindsFromPath(archiveDir)
			if err != nil {
				return errors.Wrap(err, "failed to load kots kinds from path")
			}

			spec, err := kotsKinds.Marshal("troubleshoot.sh", "v1beta2", "SupportBundle")
			if err != nil {
				return errors.Wrap(err, "failed to marshal kots kinds")
			}

			query := `update app_version set supportbundle_spec = $1 where app_id = $2 and sequence = $3`
			_, err = db.Exec(query, spec, version.appID, version.sequence)
			if err != nil {
				return errors.Wrap(err, "failed to set supportbundle_spec")
			}

			return nil
		}()
		if err != nil {
			logger.Error(errors.Wrapf(err, "failed to migrate supportbundle_spec app %v sequece %v", version.appID, version.sequence))
		}
	}

	return nil
}

func (s S3PGStore) migratePreflightSpec() error {
	db := persistence.MustGetPGSession()
	query := `select app_id, sequence from app_version where preflight_spec is null or not preflight_spec like '%apiVersion%'`

	rows, err := db.Query(query)
	if err != nil {
		return errors.Wrap(err, "failed to query db")
	}
	defer rows.Close()

	type versionType struct {
		appID    string
		sequence int64
	}

	versions := make([]versionType, 0)
	for rows.Next() {
		var appID string
		var sequence int64

		if err := rows.Scan(&appID, &sequence); err != nil {
			return errors.Wrap(err, "failed to scan")
		}

		versions = append(versions, versionType{
			appID:    appID,
			sequence: sequence,
		})
	}

	for _, version := range versions {
		logger.Info(fmt.Sprintf("Migrating preflight_spec for app %s sequence %d", version.appID, version.sequence))
		err := func() error {
			archiveDir, err := s.GetAppVersionArchive(version.appID, version.sequence)
			if err != nil {
				return errors.Wrap(err, "failed to get app version archive")
			}
			defer os.RemoveAll(archiveDir)

			kotsKinds, err := kotsutil.LoadKotsKindsFromPath(archiveDir)
			if err != nil {
				return errors.Wrap(err, "failed to load kots kinds from path")
			}

			spec, err := kotsKinds.Marshal("troubleshoot.sh", "v1beta2", "Preflight")
			if err != nil {
				return errors.Wrap(err, "failed to marshal kots kinds")
			}

			query := `update app_version set preflight_spec = $1 where app_id = $2 and sequence = $3`
			_, err = db.Exec(query, spec, version.appID, version.sequence)
			if err != nil {
				return errors.Wrap(err, "failed to set preflight_spec")
			}

			return nil
		}()
		if err != nil {
			logger.Error(errors.Wrapf(err, "failed to migrate preflight_spec app %v sequece %v", version.appID, version.sequence))
		}
	}

	return nil
}

func (s S3PGStore) migrateAnalyzerSpec() error {
	db := persistence.MustGetPGSession()
	query := `select app_id, sequence from app_version where analyzer_spec is null or not analyzer_spec like '%apiVersion%'`

	rows, err := db.Query(query)
	if err != nil {
		return errors.Wrap(err, "failed to query db")
	}
	defer rows.Close()

	type versionType struct {
		appID    string
		sequence int64
	}

	versions := make([]versionType, 0)
	for rows.Next() {
		var appID string
		var sequence int64

		if err := rows.Scan(&appID, &sequence); err != nil {
			return errors.Wrap(err, "failed to scan")
		}

		versions = append(versions, versionType{
			appID:    appID,
			sequence: sequence,
		})
	}

	for _, version := range versions {
		logger.Info(fmt.Sprintf("Migrating analyzer_spec for app %s sequence %d", version.appID, version.sequence))
		err := func() error {
			archiveDir, err := s.GetAppVersionArchive(version.appID, version.sequence)
			if err != nil {
				return errors.Wrap(err, "failed to get app version archive")
			}
			defer os.RemoveAll(archiveDir)

			kotsKinds, err := kotsutil.LoadKotsKindsFromPath(archiveDir)
			if err != nil {
				return errors.Wrap(err, "failed to load kots kinds from path")
			}

			spec, err := kotsKinds.Marshal("troubleshoot.sh", "v1beta2", "Analyzer")
			if err != nil {
				return errors.Wrap(err, "failed to marshal kots kinds")
			}

			query := `update app_version set analyzer_spec = $1 where app_id = $2 and sequence = $3`
			_, err = db.Exec(query, spec, version.appID, version.sequence)
			if err != nil {
				return errors.Wrap(err, "failed to set analyzer_spec")
			}

			return nil
		}()
		if err != nil {
			logger.Error(errors.Wrapf(err, "failed to migrate analyzer_spec app %v sequece %v", version.appID, version.sequence))
		}
	}

	return nil
}

func (s S3PGStore) migrateAppSpec() error {
	db := persistence.MustGetPGSession()
	query := `select app_id, sequence from app_version where app_spec is null or not app_spec like '%apiVersion%'`

	rows, err := db.Query(query)
	if err != nil {
		return errors.Wrap(err, "failed to query db")
	}
	defer rows.Close()

	type versionType struct {
		appID    string
		sequence int64
	}

	versions := make([]versionType, 0)
	for rows.Next() {
		var appID string
		var sequence int64

		if err := rows.Scan(&appID, &sequence); err != nil {
			return errors.Wrap(err, "failed to scan")
		}

		versions = append(versions, versionType{
			appID:    appID,
			sequence: sequence,
		})
	}

	for _, version := range versions {
		logger.Info(fmt.Sprintf("Migrating app_spec for app %s sequence %d", version.appID, version.sequence))
		err := func() error {
			archiveDir, err := s.GetAppVersionArchive(version.appID, version.sequence)
			if err != nil {
				return errors.Wrap(err, "failed to get app version archive")
			}
			defer os.RemoveAll(archiveDir)

			kotsKinds, err := kotsutil.LoadKotsKindsFromPath(archiveDir)
			if err != nil {
				return errors.Wrap(err, "failed to load kots kinds from path")
			}

			spec, err := kotsKinds.Marshal("app.k8s.io", "v1beta1", "Application")
			if err != nil {
				return errors.Wrap(err, "failed to marshal kots kinds")
			}

			query := `update app_version set app_spec = $1 where app_id = $2 and sequence = $3`
			_, err = db.Exec(query, spec, version.appID, version.sequence)
			if err != nil {
				return errors.Wrap(err, "failed to set app_spec")
			}

			return nil
		}()
		if err != nil {
			logger.Error(errors.Wrapf(err, "failed to migrate app_spec app %v sequece %v", version.appID, version.sequence))
		}
	}

	return nil
}
