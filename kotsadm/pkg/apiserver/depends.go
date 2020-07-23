package apiserver

import (
	"context"
	"log"
	"os"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	awssession "github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/pkg/errors"
	"github.com/replicatedhq/kots/kotsadm/pkg/logger"
	"github.com/replicatedhq/kots/kotsadm/pkg/persistence"
	kotss3 "github.com/replicatedhq/kots/kotsadm/pkg/s3"
)

func waitForDependencies(ctx context.Context) error {
	numChecks := 2
	errCh := make(chan error, numChecks)

	go func() {
		errCh <- waitForS3Bucket(ctx)
	}()

	go func() {
		errCh <- waitForPostgres(ctx)
	}()

	isError := false
	for i := 0; i < numChecks; i++ {
		err := <-errCh
		if err != nil {
			log.Println(err.Error())
			isError = true
		}
	}

	if isError {
		return errors.New("failed to wait for dependencies")
	}

	return nil
}

func waitForS3Bucket(ctx context.Context) error {
	logger.Debug("waiting for s3 bucket to be created")

	period := 1 * time.Second // TOOD: backoff
	for {
		newSession := awssession.New(kotss3.GetConfig())
		s3Client := s3.New(newSession)

		_, err := s3Client.HeadBucket(&s3.HeadBucketInput{
			Bucket: aws.String(os.Getenv("S3_BUCKET_NAME")),
		})

		if err == nil {
			return nil
		}

		select {
		case <-time.After(period):
			continue
		case <-ctx.Done():
			return errors.Wrap(ctx.Err(), "failed to find s3 bucket")
		}
	}
}

func waitForPostgres(ctx context.Context) error {
	logger.Debug("waiting for database to be ready")

	period := 1 * time.Second // TOOD: backoff
	for {
		db := persistence.MustGetPGSession()

		// any SQL will do.  just need tables to be created.
		query := `select count(1) from app`
		row := db.QueryRow(query)

		var count int
		if err := row.Scan(&count); err == nil {
			return nil
		}

		select {
		case <-time.After(period):
			continue
		case <-ctx.Done():
			return errors.Wrap(ctx.Err(), "failed to find valid database")
		}
	}
}
