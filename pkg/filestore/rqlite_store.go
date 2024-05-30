package filestore

import (
	"context"
	"encoding/base64"
	"fmt"
	"io"
	"os"

	"github.com/pkg/errors"
	"github.com/replicatedhq/kots/pkg/persistence"
	"github.com/rqlite/gorqlite"
)

var (
	ErrNotFound = errors.New("not found")
)

type RqliteStore struct {
}

func (s *RqliteStore) Init() error {
	// there's no initialization, the table is created by schemahero
	return nil
}

func (s *RqliteStore) WaitForReady(ctx context.Context) error {
	// there's no waiting, the table must exist at this point
	db := persistence.MustGetDBSession()

	query := `SELECT filepath FROM object_store LIMIT 1`
	_, err := db.QueryOne(query)
	if err != nil {
		return errors.Wrap(err, "failed to query")
	}

	return nil
}

func (s *RqliteStore) WriteArchive(outputPath string, body io.ReadSeeker) error {
	db := persistence.MustGetDBSession()

	bodyBytes, err := io.ReadAll(body)
	if err != nil {
		return errors.Wrap(err, "failed to read body")
	}

	query := `
INSERT INTO object_store (filepath, encoded_block)
VALUES (?, ?)
ON CONFLICT (filepath) DO UPDATE SET
	encoded_block = excluded.encoded_block
`
	wr, err := db.WriteOneParameterized(gorqlite.ParameterizedStatement{
		Query:     query,
		Arguments: []interface{}{outputPath, base64.StdEncoding.EncodeToString(bodyBytes)},
	})
	if err != nil {
		return fmt.Errorf("failed to write: %v: %v", err, wr.Err)
	}

	return nil
}

func (s *RqliteStore) ReadArchive(path string) (string, error) {
	db := persistence.MustGetDBSession()

	query := `SELECT encoded_block FROM object_store WHERE filepath = ?`
	rows, err := db.QueryOneParameterized(gorqlite.ParameterizedStatement{
		Query:     query,
		Arguments: []interface{}{path},
	})
	if err != nil {
		return "", fmt.Errorf("failed to read: %v: %v", err, rows.Err)
	}
	if !rows.Next() {
		return "", ErrNotFound
	}

	var encoded string
	if err := rows.Scan(&encoded); err != nil {
		return "", errors.Wrap(err, "failed to scan")
	}

	decoded, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return "", errors.Wrap(err, "failed to decode")
	}

	tmpFile, err := os.CreateTemp("", "kotsadm")
	if err != nil {
		return "", errors.Wrap(err, "failed to create temp file")
	}
	if err := os.WriteFile(tmpFile.Name(), decoded, 0644); err != nil {
		return "", errors.Wrap(err, "failed to write to temp file")
	}

	return tmpFile.Name(), nil
}

func (s *RqliteStore) DeleteArchive(path string) error {
	db := persistence.MustGetDBSession()

	query := `DELETE FROM object_store WHERE filepath = ?`
	wr, err := db.WriteOneParameterized(gorqlite.ParameterizedStatement{
		Query:     query,
		Arguments: []interface{}{path},
	})
	if err != nil {
		return fmt.Errorf("failed to delete: %v: %v", err, wr.Err)
	}

	return nil
}
