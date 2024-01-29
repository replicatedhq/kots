package gitops

import (
	"context"
	"crypto/ed25519"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/go-git/go-git/v5/plumbing/transport"
	go_git_ssh "github.com/go-git/go-git/v5/plumbing/transport/ssh"
	"github.com/mikesmitty/edkey"
	"github.com/pkg/errors"
	"github.com/replicatedhq/kots/pkg/apparchive"
	"github.com/replicatedhq/kots/pkg/binaries"
	"github.com/replicatedhq/kots/pkg/crypto"
	"github.com/replicatedhq/kots/pkg/k8sutil"
	"github.com/replicatedhq/kots/pkg/kotsadm/types"
	"github.com/replicatedhq/kots/pkg/util"
	"golang.org/x/crypto/ssh"
	v1 "k8s.io/api/core/v1"
	kuberneteserrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

type GitOpsConfig struct {
	Provider    string `json:"provider"`
	RepoURI     string `json:"repoUri"`
	Hostname    string `json:"hostname"`
	HTTPPort    string `json:"httpPort"`
	SSHPort     string `json:"sshPort"`
	Path        string `json:"path"`
	Branch      string `json:"branch"`
	Format      string `json:"format"`
	Action      string `json:"action"`
	PublicKey   string `json:"publicKey"`
	PrivateKey  string `json:"-"`
	IsConnected bool   `json:"isConnected"`
}

type GlobalGitOpsConfig struct {
	Enabled  bool   `json:"enabled"`
	Hostname string `json:"hostname"`
	HTTPPort string `json:"httpPort"`
	SSHPort  string `json:"sshPort"`
	Provider string `json:"provider"`
	URI      string `json:"uri"`
}

type KeyPair struct {
	PrivateKeyPEM string
	PublicKeySSH  string
}

func (g *GitOpsConfig) CommitURL(hash string) string {
	switch g.Provider {
	case "github", "github_enterprise":
		return fmt.Sprintf("%s/commit/%s", g.RepoURI, hash)

	case "gitlab", "gitlab_enterprise":
		return fmt.Sprintf("%s/commit/%s", g.RepoURI, hash)

	case "bitbucket", "bitbucket_server":
		return fmt.Sprintf("%s/commits/%s", g.RepoURI, hash)

	default:
		return fmt.Sprintf("%s/commit/%s", g.RepoURI, hash)
	}
}

func (g *GitOpsConfig) CloneURL() (string, error) {
	// copied this logic from node js api
	uriParts := strings.Split(g.RepoURI, "/")

	if len(uriParts) < 5 {
		return "", errors.Errorf("unexpected url format: %s", g.RepoURI)
	}

	owner := uriParts[3]
	repo := uriParts[4]

	if g.Provider == "bitbucket_server" {
		if len(uriParts) < 7 {
			return "", errors.Errorf("unexpected bitbucket server url format: %s", g.RepoURI)
		}
		owner = uriParts[4]
		repo = uriParts[6]
	}

	switch g.Provider {
	case "github":
		return fmt.Sprintf("git@github.com:%s/%s.git", owner, repo), nil
	case "gitlab":
		return fmt.Sprintf("git@gitlab.com:%s/%s.git", owner, repo), nil
	case "bitbucket":
		return fmt.Sprintf("git@bitbucket.org:%s/%s.git", owner, repo), nil
	case "bitbucket_server":
		return fmt.Sprintf("git@%s:%s/%s/%s.git", g.Hostname, g.SSHPort, owner, repo), nil
	case "github_enterprise", "gitlab_enterprise":
		return fmt.Sprintf("git@%s:%s/%s.git", g.Hostname, owner, repo), nil
	}

	return "", errors.Errorf("unsupported provider type: %s", g.Provider)
}

// GetDownstreamGitOps will return the gitops config for a downstream,
// This implementation copies how it works in typescript.
func GetDownstreamGitOps(appID string, clusterID string) (*GitOpsConfig, error) {
	clientset, err := k8sutil.GetClientset()
	if err != nil {
		return nil, errors.Wrap(err, "failed to get k8s client set")
	}

	config, err := GetDownstreamGitOpsConfig(clientset, appID, clusterID)
	return config, errors.Wrap(err, "failed to get downstream gitops config")
}

func GetDownstreamGitOpsConfig(clientset kubernetes.Interface, appID string, clusterID string) (*GitOpsConfig, error) {
	secret, err := clientset.CoreV1().Secrets(util.PodNamespace).Get(context.TODO(), "kotsadm-gitops", metav1.GetOptions{})
	if kuberneteserrors.IsNotFound(err) {
		return nil, nil
	}

	configMap, err := clientset.CoreV1().ConfigMaps(util.PodNamespace).Get(context.TODO(), "kotsadm-gitops", metav1.GetOptions{})
	if kuberneteserrors.IsNotFound(err) {
		return nil, nil
	}

	configMapDataKey := fmt.Sprintf("%s-%s", appID, clusterID)
	configMapDataEncoded, ok := configMap.Data[configMapDataKey]
	if !ok {
		return nil, nil
	}
	configMapDataDecoded, err := base64.StdEncoding.DecodeString(configMapDataEncoded)
	if err != nil {
		return nil, errors.Wrap(err, "failed to decode configmap data")
	}

	configMapData := map[string]string{}
	if err := json.Unmarshal(configMapDataDecoded, &configMapData); err != nil {
		return nil, errors.Wrap(err, "failed to unmarshal configmap data")
	}

	repoURI := configMapData["repoUri"]

	for key, val := range secret.Data {
		splitKey := strings.Split(key, ".")
		if len(splitKey) != 3 {
			continue
		}

		if splitKey[2] == "repoUri" {
			if string(val) == repoURI {
				// this is the provider we want
				idx, err := strconv.ParseInt(splitKey[1], 10, 64)
				if err != nil {
					return nil, errors.Wrap(err, "failed to parse index")
				}
				provider, publicKey, privateKey, repoURI, hostname, httpPort, sshPort := gitOpsConfigFromSecretData(idx, secret.Data)

				decodedPrivateKey, err := base64.StdEncoding.DecodeString(privateKey)
				if err != nil {
					return nil, errors.Wrap(err, "failed to decode")
				}

				decryptedPrivateKey, err := crypto.Decrypt([]byte(decodedPrivateKey))
				if err != nil {
					return nil, errors.Wrap(err, "failed to decrypt")
				}

				gitOpsConfig := GitOpsConfig{
					Provider:   provider,
					PublicKey:  publicKey,
					PrivateKey: string(decryptedPrivateKey),
					RepoURI:    repoURI,
					Hostname:   hostname,
					HTTPPort:   httpPort,
					SSHPort:    sshPort,
					Branch:     configMapData["branch"],
					Path:       configMapData["path"],
					Format:     configMapData["format"],
					Action:     configMapData["action"],
				}

				if lastError, ok := configMapData["lastError"]; ok && lastError == "" {
					gitOpsConfig.IsConnected = true
				}

				return &gitOpsConfig, nil
			}
		}
	}

	return nil, nil
}

func DisableDownstreamGitOps(appID string, clusterID string, gitOpsConfig *GitOpsConfig) error {
	clientset, err := k8sutil.GetClientset()
	if err != nil {
		return errors.Wrap(err, "failed to get k8s client set")
	}

	err = deleteDownstreamGitOps(clientset, appID, clusterID, gitOpsConfig.RepoURI)
	return errors.Wrap(err, "failed to delete data from gitops configmap")
}

// deleteDownstreamGitOps will delete the gitops config for a downstream, and delete the provider from the secret
func deleteDownstreamGitOps(clientset kubernetes.Interface, appID string, clusterID string, repoURI string) error {
	configMap, err := clientset.CoreV1().ConfigMaps(util.PodNamespace).Get(context.TODO(), "kotsadm-gitops", metav1.GetOptions{})
	if kuberneteserrors.IsNotFound(err) {
		return errors.Wrap(err, "gitops config map not found")
	}
	// if multiple apps have same repo, don't delete the provider from the secret
	isRepoConfiguredForMultipleApps, err := isGitOpsRepoConfiguredForMultipleApps(configMap.Data, repoURI)
	if err != nil {
		return errors.Wrap(err, "failed to check if repo is configured for other apps")
	}

	configMapDataKey := fmt.Sprintf("%s-%s", appID, clusterID)
	_, ok := configMap.Data[configMapDataKey]
	if ok {
		delete(configMap.Data, configMapDataKey)
	}

	_, err = clientset.CoreV1().ConfigMaps(util.PodNamespace).Update(context.TODO(), configMap, metav1.UpdateOptions{})
	if err != nil {
		return errors.Wrap(err, "failed to update config map")
	}

	if !isRepoConfiguredForMultipleApps {
		if err := deleteKeysFromGitOpsSecret(clientset, repoURI); err != nil {
			return errors.Wrap(err, "failed to delete keys from gitops secret")
		}
	}

	return nil
}

// isGitOpsRepoConfiguredForMultipleApps returns true if the repo is configured for multiple apps
func isGitOpsRepoConfiguredForMultipleApps(gitOpsEncodedMap map[string]string, repoURI string) (bool, error) {
	repoURICount := 0
	for _, val := range gitOpsEncodedMap {
		configMapDataDecoded, err := base64.StdEncoding.DecodeString(val)
		if err != nil {
			return false, errors.Wrap(err, "failed to decode configmap data")
		}

		configMapData := map[string]string{}
		if err := json.Unmarshal(configMapDataDecoded, &configMapData); err != nil {
			return false, errors.Wrap(err, "failed to unmarshal configmap data")
		}

		if configMapData["repoUri"] == repoURI {
			repoURICount++
		}

		if repoURICount > 1 {
			return true, nil
		}
	}
	return false, nil
}

// deleteKeysFromGitOpsSecret deletes all keys from the gitops secret that match the given repoURL
func deleteKeysFromGitOpsSecret(clientset kubernetes.Interface, repoURL string) error {
	secret, err := clientset.CoreV1().Secrets(util.PodNamespace).Get(context.TODO(), "kotsadm-gitops", metav1.GetOptions{})
	if kuberneteserrors.IsNotFound(err) {
		return errors.Wrap(err, "gitops secret not found")
	}

	var keyIndex int64 = -1
	for key, val := range secret.Data {
		splitKey := strings.Split(key, ".")
		if len(splitKey) != 3 {
			continue
		}

		if splitKey[2] == "repoUri" {
			if string(val) == repoURL {
				keyIndex, err = strconv.ParseInt(splitKey[1], 10, 64)
				if err != nil {
					return errors.Wrap(err, "failed to parse index")
				}
				break
			}
		}
	}

	if keyIndex == -1 {
		return nil
	}

	// delete all keys that match "provider.keyIndex.*"
	for key, _ := range secret.Data {
		if strings.HasPrefix(key, fmt.Sprintf("provider.%d.", keyIndex)) {
			delete(secret.Data, key)
		}
	}

	_, err = clientset.CoreV1().Secrets(util.PodNamespace).Update(context.TODO(), secret, metav1.UpdateOptions{})
	if err != nil {
		return errors.Wrap(err, "failed to update secret")
	}

	return nil
}

func UpdateDownstreamGitOps(appID, clusterID, uri, branch, path, format, action string) error {
	clientset, err := k8sutil.GetClientset()
	if err != nil {
		return errors.Wrap(err, "failed to get k8s client set")
	}

	err = updateDownstreamGitOps(clientset, appID, clusterID, uri, branch, path, format, action)
	return errors.Wrap(err, "failed to update downstream gitops config")
}

func updateDownstreamGitOps(clientset kubernetes.Interface, appID, clusterID, uri, branch, path, format, action string) error {
	configMap, err := clientset.CoreV1().ConfigMaps(util.PodNamespace).Get(context.TODO(), "kotsadm-gitops", metav1.GetOptions{})
	if err != nil && !kuberneteserrors.IsNotFound(err) {
		return errors.Wrap(err, "failed to get configmap")
	}

	configMapExists := true
	if kuberneteserrors.IsNotFound(err) {
		configMapExists = false
	}

	configMapData := map[string]string{}
	if configMapExists && configMap.Data != nil {
		configMapData = configMap.Data
	}

	appKey := fmt.Sprintf("%s-%s", appID, clusterID)
	newAppData := map[string]string{
		"repoUri": uri,
		"branch":  branch,
		"path":    path,
		"format":  format,
		"action":  action,
	}

	// check if to reset or keep last error
	appDataEncoded, ok := configMapData[appKey]
	if ok {
		appDataDecoded, err := base64.StdEncoding.DecodeString(appDataEncoded)
		if err != nil {
			return errors.Wrap(err, "failed to decode app data")
		}

		appDataUnmarshalled := map[string]string{}
		if err := json.Unmarshal(appDataDecoded, &appDataUnmarshalled); err != nil {
			return errors.Wrap(err, "failed to unmarshal app data")
		}

		oldUri, _ := appDataUnmarshalled["repoUri"]
		oldBranch, _ := appDataUnmarshalled["branch"]
		if oldBranch == branch && oldUri == uri {
			lastError, ok := appDataUnmarshalled["lastError"]
			if ok {
				newAppData["lastError"] = lastError // keep last error
			}
		}
	}

	// update/set app data in config map
	newAppDataMarshalled, err := json.Marshal(newAppData)
	if err != nil {
		return errors.Wrap(err, "failed to marshal new app data")
	}
	newAppDataEncoded := base64.StdEncoding.EncodeToString([]byte(newAppDataMarshalled))
	configMapData[appKey] = newAppDataEncoded

	if configMapExists {
		configMap.Data = configMapData
		_, err = clientset.CoreV1().ConfigMaps(util.PodNamespace).Update(context.TODO(), configMap, metav1.UpdateOptions{})
		if err != nil {
			return errors.Wrap(err, "failed to update config map")
		}
	} else {
		configMap = &v1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "kotsadm-gitops",
				Namespace: util.PodNamespace,
				Labels:    types.GetKotsadmLabels(),
			},
			Data: configMapData,
		}
		_, err = clientset.CoreV1().ConfigMaps(util.PodNamespace).Create(context.TODO(), configMap, metav1.CreateOptions{})
		if err != nil {
			return errors.Wrap(err, "failed to create config map")
		}
	}

	return nil
}

