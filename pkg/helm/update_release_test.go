package helm

import (
	"context"
	"fmt"
	"testing"

	"helm.sh/helm/v3/pkg/release"
	corev1 "k8s.io/api/core/v1"
	kuberneteserrors "k8s.io/apimachinery/pkg/api/errors"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/fake"
)

const (
	kotsadmNamespace     = "kotsadm"
	helmReleaseNamespace = "helm-release"
	helmReleaseName      = "test"
)

func mockKotsadmHelmReleaseSecretClient(t *testing.T, kotsadmNS string, releaseName string) kubernetes.Interface {
	kotsadmReleaseSecret := buildHelmReleaseSecret(t, kotsadmNS, releaseName)
	clientset := fake.NewSimpleClientset(
		kotsadmReleaseSecret,
	)
	return clientset
}

func mockKotsadmHelmReleaseSecretExistsClient(t *testing.T, kotsadmNS string, releaseNS string, releaseName string) kubernetes.Interface {
	kotsadmReleaseSecret := buildHelmReleaseSecret(t, kotsadmNS, releaseName)
	helmReleaseSecret := buildHelmReleaseSecret(t, releaseNS, releaseName)
	clientset := fake.NewSimpleClientset(
		kotsadmReleaseSecret,
		helmReleaseSecret,
	)
	return clientset
}

func buildHelmReleaseSecret(t *testing.T, kotsadmNS string, releaseName string) *corev1.Secret {
	helmRelease := &release.Release{
		Name:      releaseName,
		Namespace: kotsadmNS,
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
			Name:      fmt.Sprintf("sh.helm.release.v1.%s.v1", releaseName),
			Namespace: kotsadmNS,
			Labels: map[string]string{
				"owner": "helm",
				"name":  releaseName,
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
				clientset:        mockKotsadmHelmReleaseSecretClient(t, kotsadmNamespace, helmReleaseName),
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
		}, {
			name: "expect no error when helm release secret exists in the release namespace",
			args: args{
				clientset:        mockKotsadmHelmReleaseSecretExistsClient(t, kotsadmNamespace, helmReleaseNamespace, helmReleaseName),
				releaseName:      helmReleaseName,
				releaseNamespace: kotsadmNamespace,
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

			if tt.wantErr {
				return
			}

			// verify that the secret was moved to the new namespace
			movedSecret, err := tt.args.clientset.CoreV1().Secrets(tt.args.releaseNamespace).List(context.TODO(), v1.ListOptions{LabelSelector: fmt.Sprintf("owner=helm,name=%s", tt.args.releaseName)})
			if err != nil {
				t.Errorf("failed to get helm release secret: %v", err)
			}

			if len(movedSecret.Items) != tt.wantMigratedSecretsCount {
				t.Errorf("expected %d helm release secret to be moved, but found %d", tt.wantMigratedSecretsCount, len(movedSecret.Items))
			}

			for _, secret := range movedSecret.Items {
				if secret.Namespace != tt.args.releaseNamespace {
					t.Errorf("expected helm release secret to be in namespace %s, but was in %s", tt.args.releaseNamespace, secret.Namespace)
				}

				release, err := helmReleaseFromSecretData(secret.Data["release"])
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
					t.Errorf("expected helm release secret to be deleted from %s, but got error %v", tt.args.kotsadmNamespace, err)
				}
			}
		})
	}
}
