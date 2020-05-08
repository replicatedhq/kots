package supportbundle

import (
	"encoding/json"
	"os"
	"path/filepath"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	awssession "github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/pkg/errors"
	"github.com/replicatedhq/kots/kotsadm/pkg/persistence"
	kotss3 "github.com/replicatedhq/kots/kotsadm/pkg/s3"
	"github.com/replicatedhq/kots/kotsadm/pkg/supportbundle/types"
	"github.com/segmentio/ksuid"
)

func SetBundleAnalysis(id string, insights []byte) error {
	db := persistence.MustGetPGSession()
	query := `update supportbundle set status = $1 where id = $2`

	_, err := db.Exec(query, "analyzed", id)
	if err != nil {
		return errors.Wrap(err, "failed to insert support bundle")
	}

	query = `insert into supportbundle_analysis (id, supportbundle_id, error, max_severity, insights, created_at) values ($1, $2, null, null, $3, $4)`
	_, err = db.Exec(query, ksuid.New().String(), id, insights, time.Now())
	if err != nil {
		return errors.Wrap(err, "failed to insert insights")
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
	bucket := aws.String(os.Getenv("S3_BUCKET_NAME"))
	key := aws.String(filepath.Join("supportbundles", id, "supportbundle.tar.gz"))

	newSession := awssession.New(kotss3.GetConfig())

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
