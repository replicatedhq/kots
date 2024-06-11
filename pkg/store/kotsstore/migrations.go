package kotsstore

import (
	"fmt"
	"io/ioutil"
	"os"

	"github.com/pkg/errors"
	"github.com/replicatedhq/kots/pkg/kotsutil"
	"github.com/replicatedhq/kots/pkg/logger"
	"github.com/replicatedhq/kots/pkg/persistence"
	"github.com/replicatedhq/kots/pkg/tasks"
	"github.com/rqlite/gorqlite"
)

func (s *KOTSStore) RunMigrations() {
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

	if err := s.migrateSkippedPreflights(); err != nil {
		logger.Error(errors.Wrap(err, "failed to migrate skipped preflights"))
	}

	// migrate data from rqlite
	if err := s.migrateSessionsFromRqlite(); err != nil {
		logger.Error(errors.Wrap(err, "failed to migrate sessions"))
	}
	if err := s.migrateSupportBundlesFromRqlite(); err != nil {
		logger.Error(errors.Wrap(err, "failed to migrate support bundles"))
	}
	if err := tasks.MigrateTasksFromRqlite(); err != nil {
		logger.Error(errors.Wrap(err, "failed to migrate tasks"))
	}
}

func (s *KOTSStore) migrateKotsAppSpec() error {
	db := persistence.MustGetDBSession()
	query := `select app_id, sequence from app_version where kots_app_spec is null or not kots_app_spec like '%apiVersion%'`

	rows, err := db.QueryOne(query)
	if err != nil {
		return fmt.Errorf("failed to query: %v: %v", err, rows.Err)
	}

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
			archiveDir, err := ioutil.TempDir("", "kotsadm")
			if err != nil {
				return errors.Wrap(err, "failed to create temp dir")
			}
			defer os.RemoveAll(archiveDir)

			err = s.GetAppVersionArchive(version.appID, version.sequence, archiveDir)
			if err != nil {
				return errors.Wrap(err, "failed to get app version archive")
			}

			kotsKinds, err := kotsutil.LoadKotsKinds(archiveDir)
			if err != nil {
				return errors.Wrap(err, "failed to load kots kinds from path")
			}

			spec, err := kotsKinds.Marshal("kots.io", "v1beta1", "Application")
			if err != nil {
				return errors.Wrap(err, "failed to marshal kots kinds")
			}

			query := `update app_version set kots_app_spec = ? where app_id = ? and sequence = ?`
			wr, err := db.WriteOneParameterized(gorqlite.ParameterizedStatement{
				Query:     query,
				Arguments: []interface{}{spec, version.appID, version.sequence},
			})
			if err != nil {
				return fmt.Errorf("failed to set kots_app_spec: %v: %v", err, wr.Err)
			}

			return nil
		}()
		if err != nil {
			logger.Error(errors.Wrapf(err, "failed to migrate kots_app_spec, app %v sequece %v", version.appID, version.sequence))
		}
	}

	return nil
}

func (s *KOTSStore) migrateKotsInstallationSpec() error {
	db := persistence.MustGetDBSession()
	query := `select app_id, sequence from app_version where kots_installation_spec is null or not kots_installation_spec like '%apiVersion%'`

	rows, err := db.QueryOne(query)
	if err != nil {
		return fmt.Errorf("failed to query: %v: %v", err, rows.Err)
	}

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
			archiveDir, err := ioutil.TempDir("", "kotsadm")
			if err != nil {
				return errors.Wrap(err, "failed to create temp dir")
			}
			defer os.RemoveAll(archiveDir)

			err = s.GetAppVersionArchive(version.appID, version.sequence, archiveDir)
			if err != nil {
				return errors.Wrap(err, "failed to get app version archive")
			}

			kotsKinds, err := kotsutil.LoadKotsKinds(archiveDir)
			if err != nil {
				return errors.Wrap(err, "failed to load kots kinds from path")
			}

			spec, err := kotsKinds.Marshal("kots.io", "v1beta1", "Installation")
			if err != nil {
				return errors.Wrap(err, "failed to marshal kots kinds")
			}

			query := `update app_version set kots_installation_spec = ? where app_id = ? and sequence = ?`
			wr, err := db.WriteOneParameterized(gorqlite.ParameterizedStatement{
				Query:     query,
				Arguments: []interface{}{spec, version.appID, version.sequence},
			})
			if err != nil {
				return fmt.Errorf("failed to set kots_installation_spec: %v: %v", err, wr.Err)
			}

			return nil
		}()
		if err != nil {
			logger.Error(errors.Wrapf(err, "failed to migrate kots_installation_spec app %v sequece %v", version.appID, version.sequence))
		}
	}

	return nil
}

