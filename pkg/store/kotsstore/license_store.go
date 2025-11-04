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
	"github.com/replicatedhq/kotskinds/client/kotsclientset/scheme"
	"github.com/replicatedhq/kotskinds/pkg/licensewrapper"
	"github.com/rqlite/gorqlite"
	serializer "k8s.io/apimachinery/pkg/runtime/serializer/json"
)

func (s *KOTSStore) GetLatestLicenseForApp(appID string) (*licensewrapper.LicenseWrapper, error) {
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

	// Use licensewrapper to auto-detect version (v1beta1 or v1beta2)
	wrapper, err := licensewrapper.LoadLicenseFromBytes([]byte(licenseStr.String))
	if err != nil {
		return nil, errors.Wrap(err, "failed to decode license yaml")
	}
	return &wrapper, nil
}

func (s *KOTSStore) GetLicenseForAppVersion(appID string, sequence int64) (*licensewrapper.LicenseWrapper, error) {
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
		// Use licensewrapper to auto-detect version (v1beta1 or v1beta2)
		wrapper, err := licensewrapper.LoadLicenseFromBytes([]byte(licenseStr.String))
		if err != nil {
			return nil, errors.Wrap(err, "failed to load license from bytes")
		}
		return &wrapper, nil
	}

	// Return nil (no license for this version)
	return nil, nil
}

func (s *KOTSStore) GetAllAppLicenses() ([]*licensewrapper.LicenseWrapper, error) {
	db := persistence.MustGetDBSession()
	query := `select license from app`
	rows, err := db.QueryOne(query)
	if err != nil {
		return nil, fmt.Errorf("failed to query: %v: %v", err, rows.Err)
	}

	var licenseStr gorqlite.NullString
	licenses := []*licensewrapper.LicenseWrapper{}
	for rows.Next() {
		if err := rows.Scan(&licenseStr); err != nil {
			return nil, errors.Wrap(err, "failed to scan")
		}
		if licenseStr.Valid {
			// Use licensewrapper to auto-detect version (v1beta1 or v1beta2)
			wrapper, err := licensewrapper.LoadLicenseFromBytes([]byte(licenseStr.String))
			if err != nil {
				return nil, errors.Wrap(err, "failed to load license from bytes")
			}
			licenses = append(licenses, &wrapper)
		}
	}

	return licenses, nil
}

func (s *KOTSStore) UpdateAppLicense(appID string, baseSequence int64, archiveDir string, newLicense *licensewrapper.LicenseWrapper, originalLicenseData string, channelChanged bool, failOnVersionCreate bool, renderer rendertypes.Renderer, reportingInfo *reportingtypes.ReportingInfo) (int64, error) {
	db := persistence.MustGetDBSession()

	statements := []gorqlite.ParameterizedStatement{}

	// Marshal license to YAML (handle both v1beta1 and v1beta2)
	ser := serializer.NewYAMLSerializer(serializer.DefaultMetaFactory, scheme.Scheme, scheme.Scheme)
	var b bytes.Buffer
	if newLicense.IsV1() {
		if err := ser.Encode(newLicense.V1, &b); err != nil {
			return int64(0), errors.Wrap(err, "failed to marshal v1beta1 license")
		}
	} else if newLicense.IsV2() {
		if err := ser.Encode(newLicense.V2, &b); err != nil {
			return int64(0), errors.Wrap(err, "failed to marshal v1beta2 license")
		}
	}
	encodedLicense := b.Bytes()
	if err := ioutil.WriteFile(filepath.Join(archiveDir, "upstream", "userdata", "license.yaml"), encodedLicense, 0644); err != nil {
		return int64(0), errors.Wrap(err, "failed to write new license")
	}

	selectedChannelId, err := s.GetAppSelectedChannelID(appID)
	if err != nil {
		return int64(0), errors.Wrap(err, "failed to get existing app selected channel id")
	}

	// If the license channels array has more than one entry, then the license is a true multi-channel license,
	// and we should skip updating selected_channel_id in the app table. If there's only a single entry,
	// we should update the selected_channel_id in the app table to ensure it stays consistent across channel
	// changes. This is a temporary solution until channel changes on true multi-channel licenses are supported.
	if len(newLicense.GetChannels()) > 1 {
		logger.Debug("Skipping selected_channel_id update for multi-channel license")
		//  app has the original license data received from the server
		statements = append(statements, gorqlite.ParameterizedStatement{
			Query:     `update app set license = ?, last_license_sync = ?, channel_changed = ? where id = ?`,
			Arguments: []interface{}{originalLicenseData, time.Now().Unix(), channelChanged, appID},
		})
	} else {
		//  app has the original license data received from the server
		statements = append(statements, gorqlite.ParameterizedStatement{
			Query:     `update app set license = ?, last_license_sync = ?, channel_changed = ?, selected_channel_id = ? where id = ?`,
			Arguments: []interface{}{originalLicenseData, time.Now().Unix(), channelChanged, newLicense.GetChannelID(), appID},
		})
		selectedChannelId = newLicense.GetChannelID()
	}

	appVersionStatements, newSeq, err := s.createNewVersionForLicenseChangeStatements(appID, baseSequence, archiveDir, renderer, reportingInfo, selectedChannelId)
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

func (s *KOTSStore) createNewVersionForLicenseChangeStatements(appID string, baseSequence int64, archiveDir string, renderer rendertypes.Renderer, reportingInfo *reportingtypes.ReportingInfo, selectedChannelID string) ([]gorqlite.ParameterizedStatement, int64, error) {
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
		ArchiveDir:           archiveDir,
		App:                  app,
		Downstreams:          downstreams,
		RegistrySettings:     registrySettings,
		Sequence:             nextAppSequence,
		ReportingInfo:        reportingInfo,
		AppSelectedChannelID: selectedChannelID,
	}); err != nil {
		return nil, int64(0), errors.Wrap(err, "failed to render new version")
	}

	appVersionStatements, newSequence, err := s.createAppVersionStatements(appID, &baseSequence, archiveDir, "License Change", false, false, false)
	if err != nil {
		return nil, int64(0), errors.Wrap(err, "failed to construct app version statements")
	}

	return appVersionStatements, newSequence, nil
}
