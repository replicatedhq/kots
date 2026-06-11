package reporting

import (
	"testing"

	"github.com/golang/mock/gomock"
	apptypes "github.com/replicatedhq/kots/pkg/app/types"
	mock_store "github.com/replicatedhq/kots/pkg/store/mock"
	"github.com/replicatedhq/kots/pkg/util"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8stypes "k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/fake"
)

func Test_compareEnvironmentFingerprints(t *testing.T) {
	tests := []struct {
		name    string
		stored  environmentFingerprint
		current environmentFingerprint
		want    fingerprintDecision
	}{
		{
			name:    "same cluster keeps instance id",
			stored:  environmentFingerprint{KubeSystemUID: "uid-a", PodNamespaceUID: "ns-1"},
			current: environmentFingerprint{KubeSystemUID: "uid-a", PodNamespaceUID: "ns-1"},
			want:    decisionKeep,
		},
		{
			name:    "different cluster regenerates instance id",
			stored:  environmentFingerprint{KubeSystemUID: "uid-a", PodNamespaceUID: "ns-1"},
			current: environmentFingerprint{KubeSystemUID: "uid-b", PodNamespaceUID: "ns-2"},
			want:    decisionRegenerate,
		},
		{
			name: "in-place DR: same cluster with recreated namespace keeps instance id",
			// the kotsadm namespace was deleted and restored in the same cluster
			stored:  environmentFingerprint{KubeSystemUID: "uid-a", PodNamespaceUID: "ns-1"},
			current: environmentFingerprint{KubeSystemUID: "uid-a", PodNamespaceUID: "ns-2"},
			want:    decisionKeep,
		},
		{
			name: "namespace-scoped install: same namespace keeps instance id",
			// RBAC prevents reading kube-system; fall back to the pod namespace UID
			stored:  environmentFingerprint{PodNamespaceUID: "ns-1"},
			current: environmentFingerprint{PodNamespaceUID: "ns-1"},
			want:    decisionKeep,
		},
		{
			name:    "namespace-scoped install: different namespace regenerates",
			stored:  environmentFingerprint{PodNamespaceUID: "ns-1"},
			current: environmentFingerprint{PodNamespaceUID: "ns-2"},
			want:    decisionRegenerate,
		},
		{
			name: "kube-system uid takes precedence over namespace uid when only stored has it missing",
			// stored fingerprint was recorded by a namespace-scoped install, current can read
			// kube-system; only the namespace UID is comparable
			stored:  environmentFingerprint{PodNamespaceUID: "ns-1"},
			current: environmentFingerprint{KubeSystemUID: "uid-a", PodNamespaceUID: "ns-1"},
			want:    decisionKeep,
		},
		{
			name:    "no comparable fields keeps instance id (fail safe)",
			stored:  environmentFingerprint{KubeSystemUID: "uid-a"},
			current: environmentFingerprint{PodNamespaceUID: "ns-1"},
			want:    decisionKeep,
		},
		{
			name:    "empty current fingerprint keeps instance id (fail safe)",
			stored:  environmentFingerprint{KubeSystemUID: "uid-a", PodNamespaceUID: "ns-1"},
			current: environmentFingerprint{},
			want:    decisionKeep,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := require.New(t)
			req.Equal(tt.want, compareEnvironmentFingerprints(tt.stored, tt.current))
		})
	}
}

