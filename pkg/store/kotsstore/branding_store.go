package kotsstore

import (
	"database/sql"
	"time"

	"github.com/pkg/errors"
	"github.com/replicatedhq/kots/pkg/persistence"
	"github.com/segmentio/ksuid"
)

// GetInitialBranding returns the latest initial branding archive
func (s *KOTSStore) GetInitialBranding() ([]byte, error) {
	db := persistence.MustGetDBSession()

	query := `SELECT contents
FROM initial_branding
ORDER BY created_at DESC
LIMIT 1`
	row := db.QueryRow(query)

	var brandingArchive []byte
	err := row.Scan(&brandingArchive)
	if err == sql.ErrNoRows {
		// no initial branding, return empty
		return nil, nil
	} else if err != nil {
		return nil, errors.Wrap(err, "failed to scan branding")
	}

	return brandingArchive, nil
}

// CreateInitialBranding creates a new initial branding archive
func (s *KOTSStore) CreateInitialBranding(brandingArchive []byte) (string, error) {
	db := persistence.MustGetDBSession()

	id := ksuid.New().String()

	query := `INSERT INTO
initial_branding (id, contents, created_at)
VALUES ($1, $2, $3)`
	_, err := db.Exec(query, id, brandingArchive, time.Now())
	if err != nil {
		return "", errors.Wrap(err, "failed to insert initial branding")
	}

	return id, nil
}

// GetLatestBranding returns the latest branding archive for any app
func (s *KOTSStore) GetLatestBranding() ([]byte, error) {
	db := persistence.MustGetDBSession()

	// get the branding for the latest deployed version of any app
	query := `SELECT av.branding_archive
FROM
	app_downstream AS ad
INNER JOIN
	app_version AS av
ON
	ad.app_id = av.app_id AND ad.current_sequence = av.sequence
LEFT JOIN
	app_downstream_version AS adv
ON
	adv.app_id = av.app_id AND adv.parent_sequence = av.sequence
ORDER BY
	adv.applied_at DESC
LIMIT 1`
	row := db.QueryRow(query)

	var brandingArchive []byte
	err := row.Scan(&brandingArchive)
	if err == sql.ErrNoRows {
		// no versioned branding, return the initial branding
		return s.GetInitialBranding()
	} else if err != nil {
		return nil, errors.Wrap(err, "failed to scan latest deployed branding")
	}

	return brandingArchive, nil
}

// GetLatestBrandingForApp returns the latest branding archive for a specific app
func (s *KOTSStore) GetLatestBrandingForApp(appID string) ([]byte, error) {
	db := persistence.MustGetDBSession()

	// get the branding for the latest deployed version of this app
	query := `SELECT av.branding_archive
FROM
	app_downstream AS ad
INNER JOIN
	app_version AS av
ON
	ad.app_id = av.app_id AND ad.current_sequence = av.sequence
LEFT JOIN
	app_downstream_version AS adv
ON
	adv.app_id = av.app_id AND adv.parent_sequence = av.sequence
WHERE
	ad.app_id = $1
ORDER BY
	adv.applied_at DESC
LIMIT 1`
	row := db.QueryRow(query, appID)

	var brandingArchive []byte
	err := row.Scan(&brandingArchive)
	if err == sql.ErrNoRows {
		// no versioned branding, return the initial branding
		return s.GetInitialBranding()
	} else if err != nil {
		return nil, errors.Wrapf(err, "failed to scan latest deployed branding for app %s", appID)
	}

	return brandingArchive, nil
}
