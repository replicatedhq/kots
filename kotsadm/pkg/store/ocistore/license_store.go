package ocistore

import (
	"database/sql"

	"github.com/pkg/errors"
	kotsv1beta1 "github.com/replicatedhq/kots/kotskinds/apis/kots/v1beta1"
	"github.com/replicatedhq/kots/kotskinds/client/kotsclientset/scheme"
)

// GetInitialLicenseForApp reads from the app, not app version
func (s OCIStore) GetInitialLicenseForApp(appID string) (*kotsv1beta1.License, error) {
	query := `select license from app where id = $1`
	row := s.connection.DB.QueryRow(query, appID)

	var licenseStr sql.NullString
	if err := row.Scan(&licenseStr); err != nil {
		return nil, errors.Wrap(err, "failed to scan license")
	}

	if !licenseStr.Valid {
		return nil, ErrNotFound
	}

	decode := scheme.Codecs.UniversalDeserializer().Decode
	obj, _, err := decode([]byte(licenseStr.String), nil, nil)
	if err != nil {
		return nil, errors.Wrap(err, "failed to decode license yaml")
	}
	license := obj.(*kotsv1beta1.License)
	return license, nil
}

func (s OCIStore) GetLatestLicenseForApp(appID string) (*kotsv1beta1.License, error) {
	query := `select kots_license from app_version where app_id = $1 order by sequence desc limit 1`
	row := s.connection.DB.QueryRow(query, appID)

	var licenseStr sql.NullString
	if err := row.Scan(&licenseStr); err != nil {
		if err == sql.ErrNoRows {
			return s.GetInitialLicenseForApp(appID)
		}

		return nil, errors.Wrap(err, "failed to scan")
	}

	if !licenseStr.Valid {
		return s.GetInitialLicenseForApp(appID)
	}

	decode := scheme.Codecs.UniversalDeserializer().Decode
	obj, _, err := decode([]byte(licenseStr.String), nil, nil)
	if err != nil {
		return nil, errors.Wrap(err, "failed to decode license yaml")
	}
	license := obj.(*kotsv1beta1.License)
	return license, nil
}

func (s OCIStore) GetLicenseForAppVersion(appID string, sequence int64) (*kotsv1beta1.License, error) {
	query := `select kots_license from app_version where app_id = $1 and sequence = $2`
	row := s.connection.DB.QueryRow(query, appID, sequence)

	var licenseStr sql.NullString
	if err := row.Scan(&licenseStr); err != nil {
		if err == sql.ErrNoRows {
			return s.GetInitialLicenseForApp(appID)
		}

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

func (s OCIStore) GetAllAppLicenses() ([]*kotsv1beta1.License, error) {
	query := `select license from app`
	rows, err := s.connection.DB.Query(query)
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