func SetGitOpsError(appID string, clusterID string, errMsg string) error {
	clientset, err := k8sutil.GetClientset()
	if err != nil {
		return errors.Wrap(err, "failed to get k8s client set")
	}

	configMap, err := clientset.CoreV1().ConfigMaps(util.PodNamespace).Get(context.TODO(), "kotsadm-gitops", metav1.GetOptions{})
	if err != nil {
		return errors.Wrap(err, "failed to get configmap")
	}

	appKey := fmt.Sprintf("%s-%s", appID, clusterID)
	appDataEncoded, ok := configMap.Data[appKey]
	if !ok {
		return errors.New("app gitops data not found in configmap")
	}

	appDataDecoded, err := base64.StdEncoding.DecodeString(appDataEncoded)
	if err != nil {
		return errors.Wrap(err, "failed to decode app data")
	}

	appDataUnmarshalled := map[string]string{}
	if err := json.Unmarshal(appDataDecoded, &appDataUnmarshalled); err != nil {
		return errors.Wrap(err, "failed to unmarshal app data")
	}
	appDataUnmarshalled["lastError"] = errMsg

	appDataDecoded, err = json.Marshal(appDataUnmarshalled)
	if err != nil {
		return errors.Wrap(err, "failed to marshal app data")
	}
	appDataEncoded = base64.StdEncoding.EncodeToString([]byte(appDataDecoded))
	configMap.Data[appKey] = appDataEncoded

	_, err = clientset.CoreV1().ConfigMaps(util.PodNamespace).Update(context.TODO(), configMap, metav1.UpdateOptions{})
	if err != nil {
		return errors.Wrap(err, "failed to update config map")
	}

	return nil
}

