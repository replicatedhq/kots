package kotsstore

import (
	"context"
	"log"
	"os"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/pkg/errors"
	"github.com/replicatedhq/kots/pkg/filestore"
	"github.com/replicatedhq/kots/pkg/k8sutil"
	kotsadmtypes "github.com/replicatedhq/kots/pkg/kotsadm/types"
	"github.com/replicatedhq/kots/pkg/logger"
	"github.com/replicatedhq/kots/pkg/persistence"
	"github.com/replicatedhq/kots/pkg/util"
	kotsscheme "github.com/replicatedhq/kotskinds/client/kotsclientset/scheme"
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
	taskStatus     TaskStatus
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

func (s *KOTSStore) Init() error {
	if err := filestore.GetStore().Init(); err != nil {
		return errors.Wrap(err, "failed to initialize the file store")
	}

	return nil
}

func (s *KOTSStore) WaitForReady(ctx context.Context) error {
	errCh := make(chan error, 2)

	go func() {
		errCh <- waitForRqlite(ctx)
	}()

	go func() {
		errCh <- filestore.GetStore().WaitForReady(ctx)
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

func waitForRqlite(ctx context.Context) error {
	logger.Debug("waiting for database to be ready")

	period := 1 * time.Second // TODO: backoff
	for {
		db := persistence.MustGetDBSession()

		// any SQL will do. just need tables to be created.
		query := `select count(1) from app`
		rows, err := db.QueryOne(query)
		if err == nil && rows.Next() {
			var count int
			err := rows.Scan(&count)
			if err == nil {
				logger.Debug("database is ready")
				return nil
			}
		}

		select {
		case <-time.After(period):
			continue
		case <-ctx.Done():
			if rows.Err != nil {
				return errors.Wrap(rows.Err, "failed to query database")
			}
			return errors.Wrap(err, "failed to find valid database")
		}
	}
}

func (s *KOTSStore) IsNotFound(err error) bool {
	if err == nil {
		return false
	}

	cause := errors.Cause(err)
	if cause == ErrNotFound || cause == filestore.ErrNotFound {
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

	existingConfigmap, err := clientset.CoreV1().ConfigMaps(util.PodNamespace).Get(context.TODO(), name, metav1.GetOptions{})
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
				Namespace: util.PodNamespace,
				Labels:    kotsadmtypes.GetKotsadmLabels(),
			},
			Data: map[string]string{},
		}

		createdConfigmap, err := clientset.CoreV1().ConfigMaps(util.PodNamespace).Create(context.TODO(), &configmap, metav1.CreateOptions{})
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

	_, err = clientset.CoreV1().ConfigMaps(util.PodNamespace).Update(context.Background(), configmap, metav1.UpdateOptions{})
	if err != nil {
		return errors.Wrap(err, "failed to update config map")
	}

	return nil
}
