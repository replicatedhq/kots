package kotsstore

import (
	"bytes"
	"database/sql"
	"io/ioutil"
	"path/filepath"

	"github.com/pkg/errors"
	kotsv1beta1 "github.com/replicatedhq/kots/kotskinds/apis/kots/v1beta1"
	"github.com/replicatedhq/kots/kotskinds/client/kotsclientset/scheme"
	gitopstypes "github.com/replicatedhq/kots/pkg/gitops/types"
	"github.com/replicatedhq/kots/pkg/logger"
	"github.com/replicatedhq/kots/pkg/persistence"
	rendertypes "github.com/replicatedhq/kots/pkg/render/types"
	serializer "k8s.io/apimachinery/pkg/runtime/serializer/json"
)

func (s *KOTSStore) GetLatestLicenseForApp(appID string) (*kotsv1beta1.License, error) {
	db := persistence.MustGetPGSession()
	query := `select license from app where id = $1`
	row := db.QueryRow(query, appID)

	var licenseStr sql.NullString
	if err := row.Scan(&licenseStr); err != nil {
		return nil, errors.Wrap(err, "failed to scan")
	}

	decode := scheme.Codecs.UniversalDeserializer().Decode
	obj, _, err := decode([]byte(licenseStr.String), nil, nil)
	if err != nil {
		return nil, errors.Wrap(err, "failed to decode license yaml")
	}
	license := obj.(*kotsv1beta1.License)
	return license, nil
}

func (s *KOTSStore) GetLicenseForAppVersion(appID string, sequence int64) (*kotsv1beta1.License, error) {
	db := persistence.MustGetPGSession()
	query := `select kots_license from app_version where app_id = $1 and sequence = $2`
	row := db.QueryRow(query, appID, sequence)

	var licenseStr sql.NullString
	if err := row.Scan(&licenseStr); err != nil {
		return nil, errors.Wrap(err, "failed to scan")
	}

	if licenseStr.Valid {
		decode := scheme.Codecs.UniversalDeserializer().Decode
		obj, _, err := decode([]byte(licenseStr.String), nil, nil)
		if err != nil {
			return nil, errors.Wrap(err, "failed to decode license yaml")
		}
		license := obj.(*kotsv1beta1.License)
		return license, nil
	}

	return nil, nil
}

func (s *KOTSStore) GetAllAppLicenses() ([]*kotsv1beta1.License, error) {
	db := persistence.MustGetPGSession()
	query := `select license from app`
	rows, err := db.Query(query)
	if err != nil {
		return nil, errors.Wrap(err, "failed to query")
	}
	defer rows.Close()

	var licenseStr sql.NullString
	licenses := []*kotsv1beta1.License{}
	for rows.Next() {
		if err := rows.Scan(&licenseStr); err != nil {
			return nil, errors.Wrap(err, "failed to scan")
		}
		if licenseStr.Valid {
			decode := scheme.Codecs.UniversalDeserializer().Decode
			obj, _, err := decode([]byte(licenseStr.String), nil, nil)
			if err != nil {
				return nil, errors.Wrap(err, "failed to decode license yaml")
			}
			license := obj.(*kotsv1beta1.License)
			licenses = append(licenses, license)
		}
	}

	return licenses, nil
}

func (s *KOTSStore) UpdateAppLicense(appID string, sequence int64, archiveDir string, newLicense *kotsv1beta1.License, originalLicenseData string, failOnVersionCreate bool, gitops gitopstypes.DownstreamGitOps, renderer rendertypes.Renderer) (int64, error) {
	db := persistence.MustGetPGSession()

	tx, err := db.Begin()
	if err != nil {
		return int64(0), errors.Wrap(err, "failed to begin")
	}
	defer tx.Rollback()

	ser := serializer.NewYAMLSerializer(serializer.DefaultMetaFactory, scheme.Scheme, scheme.Scheme)
	var b bytes.Buffer
	if err := ser.Encode(newLicense, &b); err != nil {
		return int64(0), errors.Wrap(err, "failed to encode license")
	}
	encodedLicense := b.Bytes()
	if err := ioutil.WriteFile(filepath.Join(archiveDir, "upstream", "userdata", "license.yaml"), encodedLicense, 0644); err != nil {
		return int64(0), errors.Wrap(err, "failed to write new license")
	}

	//  app has the original license data received from the server
	updateQuery := `update app set license=$1 where id = $2`
	_, err = tx.Exec(updateQuery, originalLicenseData, appID)
	if err != nil {
		return int64(0), errors.Wrapf(err, "update app %q license", appID)
	}

	newSeq, err := s.createNewVersionForLicenseChange(tx, appID, sequence, archiveDir, gitops, renderer)
	if err != nil {
		// ignore error here to prevent a failure to render the current version
		// preventing the end-user from updating the application
		if failOnVersionCreate {
			return int64(0), errors.Wrap(err, "failed to create new version")
		}
		logger.Errorf("Failed to create new version from license sync: %v", err)
	}

	if err := tx.Commit(); err != nil {
		return int64(0), errors.Wrap(err, "failed to commit transaction")
	}

	return newSeq, nil
}

func (s *KOTSStore) createNewVersionForLicenseChange(tx *sql.Tx, appID string, sequence int64, archiveDir string, gitops gitopstypes.DownstreamGitOps, renderer rendertypes.Renderer) (int64, error) {
	registrySettings, err := s.GetRegistryDetailsForApp(appID)
	if err != nil {
		return int64(0), errors.Wrap(err, "failed to get registry settings for app")
	}

	app, err := s.GetApp(appID)
	if err != nil {
		return int64(0), errors.Wrap(err, "failed to get app")
	}

	downstreams, err := s.ListDownstreamsForApp(appID)
	if err != nil {
		return int64(0), errors.Wrap(err, "failed to list downstreams")
	}

	if err := renderer.RenderDir(archiveDir, app, downstreams, registrySettings, true); err != nil {
		return int64(0), errors.Wrap(err, "failed to render new version")
	}

	newSequence, err := s.createAppVersion(tx, appID, &sequence, archiveDir, "License Change", false, gitops)
	if err != nil {
		return int64(0), errors.Wrap(err, "failed to create new version")
	}

	return newSequence, nil
}
