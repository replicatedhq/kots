package gitops

import (
	"encoding/base64"
	"testing"

	"github.com/stretchr/testify/assert"
	"golang.org/x/crypto/ssh"
	corev1 "k8s.io/api/core/v1"
	kuberneteserrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/fake"
	core "k8s.io/client-go/testing"
)

func Test_createGitOps(t *testing.T) {
	tests := []struct {
		name        string
		appID       string
		clusterID   string
		provider    string
		repoURI     string
		hostname    string
		httpPort    string
		sshPort     string
		configIndex int64
		action      string
		branch      string
		format      string
		path        string
		wantKeyType string
	}{
		{
			name:        "gitlab provider",
			provider:    "gitlab",
			repoURI:     "https://1.2.3.6/test_org/test_repo",
			hostname:    "1.2.3.6",
			httpPort:    "",
			sshPort:     "",
			configIndex: 0,
			action:      "commit",
			branch:      "test0-branch",
			format:      "single",
			path:        "/test/path/0",
			wantKeyType: "ssh-ed25519",
		},
		{
			name:        "github provider",
			provider:    "github",
			repoURI:     "https://1.2.3.4/test_org/test_repo",
			hostname:    "1.2.3.4",
			httpPort:    "",
			sshPort:     "",
			configIndex: 0,
			action:      "commit",
			branch:      "test1-branch",
			format:      "single",
			path:        "/test/path/1",
			wantKeyType: "ssh-ed25519",
		},
		{
			name:        "github enterprise provider",
			provider:    "github_enterprise",
			repoURI:     "https://1.2.3.5/test_org/test_repo",
			hostname:    "1.2.3.5",
			httpPort:    "",
			sshPort:     "",
			configIndex: 1,
			action:      "commit",
			branch:      "test2-branch",
			format:      "single",
			path:        "/test/path/2",
			wantKeyType: "ssh-ed25519",
		},
	}

	clientset := fake.NewSimpleClientset()
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := createGitOps(clientset, test.provider, test.repoURI, test.hostname, test.httpPort, test.sshPort)
			assert.NoError(t, err)

			err = updateDownstreamGitOps(clientset, test.appID, test.clusterID, test.repoURI, test.branch, test.path, test.format, test.action)
			assert.NoError(t, err)

			config, err := GetDownstreamGitOpsConfig(clientset, test.appID, test.clusterID)
			assert.NoError(t, err)

			assert.Equal(t, test.provider, config.Provider)
			assert.Equal(t, test.repoURI, config.RepoURI)
			assert.Equal(t, test.hostname, config.Hostname)
			assert.Equal(t, test.httpPort, config.HTTPPort)
			assert.Equal(t, test.sshPort, config.SSHPort)
			assert.Equal(t, test.action, config.Action)
			assert.Equal(t, test.branch, config.Branch)
			assert.Equal(t, test.format, config.Format)
			assert.Equal(t, test.path, config.Path)

			publicKey, _, _, _, err := ssh.ParseAuthorizedKey([]byte(config.PublicKey))
			assert.NoError(t, err)

			assert.Equal(t, test.wantKeyType, publicKey.Type())
		})
	}
}

func mockGitOpsConfigMapNotFoundClient() kubernetes.Interface {
	mockClient := fake.Clientset{}
	mockClient.AddReactor("get", "configmaps", func(action core.Action) (bool, runtime.Object, error) {
		return true, nil, kuberneteserrors.NewNotFound(corev1.Resource("configmap"), "kotsadm-gitops")
	})
	return &mockClient
}

func mockGitOpsConfigMapFoundAndUpdatedClient() kubernetes.Interface {
	encodedAppConfig := base64.StdEncoding.EncodeToString([]byte(`{}`))

	mockClient := fake.Clientset{}
	mockClient.AddReactor("get", "configmaps", func(action core.Action) (bool, runtime.Object, error) {
		return true, &corev1.ConfigMap{
			Data: map[string]string{
				"test-app-test-cluster": encodedAppConfig,
			},
		}, nil
	})
	mockClient.AddReactor("update", "configmaps", func(action core.Action) (bool, runtime.Object, error) {
		return true, nil, nil
	})
	return &mockClient
}

func mockGitOpsConfigMapUpdateFailedClient() kubernetes.Interface {
	encodedAppConfig := base64.StdEncoding.EncodeToString([]byte(`{}`))

	mockClient := fake.Clientset{}
	mockClient.AddReactor("get", "configmaps", func(action core.Action) (bool, runtime.Object, error) {
		return true, &corev1.ConfigMap{
			Data: map[string]string{
				"test-app-test-cluster": encodedAppConfig,
			},
		}, nil
	})
	mockClient.AddReactor("update", "configmaps", func(action core.Action) (bool, runtime.Object, error) {
		return true, nil, kuberneteserrors.NewGone("kotsadm-gitops")
	})
	return &mockClient
}

