package s3pg

import (
	"context"
	"database/sql"
	"log"
	"os"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	awssession "github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/pkg/errors"
	"github.com/replicatedhq/kots/kotsadm/pkg/logger"
	"github.com/replicatedhq/kots/kotsadm/pkg/persistence"
	kotss3 "github.com/replicatedhq/kots/kotsadm/pkg/s3"
	kotsscheme "github.com/replicatedhq/kots/kotskinds/client/kotsclientset/scheme"
	troubleshootscheme "github.com/replicatedhq/troubleshoot/pkg/client/troubleshootclientset/scheme"
	veleroscheme "github.com/vmware-tanzu/velero/pkg/generated/clientset/versioned/scheme"
	"k8s.io/client-go/kubernetes/scheme"
)

type S3PGStore struct {
}

func init() {
	kotsscheme.AddToScheme(scheme.Scheme)
	veleroscheme.AddToScheme(scheme.Scheme)
	troubleshootscheme.AddToScheme(scheme.Scheme)
}

func (s S3PGStore) Init() error {
	if strings.HasPrefix(os.Getenv("STORAGE_BASEURI"), "docker://") {
		return nil
	}

	if os.Getenv("S3_BUCKET_NAME") == "ship-pacts" {
		log.Println("Not creating bucket because the desired name is ship-pacts. Consider using a different bucket name to make this work.")
		return errors.New("bad bucket name")
	}

	if os.Getenv("S3_SKIP_ENSURE_BUCKET") == "1" {
		log.Println("Not creating bucket because S3_SKIP_ENSURE_BUCKET was set.")
		return nil
	}

	newSession := awssession.New(kotss3.GetConfig())
	s3Client := s3.New(newSession)

	_, err := s3Client.HeadBucket(&s3.HeadBucketInput{
		Bucket: aws.String(os.Getenv("S3_BUCKET_NAME")),
	})

	if err == nil {
		return nil
	}

	_, err = s3Client.CreateBucket(&s3.CreateBucketInput{
		Bucket: aws.String(os.Getenv("S3_BUCKET_NAME")),
	})
	if err != nil {
		return errors.Wrap(err, "failed to create bucket")
	}

	return nil
}

func (s S3PGStore) WaitForReady(ctx context.Context) error {
	waitForPostgres := func(ctx context.Context) error {
		logger.Debug("waiting for database to be ready")

		period := 1 * time.Second // TOOD: backoff
		for {
			db := persistence.MustGetPGSession()

			// any SQL will do.  just need tables to be created.
			query := `select count(1) from app`
			row := db.QueryRow(query)

			var count int
			if err := row.Scan(&count); err == nil {
				logger.Debug("database is ready")
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

	errCh := make(chan error, 1)

	go func() {
		errCh <- waitForPostgres(ctx)
	}()

	isError := false
	for i := 0; i < 1; i++ {
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

func (s S3PGStore) IsNotFound(err error) bool {
	if err == nil {
		return false
	}
	return errors.Cause(err) == sql.ErrNoRows
}
