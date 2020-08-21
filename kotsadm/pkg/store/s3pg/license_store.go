package s3pg

import (
	"database/sql"

	"github.com/pkg/errors"
	"github.com/replicatedhq/kots/kotsadm/pkg/persistence"
	kotsv1beta1 "github.com/replicatedhq/kots/kotskinds/apis/kots/v1beta1"
	"github.com/replicatedhq/kots/kotskinds/client/kotsclientset/scheme"
)

func (s S3PGStore) GetLicenseForApp(appID string) (*kotsv1beta1.License, error) {
	db := persistence.MustGetPGSession()
	query := `select kots_license from app_version where app_id = $1 order by sequence desc limit 1`
	row := db.QueryRow(query, appID)

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