func Test_checkForEnvironmentRestore(t *testing.T) {
	const (
		kubeSystemUID = "kube-system-uid-current"
		podNSUID      = "pod-ns-uid-current"
		podNamespace  = "test"
	)

	prevPodNamespace := util.PodNamespace
	util.PodNamespace = podNamespace
	t.Cleanup(func() { util.PodNamespace = prevPodNamespace })

	newClientset := func() *fake.Clientset {
		return fake.NewSimpleClientset(
			&corev1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name: "kube-system",
					UID:  k8stypes.UID(kubeSystemUID),
				},
			},
			&corev1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name: podNamespace,
					UID:  k8stypes.UID(podNSUID),
				},
			},
		)
	}

	currentFingerprintJSON := `{"kubeSystemUID":"` + kubeSystemUID + `","podNamespaceUID":"` + podNSUID + `"}`

	t.Run("no stored fingerprint adopts current without regenerating", func(t *testing.T) {
		req := require.New(t)
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		mockStore := mock_store.NewMockStore(ctrl)

		mockStore.EXPECT().GetEnvironmentFingerprint().Return("", nil)
		mockStore.EXPECT().SetEnvironmentFingerprint(currentFingerprintJSON).Return(nil)
		// no ListInstalledApps, no SetAppInstanceID

		err := checkForEnvironmentRestore(newClientset(), mockStore)
		req.NoError(err)
	})

	t.Run("matching fingerprint does not regenerate", func(t *testing.T) {
		req := require.New(t)
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		mockStore := mock_store.NewMockStore(ctrl)

		mockStore.EXPECT().GetEnvironmentFingerprint().Return(currentFingerprintJSON, nil)

		err := checkForEnvironmentRestore(newClientset(), mockStore)
		req.NoError(err)
	})

	t.Run("matching cluster with stale namespace uid refreshes fingerprint without regenerating", func(t *testing.T) {
		req := require.New(t)
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		mockStore := mock_store.NewMockStore(ctrl)

		stored := `{"kubeSystemUID":"` + kubeSystemUID + `","podNamespaceUID":"pod-ns-uid-old"}`
		mockStore.EXPECT().GetEnvironmentFingerprint().Return(stored, nil)
		mockStore.EXPECT().SetEnvironmentFingerprint(currentFingerprintJSON).Return(nil)

		err := checkForEnvironmentRestore(newClientset(), mockStore)
		req.NoError(err)
	})

	t.Run("restored into different cluster regenerates instance ids with lineage", func(t *testing.T) {
		req := require.New(t)
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		mockStore := mock_store.NewMockStore(ctrl)

		stored := `{"kubeSystemUID":"kube-system-uid-old","podNamespaceUID":"pod-ns-uid-old"}`
		mockStore.EXPECT().GetEnvironmentFingerprint().Return(stored, nil)
		mockStore.EXPECT().ListInstalledApps().Return([]*apptypes.App{{ID: "app-1"}}, nil)
		mockStore.EXPECT().GetAppInstanceID("app-1").Return("instance-old", []string{"instance-ancient"}, nil)
		mockStore.EXPECT().SetAppInstanceID("app-1", gomock.Not(gomock.Eq("instance-old")), []string{"instance-ancient", "instance-old"}).Return(nil)
		// fingerprint is only persisted after all apps were regenerated
		mockStore.EXPECT().SetEnvironmentFingerprint(currentFingerprintJSON).Return(nil)

		err := checkForEnvironmentRestore(newClientset(), mockStore)
		req.NoError(err)
	})

	t.Run("sequential restores extend the lineage chain", func(t *testing.T) {
		req := require.New(t)
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		mockStore := mock_store.NewMockStore(ctrl)

		stored := `{"kubeSystemUID":"kube-system-uid-old"}`
		mockStore.EXPECT().GetEnvironmentFingerprint().Return(stored, nil)
		mockStore.EXPECT().ListInstalledApps().Return([]*apptypes.App{{ID: "app-1"}}, nil)
		// app never regenerated before: instance id falls back to the app id, no lineage
		mockStore.EXPECT().GetAppInstanceID("app-1").Return("app-1", nil, nil)
		mockStore.EXPECT().SetAppInstanceID("app-1", gomock.Not(gomock.Eq("app-1")), []string{"app-1"}).Return(nil)
		mockStore.EXPECT().SetEnvironmentFingerprint(currentFingerprintJSON).Return(nil)

		err := checkForEnvironmentRestore(newClientset(), mockStore)
		req.NoError(err)
	})

	t.Run("unreadable environment fingerprint is a no-op", func(t *testing.T) {
		req := require.New(t)
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		mockStore := mock_store.NewMockStore(ctrl)

		// no namespaces readable at all
		clientset := fake.NewSimpleClientset()

		err := checkForEnvironmentRestore(clientset, mockStore)
		req.NoError(err)
	})
}
