package kotsstore

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"path/filepath"
	"time"

	"github.com/pkg/errors"
	reportingtypes "github.com/replicatedhq/kots/pkg/api/reporting/types"
	"github.com/replicatedhq/kots/pkg/logger"
	"github.com/replicatedhq/kots/pkg/persistence"
	rendertypes "github.com/replicatedhq/kots/pkg/render/types"
	kotsv1beta1 "github.com/replicatedhq/kotskinds/apis/kots/v1beta1"
	"github.com/replicatedhq/kotskinds/client/kotsclientset/scheme"
	"github.com/rqlite/gorqlite"
	serializer "k8s.io/apimachinery/pkg/runtime/serializer/json"
)

func (s *KOTSStore) GetLatestLicenseForApp(appID string) (*kotsv1beta1.License, error) {
	db := persistence.MustGetDBSession()
	query := `select license from app where id = ?`
	rows, err := db.QueryOneParameterized(gorqlite.ParameterizedStatement{
		Query:     query,
		Arguments: []interface{}{appID},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to query: %v: %v", err, rows.Err)
	}
	if !rows.Next() {
		return nil, ErrNotFound
	}

	var licenseStr gorqlite.NullString
	if err := rows.Scan(&licenseStr); err != nil {
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
	db := persistence.MustGetDBSession()
	query := `select kots_license from app_version where app_id = ? and sequence = ?`
	rows, err := db.QueryOneParameterized(gorqlite.ParameterizedStatement{
		Query:     query,
		Arguments: []interface{}{appID, sequence},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to query: %v: %v", err, rows.Err)
	}
	if !rows.Next() {
		return nil, ErrNotFound
	}

	var licenseStr gorqlite.NullString
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
		return license, nil
	}

	return nil, nil
}

func (s *KOTSStore) GetAllAppLicenses() ([]*kotsv1beta1.License, error) {
	db := persistence.MustGetDBSession()
	query := `select license from app`
	rows, err := db.QueryOne(query)
	if err != nil {
		return nil, fmt.Errorf("failed to query: %v: %v", err, rows.Err)
	}

	var licenseStr gorqlite.NullString
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

func (s *KOTSStore) UpdateAppLicense(appID string, baseSequence int64, archiveDir string, newLicense *kotsv1beta1.License, originalLicenseData string, channelChanged bool, failOnVersionCreate bool, renderer rendertypes.Renderer, reportingInfo *reportingtypes.ReportingInfo) (int64, error) {
	db := persistence.MustGetDBSession()

	statements := []gorqlite.ParameterizedStatement{}

	ser := serializer.NewYAMLSerializer(serializer.DefaultMetaFactory, scheme.Scheme, scheme.Scheme)
	var b bytes.Buffer
	if err := ser.Encode(newLicense, &b); err != nil {
		return int64(0), errors.Wrap(err, "failed to encode license")
	}
	encodedLicense := b.Bytes()
	if err := ioutil.WriteFile(filepath.Join(archiveDir, "upstream", "userdata", "license.yaml"), encodedLicense, 0644); err != nil {
		return int64(0), errors.Wrap(err, "failed to write new license")
	}

	// If the license channels array has more than one entry, then the license is a true multi-channel license,
	// and we should skip updating selected_channel_id in the app table. If there's only a single entry,
	// we should update the selected_channel_id in the app table to ensure it stays consistent across channel
	// changes. This is a temporary solution until channel changes on true multi-channel licenses are supported.
	if len(newLicense.Spec.Channels) > 1 {
		//  app has the original license data received from the server
		statements = append(statements, gorqlite.ParameterizedStatement{
			Query:     `update app set license = ?, last_license_sync = ?, channel_changed = ? where id = ?`,
			Arguments: []interface{}{encodedLicense, time.Now().Unix(), channelChanged, appID},
		})
	} else {
		//  app has the original license data received from the server
		statements = append(statements, gorqlite.ParameterizedStatement{
			Query:     `update app set license = ?, last_license_sync = ?, channel_changed = ?, selected_channel_id = ? where id = ?`,
			Arguments: []interface{}{originalLicenseData, time.Now().Unix(), channelChanged, appID, newLicense.Spec.ChannelID},
		})
	}

	appVersionStatements, newSeq, err := s.createNewVersionForLicenseChangeStatements(appID, baseSequence, archiveDir, renderer, reportingInfo)
	if err != nil {
		// ignore error here to prevent a failure to render the current version
		// preventing the end-user from updating the application
		if failOnVersionCreate {
			return int64(0), errors.Wrap(err, "failed to construct app version statements for license sync")
		}
		logger.Errorf("Failed to construct app version statements for license sync: %v", err)
	} else {
		statements = append(statements, appVersionStatements...)
	}

	if wrs, err := db.WriteParameterized(statements); err != nil {
		wrErrs := []error{}
		for _, wr := range wrs {
			wrErrs = append(wrErrs, wr.Err)
		}
		return int64(0), fmt.Errorf("failed to write: %v: %v", err, wrErrs)
	}

	return newSeq, nil
}

func (s *KOTSStore) UpdateAppLicenseSyncNow(appID string) error {
	db := persistence.MustGetDBSession()
	query := `update app set last_license_sync = ? where id = ?`
	wr, err := db.WriteOneParameterized(gorqlite.ParameterizedStatement{
		Query:     query,
		Arguments: []interface{}{time.Now().Unix(), appID},
	})
	if err != nil {
		return fmt.Errorf("update app %q license sync time: %v: %v", appID, err, wr.Err)
	}

	return nil
}

func (s *KOTSStore) createNewVersionForLicenseChangeStatements(appID string, baseSequence int64, archiveDir string, renderer rendertypes.Renderer, reportingInfo *reportingtypes.ReportingInfo) ([]gorqlite.ParameterizedStatement, int64, error) {
	registrySettings, err := s.GetRegistryDetailsForApp(appID)
	if err != nil {
		return nil, int64(0), errors.Wrap(err, "failed to get registry settings for app")
	}

	app, err := s.GetApp(appID)
	if err != nil {
		return nil, int64(0), errors.Wrap(err, "failed to get app")
	}

	downstreams, err := s.ListDownstreamsForApp(appID)
	if err != nil {
		return nil, int64(0), errors.Wrap(err, "failed to list downstreams")
	}

	nextAppSequence, err := s.GetNextAppSequence(appID)
	if err != nil {
		return nil, int64(0), errors.Wrap(err, "failed to get next app sequence")
	}

	if err := renderer.RenderDir(rendertypes.RenderDirOptions{
		ArchiveDir:       archiveDir,
		App:              app,
		Downstreams:      downstreams,
		RegistrySettings: registrySettings,
		Sequence:         nextAppSequence,
		ReportingInfo:    reportingInfo,
	}); err != nil {
		return nil, int64(0), errors.Wrap(err, "failed to render new version")
	}

	appVersionStatements, newSequence, err := s.createAppVersionStatements(appID, &baseSequence, archiveDir, "License Change", false, renderer)
	if err != nil {
		return nil, int64(0), errors.Wrap(err, "failed to construct app version statements")
	}

	return appVersionStatements, newSequence, nil
}
