package kotsstore

import (
	"encoding/base64"
	"fmt"
	"time"

	"github.com/pkg/errors"
	"github.com/replicatedhq/kots/pkg/persistence"
	"github.com/rqlite/gorqlite"
	"github.com/segmentio/ksuid"
)

// GetInitialBranding returns the latest initial branding archive
func (s *KOTSStore) GetInitialBranding() ([]byte, error) {
	db := persistence.MustGetDBSession()

	query := `SELECT contents FROM initial_branding ORDER BY created_at DESC LIMIT 1`
	rows, err := db.QueryOne(query)
	if err != nil {
		return nil, fmt.Errorf("failed to query: %v: %v", err, rows.Err)
	}
	if !rows.Next() {
		// no initial branding, return empty
		return nil, nil
	}

	var encodedBrandingArchive gorqlite.NullString
	if err := rows.Scan(&encodedBrandingArchive); err != nil {
		return nil, errors.Wrap(err, "failed to scan branding")
	}

	var brandingArchive []byte
	if encodedBrandingArchive.Valid {
		b, err := base64.StdEncoding.DecodeString(encodedBrandingArchive.String)
		if err != nil {
			return nil, errors.Wrap(err, "failed to decode")
		}
		brandingArchive = b
	}

	return brandingArchive, nil
}

// CreateInitialBranding creates a new initial branding archive
func (s *KOTSStore) CreateInitialBranding(brandingArchive []byte) (string, error) {
	db := persistence.MustGetDBSession()

	id := ksuid.New().String()
	encodedBrandingArchive := base64.StdEncoding.EncodeToString(brandingArchive)

	query := `INSERT INTO initial_branding (id, contents, created_at) VALUES (?, ?, ?)`
	wr, err := db.WriteOneParameterized(gorqlite.ParameterizedStatement{
		Query:     query,
		Arguments: []interface{}{id, encodedBrandingArchive, time.Now().Unix()},
	})
	if err != nil {
		return "", fmt.Errorf("failed to insert initial branding: %v: %v", err, wr.Err)
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
	rows, err := db.QueryOne(query)
	if err != nil {
		return nil, fmt.Errorf("failed to query: %v: %v", err, rows.Err)
	}

	if !rows.Next() {
		// no versioned branding, return the initial branding
		return s.GetInitialBranding()
	}

	var encodedBrandingArchive gorqlite.NullString
	if err := rows.Scan(&encodedBrandingArchive); err != nil {
		return nil, errors.Wrap(err, "failed to scan latest deployed branding")
	}

	var brandingArchive []byte
	if encodedBrandingArchive.Valid {
		b, err := base64.StdEncoding.DecodeString(encodedBrandingArchive.String)
		if err != nil {
			return nil, errors.Wrap(err, "failed to decode")
		}
		brandingArchive = b
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
	ad.app_id = ?
ORDER BY
	adv.applied_at DESC
LIMIT 1`
	rows, err := db.QueryOneParameterized(gorqlite.ParameterizedStatement{
		Query:     query,
		Arguments: []interface{}{appID},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to query: %v: %v", err, rows.Err)
	}

	if !rows.Next() {
		// no versioned branding, return the initial branding
		return s.GetInitialBranding()
	}

	var encodedBrandingArchive gorqlite.NullString
	if err := rows.Scan(&encodedBrandingArchive); err != nil {
		return nil, errors.Wrapf(err, "failed to scan latest deployed branding for app %s", appID)
	}

	var brandingArchive []byte
	if encodedBrandingArchive.Valid {
		b, err := base64.StdEncoding.DecodeString(encodedBrandingArchive.String)
		if err != nil {
			return nil, errors.Wrap(err, "failed to decode")
		}
		brandingArchive = b
	}

	return brandingArchive, nil
}
