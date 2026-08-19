package supportbundle

import (
	"context"
	"testing"

	kotstypes "github.com/replicatedhq/kots/pkg/kotsadm/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	testclient "k8s.io/client-go/kubernetes/fake"
)

func TestCleanupLegacySpecConfigMaps(t *testing.T) {
	namespace := "kotsadm"
	label := map[string]string{kotstypes.TroubleshootKey: kotstypes.TroubleshootValue}

	objects := []runtime.Object{
		// Legacy sub-spec ConfigMaps that should be deleted.
		&corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{Name: "kotsadm-my-app-supportbundle-vendor", Namespace: namespace, Labels: label},
		},
		&corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{Name: "kotsadm-my-app-supportbundle-cluster-specific", Namespace: namespace, Labels: label},
		},
		&corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{Name: "kotsadm-my-app-supportbundle-default", Namespace: namespace, Labels: label},
		},
		// A merged support bundle ConfigMap (should not match the sub-spec pattern).
		&corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{Name: "kotsadm-my-app-supportbundle", Namespace: namespace, Labels: label},
		},
		// A non-kotsadm ConfigMap (should not be deleted).
		&corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{Name: "cluster-wide-supportbundle", Namespace: "another", Labels: label},
		},
		// A legacy sub-spec ConfigMap without the troubleshoot label (should not be deleted by label selector).
		&corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{Name: "kotsadm-your-app-supportbundle-vendor", Namespace: namespace},
		},
		// A redactor ConfigMap (should not be deleted).
		&corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{Name: "kotsadm-my-app-redact-spec", Namespace: namespace, Labels: label},
		},
		// A Secret with a legacy sub-spec name (should not be deleted because it is not a ConfigMap).
		&corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{Name: "kotsadm-my-app-supportbundle-vendor", Namespace: namespace, Labels: label},
		},
	}

	clientset := testclient.NewSimpleClientset(objects...)

	err := CleanupLegacySpecConfigMaps(context.TODO(), clientset, namespace)
	require.NoError(t, err)

	configmaps, err := clientset.CoreV1().ConfigMaps(namespace).List(context.TODO(), metav1.ListOptions{})
	require.NoError(t, err)

	remainingNames := []string{}
	for _, cm := range configmaps.Items {
		remainingNames = append(remainingNames, cm.Name)
	}

	assert.ElementsMatch(t, []string{
		"kotsadm-my-app-supportbundle",
		"kotsadm-your-app-supportbundle-vendor",
		"kotsadm-my-app-redact-spec",
	}, remainingNames)

	// Verify the Secret still exists.
	_, err = clientset.CoreV1().Secrets(namespace).Get(context.TODO(), "kotsadm-my-app-supportbundle-vendor", metav1.GetOptions{})
	require.NoError(t, err)

	// Verify the non-kotsadm ConfigMap in another namespace still exists.
	_, err = clientset.CoreV1().ConfigMaps("another").Get(context.TODO(), "cluster-wide-supportbundle", metav1.GetOptions{})
	require.NoError(t, err)
}