// TestGitOpsConnection will attempt a clone of the target gitops repo.
// It returns the default branch name from the clone.
func TestGitOpsConnection(gitOpsConfig *GitOpsConfig) (string, error) {
	auth, err := getAuth(gitOpsConfig.PrivateKey)
	if err != nil {
		return "", errors.Wrap(err, "failed to get auth")
	}

	workDir, err := ioutil.TempDir("", "kotsadm")
	if err != nil {
		return "", errors.Wrap(err, "failed to create temp dir")
	}
	defer os.RemoveAll(workDir)

	cloneURL, err := gitOpsConfig.CloneURL()
	if err != nil {
		return "", errors.Wrap(err, "failed to get clone url")
	}

	repo, err := git.PlainClone(workDir, false, &git.CloneOptions{
		URL:               cloneURL,
		RecurseSubmodules: git.DefaultSubmoduleRecursionDepth,
		Auth:              auth,
	})
	if err != nil && errors.Cause(err) != transport.ErrEmptyRemoteRepository {
		return "", errors.Wrap(err, "failed to clone repo")
	}

	ref, err := repo.Head()
	if err != nil {
		return "", errors.Wrap(err, "failed to identify HEAD of repo")
	}

	return ref.Name().Short(), nil
}

func CreateGitOps(provider string, repoURI string, hostname string, httpPort string, sshPort string) error {
	clientset, err := k8sutil.GetClientset()
	if err != nil {
		return errors.Wrap(err, "failed to get k8s client set")
	}

	err = createGitOps(clientset, provider, repoURI, hostname, httpPort, sshPort)
	return errors.Wrap(err, "failed to create gitops")
}

