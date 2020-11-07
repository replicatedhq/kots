package ocistore

import (
	"context"
	"os"

	"github.com/ocidb/ocidb/pkg/ocidb"
	ocidbtypes "github.com/ocidb/ocidb/pkg/ocidb/types"
	"github.com/pkg/errors"
	"github.com/replicatedhq/kots/kotsadm/pkg/store/ocistore/tables"
	schemasv1alpha4 "github.com/schemahero/schemahero/pkg/apis/schemas/v1alpha4"
	corev1 "k8s.io/api/core/v1"
	kuberneteserrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
)

/* OCIStore stores most data in an OCI compatible image repository,
   but does not make guarantees that every thing is sted there.
   Some data is stored locally in Kuberntes ConfigMaps and Secrets
   to speed up retrieval

   A note about "transactions": in the pg store, there were a few
   places that relied on transactions to ensure integrity
   Here, this is stored in configmaps and secrets, and this inegrity
   is provided by the Kubernetes API's enforcement of puts.
   If a caller GETs a configmap, updates it and then tries to PUT that
   configmap, but another process has modified it, the PUT will
   be rejected. This level of consistency is all that's needed for KOTS
*/
type OCIStore struct {
	ocidbtypes.ConnectOpts
	connection *ocidbtypes.Connection
}

var (
	ErrNotFound       = errors.New("not found")
	ErrNotImplemented = errors.New("not implemented in ocistore")
)

func (s OCIStore) Init() error {
	return nil
}

func (s OCIStore) WaitForReady(ctx context.Context) error {
	return nil
}

func (s OCIStore) IsNotFound(err error) bool {
	if err == nil {
		return false
	}
	return errors.Cause(err) == ErrNotFound
}

func StoreFromEnv() OCIStore {
	connectOpts := ocidbtypes.ConnectOpts{
		Host:      "kotsadm-storage-registry",
		Port:      5000,
		Namespace: "",
		Username:  "",
		Password:  "",
		Database:  "kots",
		Tables:    getSQLiteTables(),
	}

	connection, err := ocidb.Connect(context.TODO(), &connectOpts)
	if err != nil {
		panic(err)
	}

	return OCIStore{
		connectOpts,
		connection,
	}
}

func (c OCIStore) GetClientset() (*kubernetes.Clientset, error) {
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

func (s OCIStore) getSecret(name string) (*corev1.Secret, error) {
	clientset, err := s.GetClientset()
	if err != nil {
		return nil, errors.Wrap(err, "failed to get clientset")
	}

	existingSecret, err := clientset.CoreV1().Secrets(os.Getenv("POD_NAMESPACE")).Get(context.TODO(), name, metav1.GetOptions{})
	if err != nil && !kuberneteserrors.IsNotFound(err) {
		return nil, errors.Wrap(err, "failed to get secret")
	} else if kuberneteserrors.IsNotFound(err) {
		secret := corev1.Secret{
			TypeMeta: metav1.TypeMeta{
				APIVersion: "v1",
				Kind:       "Secret",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      name,
				Namespace: os.Getenv("POD_NAMESPACE"),
				Labels: map[string]string{
					"owner": "kotsadm",
				},
			},
			Data: map[string][]byte{},
		}

		createdSecret, err := clientset.CoreV1().Secrets(os.Getenv("POD_NAMESPACE")).Create(context.TODO(), &secret, metav1.CreateOptions{})
		if err != nil {
			return nil, errors.Wrap(err, "failed to create secret")
		}

		return createdSecret, nil
	}

	return existingSecret, nil
}

func (s OCIStore) updateSecret(secret *corev1.Secret) error {
	clientset, err := s.GetClientset()
	if err != nil {
		return errors.Wrap(err, "failed to get clientset")
	}

	_, err = clientset.CoreV1().Secrets(os.Getenv("POD_NAMESPACE")).Update(context.Background(), secret, metav1.UpdateOptions{})
	if err != nil {
		return errors.Wrap(err, "failed to update config map")
	}

	return nil
}

func getSQLiteTables() []schemasv1alpha4.TableSpec {
	return []schemasv1alpha4.TableSpec{
		tables.APITaskStatus(),
		tables.AppDownstreamOutput(),
		tables.AppDownstreamVersion(),
		tables.AppDownstream(),
		tables.AppStatus(),
		tables.AppVersion(),
		tables.App(),
		tables.Cluster(),
		tables.KotsadmParams(),
		tables.ObjectStore(),
		tables.PendingSupportBundle(),
		tables.PreflightResult(),
		tables.PreflightSpec(),
		tables.ScheduledInstanceSnapshots(),
		tables.ScheduledSnapshots(),
		tables.ShipUserLocal(),
		tables.ShipUser(),
		tables.SupportBundleAnalysis(),
		tables.SupportBundle(),
		tables.UserApp(),
		tables.UserCluster(),
	}
}