func (s *KOTSStore) migrateSupportBundleSpec() error {
	db := persistence.MustGetDBSession()
	query := `select app_id, sequence from app_version where supportbundle_spec is null`

	rows, err := db.QueryOne(query)
	if err != nil {
		return fmt.Errorf("failed to query: %v: %v", err, rows.Err)
	}

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
			archiveDir, err := ioutil.TempDir("", "kotsadm")
			if err != nil {
				return errors.Wrap(err, "failed to create temp dir")
			}
			defer os.RemoveAll(archiveDir)

			err = s.GetAppVersionArchive(version.appID, version.sequence, archiveDir)
			if err != nil {
				return errors.Wrap(err, "failed to get app version archive")
			}

			kotsKinds, err := kotsutil.LoadKotsKinds(archiveDir)
			if err != nil {
				return errors.Wrap(err, "failed to load kots kinds from path")
			}

			spec, err := kotsKinds.Marshal("troubleshoot.sh", "v1beta2", "SupportBundle")
			if err != nil {
				return errors.Wrap(err, "failed to marshal kots kinds")
			}

			query := `update app_version set supportbundle_spec = ? where app_id = ? and sequence = ?`
			wr, err := db.WriteOneParameterized(gorqlite.ParameterizedStatement{
				Query:     query,
				Arguments: []interface{}{spec, version.appID, version.sequence},
			})
			if err != nil {
				return fmt.Errorf("failed to set supportbundle_spec: %v: %v", err, wr.Err)
			}

			return nil
		}()
		if err != nil {
			logger.Error(errors.Wrapf(err, "failed to migrate supportbundle_spec app %v sequece %v", version.appID, version.sequence))
		}
	}

	return nil
}

func (s *KOTSStore) migratePreflightSpec() error {
	db := persistence.MustGetDBSession()
	query := `select app_id, sequence from app_version where preflight_spec is null`

	rows, err := db.QueryOne(query)
	if err != nil {
		return fmt.Errorf("failed to query: %v: %v", err, rows.Err)
	}

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
			archiveDir, err := ioutil.TempDir("", "kotsadm")
			if err != nil {
				return errors.Wrap(err, "failed to create temp dir")
			}
			defer os.RemoveAll(archiveDir)

			err = s.GetAppVersionArchive(version.appID, version.sequence, archiveDir)
			if err != nil {
				return errors.Wrap(err, "failed to get app version archive")
			}

			kotsKinds, err := kotsutil.LoadKotsKinds(archiveDir)
			if err != nil {
				return errors.Wrap(err, "failed to load kots kinds from path")
			}

			spec, err := kotsKinds.Marshal("troubleshoot.sh", "v1beta2", "Preflight")
			if err != nil {
				return errors.Wrap(err, "failed to marshal kots kinds")
			}

			query := `update app_version set preflight_spec = ? where app_id = ? and sequence = ?`
			wr, err := db.WriteOneParameterized(gorqlite.ParameterizedStatement{
				Query:     query,
				Arguments: []interface{}{spec, version.appID, version.sequence},
			})
			if err != nil {
				return fmt.Errorf("failed to set preflight_spec: %v: %v", err, wr.Err)
			}

			return nil
		}()
		if err != nil {
			logger.Error(errors.Wrapf(err, "failed to migrate preflight_spec app %v sequece %v", version.appID, version.sequence))
		}
	}

	return nil
}

