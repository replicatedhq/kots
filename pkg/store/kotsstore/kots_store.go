package kotsstore

import (
	"context"
	"database/sql"
	"log"
	"os"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	awssession "github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/pkg/errors"
	kotsscheme "github.com/replicatedhq/kots/kotskinds/client/kotsclientset/scheme"
	kotsadmtypes "github.com/replicatedhq/kots/pkg/kotsadm/types"
	"github.com/replicatedhq/kots/pkg/logger"
	"github.com/replicatedhq/kots/pkg/persistence"
	kotss3 "github.com/replicatedhq/kots/pkg/s3"
	troubleshootscheme "github.com/replicatedhq/troubleshoot/pkg/client/troubleshootclientset/scheme"
	veleroscheme "github.com/vmware-tanzu/velero/pkg/generated/clientset/versioned/scheme"
	corev1 "k8s.io/api/core/v1"
	kuberneteserrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
)

var (
	ErrNotFound = errors.New("not found")
)

type KOTSStore struct {
}

func init() {
	kotsscheme.AddToScheme(scheme.Scheme)
	veleroscheme.AddToScheme(scheme.Scheme)
	troubleshootscheme.AddToScheme(scheme.Scheme)
}

func (s KOTSStore) Init() error {
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

func (s KOTSStore) WaitForReady(ctx context.Context) error {
	errCh := make(chan error, 2)

	go func() {
		errCh <- waitForPostgres(ctx)
	}()

	go func() {
		errCh <- waitForS3(ctx)
	}()

	isError := false
	for i := 0; i < 2; i++ {
		err := <-errCh
		if err != nil {
			log.Println(err.Error())
			isError = true
			break
		}
	}

	if isError {
		return errors.New("failed to wait for dependencies")
	}

	return nil
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
		err := row.Scan(&count)
		if err == nil {
			logger.Debug("database is ready")
			return nil
		}

		select {
		case <-time.After(period):
			continue
		case <-ctx.Done():
			return errors.Wrap(err, "failed to find valid database")
		}
	}
}

func waitForS3(ctx context.Context) error {
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

func (s KOTSStore) IsNotFound(err error) bool {
	if errors.Cause(err) == sql.ErrNoRows {
		return true
	}
	if errors.Cause(err) == ErrNotFound {
		return true
	}
	return false
}

func (c KOTSStore) GetClientset() (*kubernetes.Clientset, error) {
	cfg, err := config.GetConfig()
	if err != nil {
		return nil, errors.Wrap(err, "failed to get cluster config")
	}

	clientset, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create kubernetes clientset")
	}

	return clientset, nil
}

func (s KOTSStore) getConfigmap(name string) (*corev1.ConfigMap, error) {
	clientset, err := s.GetClientset()
	if err != nil {
		return nil, errors.Wrap(err, "failed to get clientset")
	}

	existingConfigmap, err := clientset.CoreV1().ConfigMaps(os.Getenv("POD_NAMESPACE")).Get(context.TODO(), name, metav1.GetOptions{})
	if err != nil && !kuberneteserrors.IsNotFound(err) {
		return nil, errors.Wrap(err, "failed to get configmap")
	} else if kuberneteserrors.IsNotFound(err) {
		configmap := corev1.ConfigMap{
			TypeMeta: metav1.TypeMeta{
				APIVersion: "v1",
				Kind:       "ConfigMap",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      name,
				Namespace: os.Getenv("POD_NAMESPACE"),
				Labels:    kotsadmtypes.GetKotsadmLabels(),
			},
			Data: map[string]string{},
		}

		createdConfigmap, err := clientset.CoreV1().ConfigMaps(os.Getenv("POD_NAMESPACE")).Create(context.TODO(), &configmap, metav1.CreateOptions{})
		if err != nil {
			return nil, errors.Wrap(err, "failed to create configmap")
		}

		return createdConfigmap, nil
	}

	return existingConfigmap, nil
}

func (s KOTSStore) updateConfigmap(configmap *corev1.ConfigMap) error {
	clientset, err := s.GetClientset()
	if err != nil {
		return errors.Wrap(err, "failed to get clientset")
	}

	_, err = clientset.CoreV1().ConfigMaps(os.Getenv("POD_NAMESPACE")).Update(context.Background(), configmap, metav1.UpdateOptions{})
	if err != nil {
		return errors.Wrap(err, "failed to update config map")
	}

	return nil
}