func createGitOps(clientset kubernetes.Interface, provider string, repoURI string, hostname string, httpPort string, sshPort string) error {
	secret, err := clientset.CoreV1().Secrets(util.PodNamespace).Get(context.TODO(), "kotsadm-gitops", metav1.GetOptions{})
	if err != nil && !kuberneteserrors.IsNotFound(err) {
		return errors.Wrap(err, "failed to get secret")
	}

	secretExists := true
	if kuberneteserrors.IsNotFound(err) {
		secretExists = false
	}

	secretData := map[string][]byte{}
	if secretExists && secret.Data != nil {
		secretData = secret.Data
	}

	var repoIdx int64 = -1
	var repoExists bool = false
	var maxIdx int64 = -1

	for key, val := range secretData {
		splitKey := strings.Split(key, ".")
		if len(splitKey) != 3 {
			continue
		}

		if splitKey[2] != "repoUri" {
			continue
		}

		idx, err := strconv.ParseInt(splitKey[1], 10, 64)
		if err != nil {
			return errors.Wrap(err, "failed to parse repo index")
		}

		if string(val) == repoURI {
			repoIdx = idx
			repoExists = true
		}

		if idx > maxIdx {
			maxIdx = idx
		}
	}

	if !repoExists {
		repoIdx = maxIdx + 1
	}

	secretData[fmt.Sprintf("provider.%d.type", repoIdx)] = []byte(provider)
	secretData[fmt.Sprintf("provider.%d.repoUri", repoIdx)] = []byte(repoURI)

	if !repoExists {
		keyPair, err := generatePrivateKey_ed25519()
		if err != nil {
			return errors.Wrap(err, "failed to generate ed25519 key pair")
		}

		encryptedPrivateKey := crypto.Encrypt([]byte(keyPair.PrivateKeyPEM))
		encodedPrivateKey := base64.StdEncoding.EncodeToString(encryptedPrivateKey) // encoding here shouldn't be needed. moved logic from TS where ffi EncryptString function base64 encodes the value as well

		secretData[fmt.Sprintf("provider.%d.privateKey", repoIdx)] = []byte(encodedPrivateKey)
		secretData[fmt.Sprintf("provider.%d.publicKey", repoIdx)] = []byte(keyPair.PublicKeySSH)
	}

	hostnameKey := fmt.Sprintf("provider.%d.hostname", repoIdx)
	_, ok := secretData[hostnameKey]
	if ok {
		delete(secretData, hostnameKey)
	}
	if hostname != "" {
		secretData[hostnameKey] = []byte(hostname)
	}

	httpPortKey := fmt.Sprintf("provider.%d.httpPort", repoIdx)
	_, ok = secretData[httpPortKey]
	if ok {
		delete(secretData, httpPortKey)
	}
	if httpPort != "" {
		secretData[httpPortKey] = []byte(httpPort)
	}

	sshPortKey := fmt.Sprintf("provider.%d.sshPort", repoIdx)
	_, ok = secretData[sshPortKey]
	if ok {
		delete(secretData, sshPortKey)
	}
	if sshPort != "" {
		secretData[sshPortKey] = []byte(sshPort)
	}

	if secretExists {
		secret.Data = secretData
		_, err = clientset.CoreV1().Secrets(util.PodNamespace).Update(context.TODO(), secret, metav1.UpdateOptions{})
		if err != nil {
			return errors.Wrap(err, "failed to update secret")
		}
	} else {
		secret = &v1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "kotsadm-gitops",
				Namespace: util.PodNamespace,
				Labels:    types.GetKotsadmLabels(),
			},
			Data: secretData,
		}
		_, err = clientset.CoreV1().Secrets(util.PodNamespace).Create(context.TODO(), secret, metav1.CreateOptions{})
		if err != nil {
			return errors.Wrap(err, "failed to create secret")
		}
	}

	return nil
}

