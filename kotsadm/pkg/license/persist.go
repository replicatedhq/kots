package license

import (
	"github.com/pkg/errors"
	apptypes "github.com/replicatedhq/kots/kotsadm/pkg/app/types"
	"github.com/replicatedhq/kots/kotsadm/pkg/persistence"
)

func updateAppLicense(a *apptypes.App, licenseData string) error {
	db := persistence.MustGetPGSession()

	query := `update app set license=$1 where id = $2`
	_, err := db.Exec(query, licenseData, a.ID)
	return errors.Wrapf(err, "update app %q license", a.ID)
}
