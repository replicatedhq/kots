package kotsstore

import (
	"context"
	"database/sql"
	"log"
	"os"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/pkg/errors"
	kotsscheme "github.com/replicatedhq/kots/kotskinds/client/kotsclientset/scheme"
	"github.com/replicatedhq/kots/pkg/k8sutil"
	kotsadmtypes "github.com/replicatedhq/kots/pkg/kotsadm/types"
	"github.com/replicatedhq/kots/pkg/logger"
	"github.com/replicatedhq/kots/pkg/persistence"
	troubleshootscheme "github.com/replicatedhq/troubleshoot/pkg/client/troubleshootclientset/scheme"
	veleroscheme "github.com/vmware-tanzu/velero/pkg/generated/clientset/versioned/scheme"
	corev1 "k8s.io/api/core/v1"
	kuberneteserrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/scheme"
)

var (
	ErrNotFound = errors.New("not found")
)

type cachedTaskStatus struct {
	expirationTime time.Time
	taskStatus     taskStatus
}

type KOTSStore struct {
	sessionSecret     *corev1.Secret
	sessionExpiration time.Time

	cachedTaskStatus map[string]*cachedTaskStatus
}

func init() {
	kotsscheme.AddToScheme(scheme.Scheme)
	veleroscheme.AddToScheme(scheme.Scheme)
	troubleshootscheme.AddToScheme(scheme.Scheme)
}

func (s *KOTSStore) WaitForReady(ctx context.Context) error {
	errCh := make(chan error, 2)

	go func() {
		errCh <- waitForPostgres(ctx)
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

func (s *KOTSStore) IsNotFound(err error) bool {
	if err == nil {
		return false
	}

	cause := errors.Cause(err)
	if cause == sql.ErrNoRows {
		return true
	}

	if cause == ErrNotFound {
		return true
	}

	if os.IsNotExist(cause) {
		return true
	}

	if err, ok := cause.(awserr.Error); ok {
		switch err.Code() {
		case "NotFound", "NoSuchKey":
			return true
		default:
			return false
		}
	}

	if kuberneteserrors.IsNotFound(cause) {
		return true
	}

	return false
}

func canIgnoreEtcdError(err error) bool {
	if err == nil {
		return true
	}

	if strings.Contains(err.Error(), "connection refused") {
		return true
	}

	if strings.Contains(err.Error(), "request timed out") {
		return true
	}

	if strings.Contains(err.Error(), "EOF") {
		return true
	}

	return false
}

func StoreFromEnv() *KOTSStore {
	return &KOTSStore{
		cachedTaskStatus: make(map[string]*cachedTaskStatus),
	}
}

func (s *KOTSStore) getConfigmap(name string) (*corev1.ConfigMap, error) {
	clientset, err := k8sutil.GetClientset()
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

func (s *KOTSStore) updateConfigmap(configmap *corev1.ConfigMap) error {
	clientset, err := k8sutil.GetClientset()
	if err != nil {
		return errors.Wrap(err, "failed to get clientset")
	}

	_, err = clientset.CoreV1().ConfigMaps(os.Getenv("POD_NAMESPACE")).Update(context.Background(), configmap, metav1.UpdateOptions{})
	if err != nil {
		return errors.Wrap(err, "failed to update config map")
	}

	return nil
}