func ResetGitOps() error {
	clientset, err := k8sutil.GetClientset()
	if err != nil {
		return errors.Wrap(err, "failed to get k8s client set")
	}

	err = clientset.CoreV1().Secrets(util.PodNamespace).Delete(context.TODO(), "kotsadm-gitops", metav1.DeleteOptions{})
	if err != nil && !kuberneteserrors.IsNotFound(err) {
		return errors.Wrap(err, "failed to delete secret")
	}

	err = clientset.CoreV1().ConfigMaps(util.PodNamespace).Delete(context.TODO(), "kotsadm-gitops", metav1.DeleteOptions{})
	if err != nil && !kuberneteserrors.IsNotFound(err) {
		return errors.Wrap(err, "failed to delete configmap")
	}

	return nil
}

func GetGitOps() (GlobalGitOpsConfig, error) {
	clientset, err := k8sutil.GetClientset()
	if err != nil {
		return GlobalGitOpsConfig{}, errors.Wrap(err, "failed to get k8s client set")
	}

	secret, err := clientset.CoreV1().Secrets(util.PodNamespace).Get(context.TODO(), "kotsadm-gitops", metav1.GetOptions{})
	if kuberneteserrors.IsNotFound(err) {
		return GlobalGitOpsConfig{}, nil
	} else if err != nil {
		return GlobalGitOpsConfig{}, errors.Wrap(err, "get kotsadm-gitops secret")
	}

	parsedConfig := GlobalGitOpsConfig{
		Enabled:  true,
		Provider: string(secret.Data["provider.0.type"]),
		URI:      string(secret.Data["provider.0.repoUri"]),
		Hostname: string(secret.Data["provider.0.hostname"]),
		HTTPPort: string(secret.Data["provider.0.httpPort"]),
		SSHPort:  string(secret.Data["provider.0.sshPort"]),
	}

	return parsedConfig, nil
}

