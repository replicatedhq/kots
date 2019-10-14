package ship

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/hashicorp/go-uuid"
	"github.com/pkg/errors"
	"github.com/replicatedhq/kotsadm/worker/pkg/config"
)

type StateManager struct {
	c  *config.Config
	s3 *s3.S3
}

type S3State struct {
	ID     string
	PutURL string
	GetURL string
}

func NewStateManager(c *config.Config) (*StateManager, error) {
	sess, err := session.NewSession(getS3Config(c))
	if err != nil {
		return nil, errors.Wrap(err, "new aws session")
	}
	return &StateManager{c: c, s3: s3.New(sess)}, nil
}

func (m *StateManager) NewStateID() (string, error) {
	id, err := uuid.GenerateUUID()
	if err != nil {
		return "", errors.Wrap(err, "generate uuid")
	}

	return id, nil
}

func (m *StateManager) PutState(stateID string, stateJSON []byte) error {
	_, err := m.s3.PutObject(&s3.PutObjectInput{
		Bucket: aws.String(strings.TrimSpace(m.c.S3BucketName)),
		Key:    aws.String(fmt.Sprintf("/state/%s.json", stateID)),
		Body:   bytes.NewReader(stateJSON),
	})

	return errors.Wrap(err, "put state to s3")
}

func (m *StateManager) GetState(stateID string) ([]byte, error) {
	result, err := m.s3.GetObject(&s3.GetObjectInput{
		Bucket: aws.String(strings.TrimSpace(m.c.S3BucketName)),
		Key:    aws.String(fmt.Sprintf("/state/%s.json", stateID)),
	})
	if err != nil {
		return nil, errors.Wrap(err, "get state from s3")
	}
	defer result.Body.Close()

	var stateJSON bytes.Buffer
	if _, err := io.Copy(&stateJSON, result.Body); err != nil {
		return nil, err
	}
	return stateJSON.Bytes(), nil
}

func (m *StateManager) DeleteState(stateID string) error {
	_, err := m.s3.DeleteObject(&s3.DeleteObjectInput{
		Bucket: aws.String(strings.TrimSpace(m.c.S3BucketName)),
		Key:    aws.String(fmt.Sprintf("/state/%s.json", stateID)),
	})
	if err != nil {
		return errors.Wrap(err, "get state from s3")
	}

	return nil
}

func (m *StateManager) GetPresignedURLs(stateID string) (*S3State, error) {
	objectKey := fmt.Sprintf("/state/%s.json", stateID)

	putResp, _ := m.s3.PutObjectRequest(&s3.PutObjectInput{
		Bucket: aws.String(strings.TrimSpace(m.c.S3BucketName)),
		Key:    aws.String(objectKey),
	})

	putURL, err := putResp.Presign(60 * time.Minute)
	if err != nil {
		return nil, errors.Wrap(err, "presign response")
	}

	getResp, _ := m.s3.GetObjectRequest(&s3.GetObjectInput{
		Bucket: aws.String(strings.TrimSpace(m.c.S3BucketName)),
		Key:    aws.String(objectKey),
	})

	getURL, err := getResp.Presign(30 * time.Minute)
	if err != nil {
		return nil, errors.Wrap(err, "presign response")
	}

	return &S3State{
		ID:     stateID,
		PutURL: putURL,
		GetURL: getURL,
	}, nil
}

func (m *StateManager) CreateS3State(stateJSON []byte) (*S3State, error) {
	stateID, err := m.NewStateID()
	if err != nil {
		return nil, errors.Wrap(err, "create state ID")
	}

	if err := m.PutState(stateID, stateJSON); err != nil {
		return nil, errors.Wrap(err, "create s3 state")
	}

	s3State, err := m.GetPresignedURLs(stateID)
	if err != nil {
		return nil, errors.Wrap(err, "sign state URLs")
	}

	return s3State, nil
}

func getS3Config(c *config.Config) *aws.Config {
	region := "us-east-1"
	if os.Getenv("AWS_REGION") != "" {
		region = os.Getenv("AWS_REGION")
	}

	s3config := &aws.Config{
		Region: aws.String(region),
	}

	if strings.TrimSpace(c.S3Endpoint) != "" {
		s3config.Endpoint = aws.String(strings.TrimSpace(c.S3Endpoint))
	}

	if strings.TrimSpace(c.S3AccessKeyID) != "" && strings.TrimSpace(c.S3SecretAccessKey) != "" {
		s3config.Credentials = credentials.NewStaticCredentials(strings.TrimSpace(c.S3AccessKeyID), strings.TrimSpace(c.S3SecretAccessKey), "")
	}

	if strings.TrimSpace(c.S3BucketEndpoint) != "" {
		s3config.S3ForcePathStyle = aws.Bool(true)
	}

	return s3config
}
