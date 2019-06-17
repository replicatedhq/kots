package store

import (
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"mime/multipart"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
	"github.com/pkg/errors"
	"github.com/replicatedhq/ship-cluster/worker/pkg/types"
)

func (s *SQLStore) GetS3StoreURL(shipSession types.Session) (string, error) {
	sess, err := session.NewSession(s.getS3Config())
	if err != nil {
		return "", errors.Wrap(err, "new session")
	}
	svc := s3.New(sess)

	resp, _ := svc.PutObjectRequest(&s3.PutObjectInput{
		Bucket: aws.String(strings.TrimSpace(s.c.S3BucketName)),
		Key:    aws.String(shipSession.GetS3Filepath()),
	})

	presignedURL, err := resp.Presign(30 * time.Minute)
	if err != nil {
		return "", errors.Wrap(err, "presign response")
	}

	return presignedURL, nil
}

func (s *SQLStore) UploadToS3(ctx context.Context, outputSession types.Output, file multipart.File) error {
	file.Seek(0, io.SeekStart)
	s3bucket := strings.TrimSpace(s.c.S3BucketName)

	if strings.TrimSpace(s.c.S3Endpoint) != "" {
		uploadURL := fmt.Sprintf("%s%s/%s", strings.TrimSpace(s.c.S3Endpoint), s3bucket, outputSession.GetS3Filepath())

		req, err := http.NewRequest("PUT", uploadURL, file)
		if err != nil {
			return errors.Wrap(err, "create local s3 put request")
		}
		req.Header.Set("Content-Type", "text/plain")

		client := &http.Client{}
		res, err := client.Do(req)
		if err != nil {
			return errors.Wrap(err, "execute locals3 put request")
		}

		return errors.Wrap(res.Body.Close(), "close locals3 put request body")
	}

	sess, err := session.NewSession(s.getS3Config())
	if err != nil {
		return errors.Wrap(err, "new session")
	}
	svc := s3.New(sess)

	_, err = svc.PutObject(&s3.PutObjectInput{
		Bucket: aws.String(s3bucket),
		Key:    aws.String(outputSession.GetS3Filepath()),
		Body:   file,
	})

	return errors.Wrap(err, "s3PutObject")
}

func (s *SQLStore) DownloadFromS3(ctx context.Context, path string) (string, error) {
	s3bucket := strings.TrimSpace(s.c.S3BucketName)
	key := path

	sess, err := session.NewSession(s.getS3Config())
	if err != nil {
		return "", errors.Wrap(err, "new session")
	}

	file, err := ioutil.TempFile("", "shipupdate")
	if err != nil {
		return "", errors.Wrap(err, "temp file")
	}
	defer file.Close()

	downloader := s3manager.NewDownloader(sess)

	_, err = downloader.Download(file,
		&s3.GetObjectInput{
			Bucket: aws.String(s3bucket),
			Key:    aws.String(key),
		},
	)
	if err != nil {
		return "", errors.Wrap(err, "download")
	}

	return file.Name(), nil
}

func (s *SQLStore) SetOutputFilepath(ctx context.Context, outputSession types.Output) error {
	query := `insert into ship_output_files (watch_id, created_at, sequence, filepath) values ($1, $2, $3, $4)`

	_, err := s.db.ExecContext(
		ctx,
		query,
		outputSession.GetWatchID(),
		time.Now(),
		outputSession.GetUploadSequence(),
		outputSession.GetS3Filepath(),
	)
	return err
}

func (s *SQLStore) getS3Config() *aws.Config {
	region := "us-east-1"
	if os.Getenv("AWS_REGION") != "" {
		region = os.Getenv("AWS_REGION")
	}

	s3config := &aws.Config{
		Region: aws.String(region),
	}

	if strings.TrimSpace(s.c.S3Endpoint) != "" {
		s3config.Endpoint = aws.String(strings.TrimSpace(s.c.S3Endpoint))
	}

	if strings.TrimSpace(s.c.S3AccessKeyID) != "" && strings.TrimSpace(s.c.S3SecretAccessKey) != "" {
		s3config.Credentials = credentials.NewStaticCredentials(strings.TrimSpace(s.c.S3AccessKeyID), strings.TrimSpace(s.c.S3SecretAccessKey), "")
	}

	if strings.TrimSpace(s.c.S3BucketEndpoint) != "" {
		s3config.S3ForcePathStyle = aws.Bool(true)
	}

	return s3config
}
