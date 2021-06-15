package s3store

import (
	"context"
	"io"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	awssession "github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
	"github.com/pkg/errors"
	"github.com/replicatedhq/kots/pkg/logger"
	kotss3 "github.com/replicatedhq/kots/pkg/s3"
)

type S3Store struct {
}

func (s *S3Store) Init() error {
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

func (s *S3Store) WaitForReady(ctx context.Context) error {
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

	logger.Debug("waiting for object store to be ready")

	newSession := awssession.New(kotss3.GetConfig())
	s3Client := s3.New(newSession)

	period := 1 * time.Second // TOOD: backoff
	for {
		_, err := s3Client.HeadBucket(&s3.HeadBucketInput{
			Bucket: aws.String(os.Getenv("S3_BUCKET_NAME")),
		})
		if err == nil {
			logger.Debug("object store is ready")
			return nil
		}
		if awsErr, ok := err.(awserr.Error); ok && awsErr.Code() == "NotFound" {
			logger.Debug("object store is ready")
			return nil
		}

		select {
		case <-time.After(period):
			continue
		case <-ctx.Done():
			return errors.Wrap(err, "failed to find valid object store")
		}
	}
}

func (s *S3Store) PutObject(bucket string, key string, body io.ReadSeeker) error {
	newSession := awssession.New(kotss3.GetConfig())

	s3Client := s3.New(newSession)

	_, err := s3Client.PutObject(&s3.PutObjectInput{
		Body:   body,
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		return errors.Wrap(err, "failed to upload to s3")
	}

	return nil
}

func (s *S3Store) GetObject(bucket string, key string) (string, error) {
	tmpDir, err := ioutil.TempDir("", "kotsadm")
	if err != nil {
		return "", errors.Wrap(err, "failed to create temp dir")
	}

	newSession := awssession.New(kotss3.GetConfig())

	keyParts := strings.Split(key, string(os.PathSeparator))
	outputFilePath := filepath.Join(tmpDir, keyParts[len(keyParts)-1])
	outputFile, err := os.Create(outputFilePath)
	if err != nil {
		return "", errors.Wrap(err, "failed to open file")
	}
	defer outputFile.Close()

	downloader := s3manager.NewDownloader(newSession)
	_, err = downloader.Download(outputFile,
		&s3.GetObjectInput{
			Bucket: aws.String(bucket),
			Key:    aws.String(key),
		})
	if err != nil {
		return "", errors.Wrapf(err, "failed to download key %q from bucket %q", key, bucket)
	}

	return outputFilePath, nil
}
