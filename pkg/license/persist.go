package license

import (
	"github.com/pkg/errors"
	"github.com/replicatedhq/kotsadm/pkg/app"
	"github.com/replicatedhq/kotsadm/pkg/persistence"
)

func updateAppLicense(a *app.App, licenseData string) error {
	db := persistence.MustGetPGSession()

	query := `update app set license=$1 where id = $2`
	_, err := db.Exec(query, licenseData, a.ID)
	return errors.Wrap(err, "update app license")
}
