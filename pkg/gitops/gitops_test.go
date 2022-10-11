package gitops

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"golang.org/x/crypto/ssh"
	"k8s.io/client-go/kubernetes/fake"
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
			wantKeyType: "ssh-rsa",
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

			config, err := getDownstreamGitOps(clientset, test.appID, test.clusterID)
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
