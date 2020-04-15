package supportbundle

import (
	"time"

	"github.com/pkg/errors"
	"github.com/replicatedhq/kotsadm/pkg/persistence"
	"github.com/replicatedhq/kotsadm/pkg/supportbundle/types"
	"github.com/segmentio/ksuid"
)

func CreateBundle(bundleSlug string, appID string, archivePath string) (*types.SupportBundle, error) {
	db := persistence.MustGetPGSession()
	query := `insert into supportbundle (id, slug, watch_id, size, status, created_at) values ($1, $2, $3, $4, $5, $6)`

	id := ksuid.New().String()

	// TODO
	// upload the file to s3

	_, err := db.Exec(query, id, bundleSlug, appID, 0, "", time.Now())
	if err != nil {
		return nil, errors.Wrap(err, "failed to insert support bundle")
	}

	return &types.SupportBundle{
		ID: id,
	}, nil
}
