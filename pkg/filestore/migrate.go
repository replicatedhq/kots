package filestore

import (
	"bytes"
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
	_ "github.com/mattn/go-sqlite3"
	"github.com/pkg/errors"
	"github.com/replicatedhq/kots/pkg/persistence"
	kotss3 "github.com/replicatedhq/kots/pkg/s3"
	"github.com/rqlite/gorqlite"
)

const (
	RQLITE_S3_MIGRATION_SUCCESS_KEY   = "rqlite.s3.migration.success"
	RQLITE_BLOB_MIGRATION_SUCCESS_KEY = "rqlite.blob.migration.success"
	RQLITE_MIGRATION_SUCCESS_VALUE    = "true"
)

func MigrateFromS3ToRqlite(ctx context.Context) error {
	// Check if already migrated
	rqliteDB := persistence.MustGetDBSession()
	alreadyMigrated, err := isAlreadyMigrated(rqliteDB, RQLITE_S3_MIGRATION_SUCCESS_KEY)
	if err != nil {
		return errors.Wrap(err, "failed to check if already migrated")
	}
	if alreadyMigrated {
		log.Println("Already migrated from S3 to rqlite. Skipping migration...")
		return nil
	}

	log.Println("Migrating from S3 to rqlite...")

	// Initialize rqlite store
	rqliteStore := &RqliteStore{}
	if err := rqliteStore.Init(); err != nil {
		return errors.Wrap(err, "failed to init rqlite store")
	}
	if err := rqliteStore.WaitForReady(ctx); err != nil {
		return errors.Wrap(err, "failed to wait for rqlite store to become ready")
	}

	// Create a new S3 session
	sess, err := session.NewSession(kotss3.GetConfig())
	if err != nil {
		return errors.Wrap(err, "failed to create new s3 session")
	}

	// Create an S3 client
	s3Client := s3.New(sess)

	// List objects in the bucket
	listObjectsInput := &s3.ListObjectsV2Input{
		Bucket: aws.String(os.Getenv("S3_BUCKET_NAME")),
	}
	listObjectsOutput, err := s3Client.ListObjectsV2(listObjectsInput)
	if err != nil {
		return errors.Wrap(err, "failed to list objects in bucket")
	}

	// Initialize the S3 downloader
	downloader := s3manager.NewDownloader(sess)

	// Process each object
	for _, item := range listObjectsOutput.Contents {
		if item == nil || item.Key == nil {
			continue
		}
		key := *item.Key
		log.Printf("Processing key: %s\n", key)

		// Download the object
		buff := &aws.WriteAtBuffer{}
		_, err := downloader.Download(buff, &s3.GetObjectInput{
			Bucket: aws.String(os.Getenv("S3_BUCKET_NAME")),
			Key:    aws.String(key),
		})
		if err != nil {
			return errors.Wrap(err, "failed to download object")
		}

		// Write the object to rqlite
		if err := rqliteStore.WriteArchive(key, bytes.NewReader(buff.Bytes())); err != nil {
			return errors.Wrap(err, "failed to write archive to rqlite")
		}
	}

	// Record the migration success
	query := `REPLACE INTO kotsadm_params (key, value) VALUES (?, ?)`
	wr, err := rqliteDB.WriteOneParameterized(gorqlite.ParameterizedStatement{
		Query:     query,
		Arguments: []interface{}{RQLITE_S3_MIGRATION_SUCCESS_KEY, RQLITE_MIGRATION_SUCCESS_VALUE},
	})
	if err != nil {
		return fmt.Errorf("failed to mark migration as successful: %v: %v", err, wr.Err)
	}

	log.Println("Migrated from S3 to rqlite successfully!")

	return nil
}

func MigrateFromPVCToRqlite(ctx context.Context) error {
	// Check if already migrated
	rqliteDB := persistence.MustGetDBSession()
	alreadyMigrated, err := isAlreadyMigrated(rqliteDB, RQLITE_BLOB_MIGRATION_SUCCESS_KEY)
	if err != nil {
		return errors.Wrap(err, "failed to check if already migrated")
	}
	if alreadyMigrated {
		log.Println("Already migrated from PVC to rqlite. Skipping migration...")
		return nil
	}

	log.Println("Migrating from PVC to rqlite...")

	// Initialize rqlite store
	rqliteStore := &RqliteStore{}
	if err := rqliteStore.Init(); err != nil {
		return errors.Wrap(err, "failed to init rqlite store")
	}
	if err := rqliteStore.WaitForReady(ctx); err != nil {
		return errors.Wrap(err, "failed to wait for rqlite store to become ready")
	}

	// Process each object
	err = filepath.Walk(ArchivesDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return errors.Wrap(err, "failed to walk path")
		}

		if info.IsDir() {
			return nil
		}

		key, err := filepath.Rel(ArchivesDir, path)
		if err != nil {
			return errors.Wrap(err, "failed to get relative path")
		}
		log.Printf("Processing key: %s\n", key)

		// Open the file
		file, err := os.Open(path)
		if err != nil {
			return errors.Wrap(err, "failed to open file")
		}
		defer file.Close()

		// Write the object to rqlite
		if err := rqliteStore.WriteArchive(key, file); err != nil {
			return errors.Wrap(err, "failed to write archive to rqlite")
		}

		return nil
	})
	if err != nil {
		return errors.Wrap(err, "failed to walk PVC mount")
	}

	// Record the migration success
	query := `REPLACE INTO kotsadm_params (key, value) VALUES (?, ?)`
	wr, err := rqliteDB.WriteOneParameterized(gorqlite.ParameterizedStatement{
		Query:     query,
		Arguments: []interface{}{RQLITE_BLOB_MIGRATION_SUCCESS_KEY, RQLITE_MIGRATION_SUCCESS_VALUE},
	})
	if err != nil {
		return fmt.Errorf("failed to mark migration as successful: %v: %v", err, wr.Err)
	}

	log.Println("Migrated from PVC to rqlite successfully!")

	return nil
}

func isAlreadyMigrated(rqliteDB *gorqlite.Connection, migrationKey string) (bool, error) {
	rows, err := rqliteDB.QueryOneParameterized(gorqlite.ParameterizedStatement{
		Query:     `SELECT value FROM kotsadm_params WHERE key = ?`,
		Arguments: []interface{}{migrationKey},
	})
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

	return value == RQLITE_MIGRATION_SUCCESS_VALUE, nil
}
