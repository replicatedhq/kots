package helm

import (
	"context"
	"testing"

	"helm.sh/helm/v3/pkg/release"
	corev1 "k8s.io/api/core/v1"
	kuberneteserrors "k8s.io/apimachinery/pkg/api/errors"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/fake"
)

const (
	kotsadmNamespace      = "kotsadm"
	helmReleaseNamespace  = "helm-release"
	helmReleaseSecretName = "sh.helm.release.v1.test.v1"
	helmReleaseName       = "test"
)

func mockKotsadmHelmReleaseSecretClient(t *testing.T) kubernetes.Interface {
	testReleaseSecret := buildHelmReleaseSecret(t)
	clientset := fake.NewSimpleClientset(
		testReleaseSecret,
	)
	return clientset
}

func buildHelmReleaseSecret(t *testing.T) *corev1.Secret {
	helmRelease := &release.Release{
		Name:      helmReleaseName,
		Namespace: kotsadmNamespace,
		Version:   1,
		Info: &release.Info{
			Status: release.StatusDeployed,
		},
	}
	encodedRelease, err := encodeRelease(helmRelease)
	if err != nil {
		t.Errorf("failed to encode helm release: %v", err)
	}
	return &corev1.Secret{
		Type: "helm.sh/release.v1",
		TypeMeta: v1.TypeMeta{
			Kind:       "Secret",
			APIVersion: "v1",
		},
		ObjectMeta: v1.ObjectMeta{
			Name:      helmReleaseSecretName,
			Namespace: kotsadmNamespace,
			Labels: map[string]string{
				"owner": "helm",
				"name":  helmReleaseName,
			},
		},
		Data: map[string][]byte{
			"release": []byte(encodedRelease),
		},
	}
}

func TestMigrateExistingHelmReleaseSecrets(t *testing.T) {
	type args struct {
		clientset        kubernetes.Interface
		releaseName      string
		releaseNamespace string
		kotsadmNamespace string
	}
	tests := []struct {
		name                     string
		args                     args
		wantErr                  bool
		wantMigratedSecretsCount int
	}{
		{
			name: "expect no error and migrate existing helm release secret",
			args: args{
				clientset:        mockKotsadmHelmReleaseSecretClient(t),
				releaseName:      helmReleaseName,
				releaseNamespace: helmReleaseNamespace,
				kotsadmNamespace: kotsadmNamespace,
			},
			wantErr:                  false,
			wantMigratedSecretsCount: 1,
		},
		{
			name: "expect no error when no helm release secret exists",
			args: args{
				clientset:        fake.NewSimpleClientset(),
				releaseName:      helmReleaseName,
				releaseNamespace: helmReleaseNamespace,
				kotsadmNamespace: kotsadmNamespace,
			},
			wantErr:                  false,
			wantMigratedSecretsCount: 0,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := MigrateExistingHelmReleaseSecrets(tt.args.clientset, tt.args.releaseName, tt.args.releaseNamespace, tt.args.kotsadmNamespace); (err != nil) != tt.wantErr {
				t.Errorf("MigrateExistingHelmReleaseSecrets() error = %v, wantErr %v", err, tt.wantErr)
			}

			if tt.wantMigratedSecretsCount == 0 {
				return
			}

			// verify that the secret was moved to the new namespace
			movedSecret, err := tt.args.clientset.CoreV1().Secrets(tt.args.releaseNamespace).Get(context.TODO(), helmReleaseSecretName, v1.GetOptions{})
			if err != nil {
				t.Errorf("failed to get helm release secret: %v", err)
			}

			if movedSecret.Namespace != tt.args.releaseNamespace {
				t.Errorf("expected helm release secret to be in namespace %s, but was in %s", tt.args.releaseNamespace, movedSecret.Namespace)
			}

			release, err := HelmReleaseFromSecretData(movedSecret.Data["release"])
			if err != nil {
				t.Errorf("failed to get helm release from secret data: %v", err)
			}

			if release.Namespace != tt.args.releaseNamespace {
				t.Errorf("expected helm release to be in namespace %s, but was in %s", tt.args.releaseNamespace, release.Namespace)
			}

			_, err = tt.args.clientset.CoreV1().Secrets(tt.args.kotsadmNamespace).Get(context.TODO(), tt.args.releaseName, v1.GetOptions{})
			if err == nil {
				t.Errorf("expected helm release secret to be deleted from %s, but it was not", tt.args.kotsadmNamespace)
			}
			if !kuberneteserrors.IsNotFound(err) {
				t.Errorf("expected helm release secret to be deleted from %s, but got error: %v", kotsadmNamespace, err)
			}
		})
	}
}