func gitOpsConfigFromSecretData(idx int64, secretData map[string][]byte) (string, string, string, string, string, string, string) {
	provider := ""
	publicKey := ""
	privateKey := ""
	repoURI := ""
	hostname := ""
	httpPort := ""
	sshPort := ""

	providerDecoded, ok := secretData[fmt.Sprintf("provider.%d.type", idx)]
	if ok {
		provider = string(providerDecoded)
	}

	publicKeyDecoded, ok := secretData[fmt.Sprintf("provider.%d.publicKey", idx)]
	if ok {
		publicKey = string(publicKeyDecoded)
	}

	privateKeyDecoded, ok := secretData[fmt.Sprintf("provider.%d.privateKey", idx)]
	if ok {
		privateKey = string(privateKeyDecoded)
	}

	repoURIDecoded, ok := secretData[fmt.Sprintf("provider.%d.repoUri", idx)]
	if ok {
		repoURI = string(repoURIDecoded)
	}

	hostnameDecoded, ok := secretData[fmt.Sprintf("provider.%d.hostname", idx)]
	if ok {
		hostname = string(hostnameDecoded)
	}

	httpPortDecoded, ok := secretData[fmt.Sprintf("provider.%d.httpPort", idx)]
	if ok {
		httpPort = string(httpPortDecoded)
	}

	sshPortDecoded, ok := secretData[fmt.Sprintf("provider.%d.sshPort", idx)]
	if ok {
		sshPort = string(sshPortDecoded)
	}

	return provider, publicKey, privateKey, repoURI, hostname, httpPort, sshPort
}

func getAuth(privateKey string) (transport.AuthMethod, error) {
	var auth transport.AuthMethod
	signer, err := ssh.ParsePrivateKey([]byte(privateKey))
	if err != nil {
		return nil, errors.Wrap(err, "failed to parse deploy key")
	}
	auth = &go_git_ssh.PublicKeys{User: "git", Signer: signer}
	auth.(*go_git_ssh.PublicKeys).HostKeyCallback = ssh.InsecureIgnoreHostKey()
	return auth, nil
}

