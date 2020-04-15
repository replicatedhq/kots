package supportbundle

import (
	"encoding/json"
	"os"
	"path/filepath"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	awssession "github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/pkg/errors"
	"github.com/replicatedhq/kotsadm/pkg/persistence"
	"github.com/replicatedhq/kotsadm/pkg/supportbundle/types"
	"github.com/segmentio/ksuid"
)

func SetBundleStatus(id string, status string) error {
	db := persistence.MustGetPGSession()
	query := `update supportbundle set status = $1 where id = $2`

	_, err := db.Exec(query, status, id)
	if err != nil {
		return errors.Wrap(err, "failed to insert support bundle")
	}

	return nil
}

func CreateBundle(requestedID string, appID string, archivePath string) (*types.SupportBundle, error) {
	fi, err := os.Stat(archivePath)
	if err != nil {
		return nil, errors.Wrap(err, "failed to read archive")
	}

	id := ksuid.New().String()
	if requestedID != "" {
		id = requestedID
	}

	// upload the file to s3
	if err := uploadBundleToS3(id, archivePath); err != nil {
		return nil, errors.Wrap(err, "failed to upload to s3")
	}

	fileTree, err := archiveToFileTree(archivePath)
	if err != nil {
		return nil, errors.Wrap(err, "failed to generate file tree")
	}

	marshaledTree, err := json.Marshal(fileTree.Nodes)
	if err != nil {
		return nil, errors.Wrap(err, "failed to marshal tree index")
	}

	db := persistence.MustGetPGSession()
	query := `insert into supportbundle (id, slug, watch_id, size, status, created_at, tree_index) values ($1, $2, $3, $4, $5, $6, $7)`

	_, err = db.Exec(query, id, id, appID, fi.Size(), "uploaded", time.Now(), marshaledTree)
	if err != nil {
		return nil, errors.Wrap(err, "failed to insert support bundle")
	}

	return &types.SupportBundle{
		ID: id,
	}, nil
}

func uploadBundleToS3(id string, archivePath string) error {
	forcePathStyle := false
	if os.Getenv("S3_BUCKET_ENDPOINT") == "true" {
		forcePathStyle = true
	}

	bucket := aws.String(os.Getenv("S3_BUCKET_NAME"))
	key := aws.String(filepath.Join("supportbundles", id, "supportbundle.tar.gz"))

	s3Config := &aws.Config{
		Credentials:      credentials.NewStaticCredentials(os.Getenv("S3_ACCESS_KEY_ID"), os.Getenv("S3_SECRET_ACCESS_KEY"), ""),
		Endpoint:         aws.String(os.Getenv("S3_ENDPOINT")),
		Region:           aws.String("us-east-1"),
		DisableSSL:       aws.Bool(true),
		S3ForcePathStyle: aws.Bool(forcePathStyle),
	}

	newSession := awssession.New(s3Config)

	s3Client := s3.New(newSession)

	f, err := os.Open(archivePath)
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