func Test_deleteDownstreamGitOps(t *testing.T) {
	type args struct {
		clientset kubernetes.Interface
		appID     string
		clusterID string
		repoURI   string
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "expect error when configmap is not found",
			args: args{
				clientset: mockGitOpsConfigMapNotFoundClient(),
				appID:     "test-app",
				clusterID: "test-cluster",
			},
			wantErr: true,
		},
		{
			name: "expect no error when configmap is found and updated",
			args: args{
				clientset: mockGitOpsConfigMapFoundAndUpdatedClient(),
				appID:     "test-app",
				clusterID: "test-cluster",
			},
			wantErr: false,
		},
		{
			name: "expect error when configmap is found but update failed",
			args: args{
				clientset: mockGitOpsConfigMapUpdateFailedClient(),
				appID:     "test-app",
				clusterID: "test-cluster",
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := deleteDownstreamGitOps(tt.args.clientset, tt.args.appID, tt.args.clusterID, tt.args.repoURI); (err != nil) != tt.wantErr {
				t.Errorf("deleteDownstreamGitOps() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func mockGitOpsSecretNotFoundClient() kubernetes.Interface {
	mockClient := fake.Clientset{}
	mockClient.AddReactor("get", "secrets", func(action core.Action) (bool, runtime.Object, error) {
		return true, nil, kuberneteserrors.NewNotFound(corev1.Resource("secrets"), "kotsadm-gitops")
	})
	return &mockClient
}

func mockGitOpsSecretFoundAndUpdatedClient() kubernetes.Interface {
	mockClient := fake.Clientset{}
	mockClient.AddReactor("get", "secrets", func(action core.Action) (bool, runtime.Object, error) {
		return true, &corev1.Secret{
			Data: map[string][]byte{
				"provider.0.repoUri": []byte("test-repo"),
			},
		}, nil
	})
	mockClient.AddReactor("update", "secrets", func(action core.Action) (bool, runtime.Object, error) {
		return true, nil, nil
	})
	return &mockClient
}

func mockGitOpsSecretUpdateFailedClient() kubernetes.Interface {
	mockClient := fake.Clientset{}
	mockClient.AddReactor("get", "secrets", func(action core.Action) (bool, runtime.Object, error) {
		return true, &corev1.Secret{
			Data: map[string][]byte{
				"provider.0.repoUri": []byte("test-repo"),
			},
		}, nil
	})
	mockClient.AddReactor("update", "secrets", func(action core.Action) (bool, runtime.Object, error) {
		return true, nil, kuberneteserrors.NewGone("kotsadm-gitops")
	})
	return &mockClient
}

func Test_deleteKeysFromGitOpsSecret(t *testing.T) {
	type args struct {
		clientset kubernetes.Interface
		repoURL   string
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "expect error when secret is not found",
			args: args{
				clientset: mockGitOpsSecretNotFoundClient(),
				repoURL:   "test-repo",
			},
			wantErr: true,
		},
		{
			name: "expect no error when secret is found and updated",
			args: args{
				clientset: mockGitOpsSecretFoundAndUpdatedClient(),
				repoURL:   "test-repo",
			},
			wantErr: false,
		},
		{
			name: "expect error when secret is found but update failed",
			args: args{
				clientset: mockGitOpsSecretUpdateFailedClient(),
				repoURL:   "test-repo",
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := deleteKeysFromGitOpsSecret(tt.args.clientset, tt.args.repoURL); (err != nil) != tt.wantErr {
				t.Errorf("deleteKeysFromGitOpsSecret() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func Test_isGitOpsRepoConfiguredForMultipleApps(t *testing.T) {
	type args struct {
		gitOpsEncodedMap map[string]string
		repoURI          string
	}
	tests := []struct {
		name    string
		args    args
		want    bool
		wantErr bool
	}{
		{
			name: "expect false when no apps are configured for the repo",
			args: args{
				gitOpsEncodedMap: map[string]string{},
				repoURI:          "test-repo",
			},
			want:    false,
			wantErr: false,
		},
		{
			name: "expect false when only one app is configured for the repo",
			args: args{
				gitOpsEncodedMap: map[string]string{
					"test-app-test-cluster": base64.StdEncoding.EncodeToString([]byte(`{"repoUri":"test-repo"}`)),
				},
				repoURI: "test-repo",
			},
			want:    false,
			wantErr: false,
		},
		{
			name: "expect true when multiple apps are configured for the repo",
			args: args{
				gitOpsEncodedMap: map[string]string{
					"test-app-test-cluster":  base64.StdEncoding.EncodeToString([]byte(`{"repoUri":"test-repo"}`)),
					"test-app2-test-cluster": base64.StdEncoding.EncodeToString([]byte(`{"repoUri":"test-repo"}`)),
				},
				repoURI: "test-repo",
			},
			want:    true,
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := isGitOpsRepoConfiguredForMultipleApps(tt.args.gitOpsEncodedMap, tt.args.repoURI)
			if (err != nil) != tt.wantErr {
				t.Errorf("isGitOpsRepoConfiguredForMultipleApps() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("isGitOpsRepoConfiguredForMultipleApps() = %v, want %v", got, tt.want)
			}
		})
	}
}
