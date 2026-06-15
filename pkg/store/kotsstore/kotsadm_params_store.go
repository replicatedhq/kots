package kotsstore

import (
	"fmt"
	"strconv"

	"github.com/pkg/errors"
	"github.com/replicatedhq/kots/pkg/persistence"
	"github.com/rqlite/gorqlite"
)

const environmentFingerprintKey = "ENVIRONMENT_FINGERPRINT"

// IsKotsadmIDGenerated retrieves the id of kotsadm if the pod is already
func (s *KOTSStore) IsKotsadmIDGenerated() (bool, error) {
	db := persistence.MustGetDBSession()
	query := `select value from kotsadm_params where key = 'IS_KOTSADM_ID_GENERATED'`
	rows, err := db.QueryOne(query)
	if err != nil {
		return false, fmt.Errorf("failed to query: %v: %v", err, rows.Err)
	}
	if !rows.Next() {
		return false, nil
	}

	var value string
	if err := rows.Scan(&value); err != nil {
		return false, errors.Wrap(err, "failed to scan")
	}

	parsedValue, err := strconv.ParseBool(value)
	if err != nil {
		return false, errors.Wrap(err, "failed to parse boolean value")
	}

	return parsedValue, nil
}

// GetEnvironmentFingerprint returns the stored environment fingerprint, or an empty
// string if one has not been recorded yet. The fingerprint is stored in the database
// on purpose: it is included in snapshots, so after a restore it describes the
// environment the backup was taken in.
func (s *KOTSStore) GetEnvironmentFingerprint() (string, error) {
	db := persistence.MustGetDBSession()
	rows, err := db.QueryOneParameterized(gorqlite.ParameterizedStatement{
		Query:     `select value from kotsadm_params where key = ?`,
		Arguments: []interface{}{environmentFingerprintKey},
	})
	if err != nil {
		return "", fmt.Errorf("failed to query: %v: %v", err, rows.Err)
	}
	if !rows.Next() {
		return "", nil
	}

	var value string
	if err := rows.Scan(&value); err != nil {
		return "", errors.Wrap(err, "failed to scan")
	}

	return value, nil
}

func (s *KOTSStore) SetEnvironmentFingerprint(fingerprint string) error {
	db := persistence.MustGetDBSession()

	query := `insert into kotsadm_params (key, value) values (?, ?) on conflict (key) do update set value = EXCLUDED.value`
	wr, err := db.WriteOneParameterized(gorqlite.ParameterizedStatement{
		Query:     query,
		Arguments: []interface{}{environmentFingerprintKey, fingerprint},
	})
	if err != nil {
		return fmt.Errorf("failed to write: %v: %v", err, wr.Err)
	}
	return nil
}

// SetIsKotsadmIDGenerated sets the status to true if the pod is starting for the first time
func (s *KOTSStore) SetIsKotsadmIDGenerated() error {
	db := persistence.MustGetDBSession()

	query := `insert into kotsadm_params (key, value) values (?, ?) on conflict (key) do update set value = EXCLUDED.value`
	wr, err := db.WriteOneParameterized(gorqlite.ParameterizedStatement{
		Query:     query,
		Arguments: []interface{}{"IS_KOTSADM_ID_GENERATED", true},
	})
	if err != nil {
		return fmt.Errorf("failed to write: %v: %v", err, wr.Err)
	}
	return nil
}