func CreateGitOpsCommit(gitOpsConfig *GitOpsConfig, appSlug string, appName string, newSequence int, archiveDir string, downstreamName string) (string, error) {
	out, _, err := apparchive.GetRenderedApp(archiveDir, downstreamName, binaries.GetKustomizeBinPath())
	if err != nil {
		return "", errors.Wrap(err, "failed to get rendered app")
	}

	// using the deploy key, create the commit in a new branch
	auth, err := getAuth(gitOpsConfig.PrivateKey)
	if err != nil {
		return "", errors.Wrap(err, "failed to get auth")
	}

	workDir, err := ioutil.TempDir("", "kotsadm")
	if err != nil {
		return "", errors.Wrap(err, "failed to create temp dir")
	}
	defer os.RemoveAll(workDir)

	cloneURL, err := gitOpsConfig.CloneURL()
	if err != nil {
		return "", errors.Wrap(err, "failed to get clone url")
	}

	cloneOptions := &git.CloneOptions{
		RemoteName:        git.DefaultRemoteName,
		URL:               cloneURL,
		RecurseSubmodules: git.DefaultSubmoduleRecursionDepth,
		Auth:              auth,
	}
	cloned, workTree, err := CloneAndCheckout(workDir, cloneOptions, gitOpsConfig.Branch)
	if err != nil {
		return "", err
	}

	dirPath := filepath.Join(workDir, gitOpsConfig.Path)
	_, err = os.Stat(dirPath)
	if os.IsNotExist(err) {
		// create subdirectory if not exist
		err := os.MkdirAll(dirPath, 0755)
		if err != nil {
			return "", errors.Wrap(err, "failed to mkdir")
		}
	} // ignore error here and let the stat of the file below handle any errors

	filePath := filepath.Join(dirPath, fmt.Sprintf("%s.yaml", appSlug))
	_, err = os.Stat(filePath)
	if err == nil { // if the file has not changed, end now
		currentRevision, err := os.ReadFile(filePath)
		if err != nil {
			return "", errors.Wrap(err, "failed to read current app yaml")
		}
		if string(currentRevision) == string(out) {
			return "", nil
		}
	} else if !os.IsNotExist(err) {
		return "", errors.Wrap(err, "failed to stat current app yaml")
	}

	err = ioutil.WriteFile(filePath, out, 0644)
	if err != nil {
		return "", errors.Wrap(err, "failed to write updated app yaml")
	}

	_, err = workTree.Add(strings.TrimPrefix(filepath.Join(gitOpsConfig.Path, fmt.Sprintf("%s.yaml", appSlug)), "/"))
	if err != nil {
		return "", errors.Wrap(err, "failed to add to worktree")
	}

	// commit it
	updatedHash, err := workTree.Commit(fmt.Sprintf("Updating %s to version %d", appName, newSequence), &git.CommitOptions{
		Author: &object.Signature{
			Name:  "KOTS Admin Console",
			Email: "help@replicated.com",
			When:  time.Now(),
		},
	})
	if err != nil {
		return "", errors.Wrap(err, "failed to commit")
	}

	err = cloned.Push(&git.PushOptions{
		RemoteName: cloneOptions.RemoteName,
		Auth:       auth,
	})
	if err != nil {
		return "", errors.Wrap(err, "failed to push")
	}

	return gitOpsConfig.CommitURL(updatedHash.String()), nil
}

func generatePrivateKey_ed25519() (*KeyPair, error) {
	publicKey, privateKey, err := ed25519.GenerateKey(nil)
	if err != nil {
		return nil, errors.Wrap(err, "generate ed25519 key pair")
	}

	sshPublicKey, err := ssh.NewPublicKey(publicKey)
	if err != nil {
		return nil, errors.Wrap(err, "convert public key to ssh")
	}

	pemPrivateKey := &pem.Block{
		Type:  "OPENSSH PRIVATE KEY",
		Bytes: edkey.MarshalED25519PrivateKey(privateKey),
	}

	keyPair := &KeyPair{
		PublicKeySSH:  string(ssh.MarshalAuthorizedKey(sshPublicKey)),
		PrivateKeyPEM: string(pem.EncodeToMemory(pemPrivateKey)),
	}
	return keyPair, nil
}