func (s *KOTSStore) migrateAnalyzerSpec() error {
	db := persistence.MustGetDBSession()
	query := `select app_id, sequence from app_version where analyzer_spec is null`

	rows, err := db.QueryOne(query)
	if err != nil {
		return fmt.Errorf("failed to query: %v: %v", err, rows.Err)
	}

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
			archiveDir, err := ioutil.TempDir("", "kotsadm")
			if err != nil {
				return errors.Wrap(err, "failed to create temp dir")
			}
			defer os.RemoveAll(archiveDir)

			err = s.GetAppVersionArchive(version.appID, version.sequence, archiveDir)
			if err != nil {
				return errors.Wrap(err, "failed to get app version archive")
			}

			kotsKinds, err := kotsutil.LoadKotsKinds(archiveDir)
			if err != nil {
				return errors.Wrap(err, "failed to load kots kinds from path")
			}

			spec, err := kotsKinds.Marshal("troubleshoot.sh", "v1beta2", "Analyzer")
			if err != nil {
				return errors.Wrap(err, "failed to marshal kots kinds")
			}

			query := `update app_version set analyzer_spec = ? where app_id = ? and sequence = ?`
			wr, err := db.WriteOneParameterized(gorqlite.ParameterizedStatement{
				Query:     query,
				Arguments: []interface{}{spec, version.appID, version.sequence},
			})
			if err != nil {
				return fmt.Errorf("failed to set analyzer_spec: %v: %v", err, wr.Err)
			}

			return nil
		}()
		if err != nil {
			logger.Error(errors.Wrapf(err, "failed to migrate analyzer_spec app %v sequece %v", version.appID, version.sequence))
		}
	}

	return nil
}

func (s *KOTSStore) migrateAppSpec() error {
	db := persistence.MustGetDBSession()
	query := `select app_id, sequence from app_version where app_spec is null`

	rows, err := db.QueryOne(query)
	if err != nil {
		return fmt.Errorf("failed to query: %v: %v", err, rows.Err)
	}

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
			archiveDir, err := ioutil.TempDir("", "kotsadm")
			if err != nil {
				return errors.Wrap(err, "failed to create temp dir")
			}
			defer os.RemoveAll(archiveDir)

			err = s.GetAppVersionArchive(version.appID, version.sequence, archiveDir)
			if err != nil {
				return errors.Wrap(err, "failed to get app version archive")
			}

			kotsKinds, err := kotsutil.LoadKotsKinds(archiveDir)
			if err != nil {
				return errors.Wrap(err, "failed to load kots kinds from path")
			}

			spec, err := kotsKinds.Marshal("app.k8s.io", "v1beta1", "Application")
			if err != nil {
				return errors.Wrap(err, "failed to marshal kots kinds")
			}

			query := `update app_version set app_spec = ? where app_id = ? and sequence = ?`
			wr, err := db.WriteOneParameterized(gorqlite.ParameterizedStatement{
				Query:     query,
				Arguments: []interface{}{spec, version.appID, version.sequence},
			})
			if err != nil {
				return fmt.Errorf("failed to set app_spec: %v: %v", err, wr.Err)
			}

			return nil
		}()
		if err != nil {
			logger.Error(errors.Wrapf(err, "failed to migrate app_spec app %v sequece %v", version.appID, version.sequence))
		}
	}

	return nil
}

func (s *KOTSStore) migrateSkippedPreflights() error {
	db := persistence.MustGetDBSession()

	query := `update app_downstream_version set preflight_skipped = true where preflight_result_created_at is null`
	wr, err := db.WriteOne(query)
	if err != nil {
		return fmt.Errorf("failed to set preflight_skipped: %v: %v", err, wr.Err)
	}

	return nil
}
