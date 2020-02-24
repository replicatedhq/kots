package app

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	awssession "github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
	"github.com/mholt/archiver"
	"github.com/pkg/errors"
)

// CreateAppVersion takes an unarchived app, makes an archive and then uploads it
// to s3 with the appID and sequence specified
func CreateAppVersionArchive(appID string, sequence int64, archivePath string) error {
	paths := []string{
		filepath.Join(archivePath, "upstream"),
		filepath.Join(archivePath, "base"),
		filepath.Join(archivePath, "overlays"),
	}

	tmpDir, err := ioutil.TempDir("", "kotsadm")
	if err != nil {
		return errors.Wrap(err, "failed to create temp file")
	}
	defer os.RemoveAll(tmpDir)
	fileToUpload := filepath.Join(tmpDir, "archive.tar.gz")

	tarGz := archiver.TarGz{
		Tar: &archiver.Tar{
			ImplicitTopLevelFolder: false,
		},
	}
	if err := tarGz.Archive(paths, fileToUpload); err != nil {
		return errors.Wrap(err, "failed to create archive")
	}

	forcePathStyle := false
	if os.Getenv("S3_BUCKET_ENDPOINT") == "true" {
		forcePathStyle = true
	}

	bucket := aws.String(os.Getenv("S3_BUCKET_NAME"))
	key := aws.String(fmt.Sprintf("%s/%d.tar.gz", appID, sequence))

	s3Config := &aws.Config{
		Credentials:      credentials.NewStaticCredentials(os.Getenv("S3_ACCESS_KEY_ID"), os.Getenv("S3_SECRET_ACCESS_KEY"), ""),
		Endpoint:         aws.String(os.Getenv("S3_ENDPOINT")),
		Region:           aws.String("us-east-1"),
		DisableSSL:       aws.Bool(true),
		S3ForcePathStyle: aws.Bool(forcePathStyle),
	}

	newSession := awssession.New(s3Config)

	s3Client := s3.New(newSession)

	f, err := os.Open(fileToUpload)
	if err != nil {
		return errors.Wrap(err, "failed to open archive file")
	}

	_, err = s3Client.PutObject(&s3.PutObjectInput{
		Body:   f,
		Bucket: bucket,
		Key:    key,
	})
	if err != nil {
		return errors.Wrap(err, "failed to upload to s3")
	}

	return nil
}

// GetAppVersionArchive will fetch the archive and return a string that contains a
// directory name where it's extracted into
func GetAppVersionArchive(appID string, sequence int) (string, error) {
	tmpDir, err := ioutil.TempDir("", "kotsadm")
	if err != nil {
		return "", errors.Wrap(err, "failed to create temp dir")
	}

	// Get the archive from object store

	forcePathStyle := false
	if os.Getenv("S3_BUCKET_ENDPOINT") == "true" {
		forcePathStyle = true
	}

	bucket := aws.String(os.Getenv("S3_BUCKET_NAME"))
	key := aws.String(fmt.Sprintf("%s/%d.tar.gz", appID, sequence))

	s3Config := &aws.Config{
		Credentials:      credentials.NewStaticCredentials(os.Getenv("S3_ACCESS_KEY_ID"), os.Getenv("S3_SECRET_ACCESS_KEY"), ""),
		Endpoint:         aws.String(os.Getenv("S3_ENDPOINT")),
		Region:           aws.String("us-east-1"),
		DisableSSL:       aws.Bool(true),
		S3ForcePathStyle: aws.Bool(forcePathStyle),
	}
	newSession := awssession.New(s3Config)

	tmpFile, err := ioutil.TempFile("", "kotsadm")
	if err != nil {
		return "", errors.Wrap(err, "failed to create temp file")
	}
	defer tmpFile.Close()
	defer os.RemoveAll(tmpFile.Name())

	downloader := s3manager.NewDownloader(newSession)
	_, err = downloader.Download(tmpFile,
		&s3.GetObjectInput{
			Bucket: bucket,
			Key:    key,
		})
	if err != nil {
		return "", errors.Wrap(err, "failed to download file")
	}

	tarGz := archiver.TarGz{
		Tar: &archiver.Tar{
			ImplicitTopLevelFolder: false,
		},
	}
	if err := tarGz.Unarchive(tmpFile.Name(), tmpDir); err != nil {
		return "", errors.Wrap(err, "failed to unarchive")
	}

	return tmpDir, nil
}

func ExtractArchiveToTempDirectory(archiveFilename string) (string, error) {
	tmpDir, err := ioutil.TempDir("", "kotsadm")
	if err != nil {
		return "", errors.Wrap(err, "failed to create temp dir")
	}

	tarGz := archiver.TarGz{
		Tar: &archiver.Tar{
			ImplicitTopLevelFolder: false,
		},
	}
	if err := tarGz.Unarchive(archiveFilename, tmpDir); err != nil {
		return "", errors.Wrap(err, "failed to unarchive")
	}

	return tmpDir, nil
}
