package gitops

import (
	"context"
	"crypto/rsa"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/go-git/go-git/v5/plumbing/transport"
	go_git_ssh "github.com/go-git/go-git/v5/plumbing/transport/ssh"
	"github.com/pkg/errors"
	"github.com/replicatedhq/kots/pkg/crypto"
	"github.com/replicatedhq/kots/pkg/kotsutil"
	"golang.org/x/crypto/ssh"
	v1 "k8s.io/api/core/v1"
	kuberneteserrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
)

type GitOpsConfig struct {
	Provider    string `json:"provider"`
	RepoURI     string `json:"repoUri"`
	Hostname    string `json:"hostname"`
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

	case "gitlab":
		return fmt.Sprintf("%s/commit/%s", g.RepoURI, hash)

	case "bitbucket":
		return fmt.Sprintf("%s/commits/%s", g.RepoURI, hash)

	default:
		return fmt.Sprintf("%s/commit/%s", g.RepoURI, hash)
	}
}

func (g *GitOpsConfig) CloneURL() string {
	// copied this logic from node js api
	// this feels incomplete and fragile....  needs enterprise support
	uriParts := strings.Split(g.RepoURI, "/")

	switch g.Provider {
	case "github":
		return fmt.Sprintf("git@github.com:%s/%s.git", uriParts[3], uriParts[4])
	case "github_enterprise":
		return fmt.Sprintf("git@%s:%s/%s.git", uriParts[2], uriParts[3], uriParts[4])
	case "gitlab":
		return fmt.Sprintf("git@gitlab.com:%s/%s.git", uriParts[3], uriParts[4])
	case "bitbucket":
		return fmt.Sprintf("git@bitbucket.org:%s/%s.git", uriParts[3], uriParts[4])
	}

	return ""
}

// GetDownstreamGitOps will return the gitops config for a downstrea,
// This implementation copies how it works in typescript.
func GetDownstreamGitOps(appID string, clusterID string) (*GitOpsConfig, error) {
	cfg, err := config.GetConfig()
	if err != nil {
		return nil, errors.Wrap(err, "failed to get cluster config")
	}

	clientset, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create kubernetes clientset")
	}

	secret, err := clientset.CoreV1().Secrets(os.Getenv("POD_NAMESPACE")).Get(context.TODO(), "kotsadm-gitops", metav1.GetOptions{})
	if kuberneteserrors.IsNotFound(err) {
		return nil, nil
	}

	configMap, err := clientset.CoreV1().ConfigMaps(os.Getenv("POD_NAMESPACE")).Get(context.TODO(), "kotsadm-gitops", metav1.GetOptions{})
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
				provider, publicKey, privateKey, repoURI, hostname := gitOpsConfigFromSecretData(idx, secret.Data)

				cipher, err := crypto.AESCipherFromString(os.Getenv("API_ENCRYPTION_KEY"))
				if err != nil {
					return nil, errors.Wrap(err, "failed to create aes cipher")
				}
				decodedPrivateKey, err := base64.StdEncoding.DecodeString(privateKey)
				if err != nil {
					return nil, errors.Wrap(err, "failed to decode")
				}

				decryptedPrivateKey, err := cipher.Decrypt([]byte(decodedPrivateKey))
				if err != nil {
					return nil, errors.Wrap(err, "failed to decrypt")
				}

				gitOpsConfig := GitOpsConfig{
					Provider:   provider,
					PublicKey:  publicKey,
					PrivateKey: string(decryptedPrivateKey),
					RepoURI:    repoURI,
					Hostname:   hostname,
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

func DisableDownstreamGitOps(appID string, clusterID string) error {
	cfg, err := config.GetConfig()
	if err != nil {
		return errors.Wrap(err, "failed to get cluster config")
	}

	clientset, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		return errors.Wrap(err, "failed to create kubernetes clientset")
	}

	configMap, err := clientset.CoreV1().ConfigMaps(os.Getenv("POD_NAMESPACE")).Get(context.TODO(), "kotsadm-gitops", metav1.GetOptions{})
	if kuberneteserrors.IsNotFound(err) {
		return errors.Wrap(err, "gitops config map not found")
	}

	configMapDataKey := fmt.Sprintf("%s-%s", appID, clusterID)
	_, ok := configMap.Data[configMapDataKey]
	if ok {
		delete(configMap.Data, configMapDataKey)
	}

	_, err = clientset.CoreV1().ConfigMaps(os.Getenv("POD_NAMESPACE")).Update(context.TODO(), configMap, metav1.UpdateOptions{})
	if err != nil {
		return errors.Wrap(err, "failed to update config map")
	}

	return nil
}

func UpdateDownstreamGitOps(appID, clusterID, uri, branch, path, format, action string) error {
	cfg, err := config.GetConfig()
	if err != nil {
		return errors.Wrap(err, "failed to get cluster config")
	}

	clientset, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		return errors.Wrap(err, "failed to create kubernetes clientset")
	}

	configMap, err := clientset.CoreV1().ConfigMaps(os.Getenv("POD_NAMESPACE")).Get(context.TODO(), "kotsadm-gitops", metav1.GetOptions{})
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
		_, err = clientset.CoreV1().ConfigMaps(os.Getenv("POD_NAMESPACE")).Update(context.TODO(), configMap, metav1.UpdateOptions{})
		if err != nil {
			return errors.Wrap(err, "failed to update config map")
		}
	} else {
		configMap = &v1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "kotsadm-gitops",
				Namespace: os.Getenv("POD_NAMESPACE"),
			},
			Data: configMapData,
		}
		_, err = clientset.CoreV1().ConfigMaps(os.Getenv("POD_NAMESPACE")).Create(context.TODO(), configMap, metav1.CreateOptions{})
		if err != nil {
			return errors.Wrap(err, "failed to create config map")
		}
	}

	return nil
}

func SetGitOpsError(appID string, clusterID string, errMsg string) error {
	cfg, err := config.GetConfig()
	if err != nil {
		return errors.Wrap(err, "failed to get cluster config")
	}

	clientset, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		return errors.Wrap(err, "failed to create kubernetes clientset")
	}

	configMap, err := clientset.CoreV1().ConfigMaps(os.Getenv("POD_NAMESPACE")).Get(context.TODO(), "kotsadm-gitops", metav1.GetOptions{})
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

	_, err = clientset.CoreV1().ConfigMaps(os.Getenv("POD_NAMESPACE")).Update(context.TODO(), configMap, metav1.UpdateOptions{})
	if err != nil {
		return errors.Wrap(err, "failed to update config map")
	}

	return nil
}

func TestGitOpsConnection(gitOpsConfig *GitOpsConfig) error {
	auth, err := getAuth(gitOpsConfig.PrivateKey)
	if err != nil {
		return errors.Wrap(err, "failed to get auth")
	}

	workDir, err := ioutil.TempDir("", "kotsadm")
	if err != nil {
		return errors.Wrap(err, "failed to create temp dir")
	}
	defer os.RemoveAll(workDir)

	_, err = git.PlainClone(workDir, false, &git.CloneOptions{
		URL:               gitOpsConfig.CloneURL(),
		RecurseSubmodules: git.DefaultSubmoduleRecursionDepth,
		Auth:              auth,
	})
	if err != nil && errors.Cause(err) != transport.ErrEmptyRemoteRepository {
		return errors.Wrap(err, "failed to clone repo")
	}

	return nil
}

func CreateGitOps(provider string, repoURI string, hostname string) error {
	cfg, err := config.GetConfig()
	if err != nil {
		return errors.Wrap(err, "failed to get cluster config")
	}

	clientset, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		return errors.Wrap(err, "failed to create kubernetes clientset")
	}

	secret, err := clientset.CoreV1().Secrets(os.Getenv("POD_NAMESPACE")).Get(context.TODO(), "kotsadm-gitops", metav1.GetOptions{})
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
		keyPair, err := generateKeyPair()
		if err != nil {
			return errors.Wrap(err, "failed to generate key pair")
		}

		cipher, err := crypto.AESCipherFromString(os.Getenv("API_ENCRYPTION_KEY"))
		if err != nil {
			return errors.Wrap(err, "failed to create aes cipher")
		}
		encryptedPrivateKey := cipher.Encrypt([]byte(keyPair.PrivateKeyPEM))
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

	if secretExists {
		secret.Data = secretData
		_, err = clientset.CoreV1().Secrets(os.Getenv("POD_NAMESPACE")).Update(context.TODO(), secret, metav1.UpdateOptions{})
		if err != nil {
			return errors.Wrap(err, "failed to update secret")
		}
	} else {
		secret = &v1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "kotsadm-gitops",
				Namespace: os.Getenv("POD_NAMESPACE"),
			},
			Data: secretData,
		}
		_, err = clientset.CoreV1().Secrets(os.Getenv("POD_NAMESPACE")).Create(context.TODO(), secret, metav1.CreateOptions{})
		if err != nil {
			return errors.Wrap(err, "failed to create secret")
		}
	}

	return nil
}

func ResetGitOps() error {
	cfg, err := config.GetConfig()
	if err != nil {
		return errors.Wrap(err, "failed to get cluster config")
	}

	clientset, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		return errors.Wrap(err, "failed to create kubernetes clientset")
	}

	err = clientset.CoreV1().Secrets(os.Getenv("POD_NAMESPACE")).Delete(context.TODO(), "kotsadm-gitops", metav1.DeleteOptions{})
	if err != nil && !kuberneteserrors.IsNotFound(err) {
		return errors.Wrap(err, "failed to delete secret")
	}

	err = clientset.CoreV1().ConfigMaps(os.Getenv("POD_NAMESPACE")).Delete(context.TODO(), "kotsadm-gitops", metav1.DeleteOptions{})
	if err != nil && !kuberneteserrors.IsNotFound(err) {
		return errors.Wrap(err, "failed to delete configmap")
	}

	return nil
}

func GetGitOps() (GlobalGitOpsConfig, error) {
	cfg, err := config.GetConfig()
	if err != nil {
		return GlobalGitOpsConfig{}, errors.Wrap(err, "failed to get cluster config")
	}

	clientset, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		return GlobalGitOpsConfig{}, errors.Wrap(err, "failed to create kubernetes clientset")
	}

	secret, err := clientset.CoreV1().Secrets(os.Getenv("POD_NAMESPACE")).Get(context.TODO(), "kotsadm-gitops", metav1.GetOptions{})
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
	}

	return parsedConfig, nil
}

func gitOpsConfigFromSecretData(idx int64, secretData map[string][]byte) (string, string, string, string, string) {
	provider := ""
	publicKey := ""
	privateKey := ""
	repoURI := ""
	hostname := ""

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

	return provider, publicKey, privateKey, repoURI, hostname
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
	kotsKinds, err := kotsutil.LoadKotsKindsFromPath(archiveDir)
	if err != nil {
		return "", errors.Wrap(err, "failed to load kots kinds")
	}

	// we use the kustomize binary here...
	cmd := exec.Command(fmt.Sprintf("kustomize%s", kotsKinds.KustomizeVersion()), "build", filepath.Join(archiveDir, "overlays", "downstreams", downstreamName))
	out, err := cmd.Output()
	if err != nil {
		if ee, ok := err.(*exec.ExitError); ok {
			err = fmt.Errorf("kustomize stderr: %q", string(ee.Stderr))
		}
		return "", errors.Wrap(err, "failed to run kustomize")
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

	cloneOptions := &git.CloneOptions{
		RemoteName:        git.DefaultRemoteName,
		URL:               gitOpsConfig.CloneURL(),
		RecurseSubmodules: git.DefaultSubmoduleRecursionDepth,
		Auth:              auth,
	}
	cloned, workTree, err := CloneAndCheckout(workDir, cloneOptions, gitOpsConfig.Branch)
	if err != nil {
		return "", err
	}

	filePath := filepath.Join(workDir, gitOpsConfig.Path, fmt.Sprintf("%s.yaml", appSlug))
	_, err = os.Stat(filePath)
	if err == nil { // if the file has not changed, end now
		currentRevision, err := ioutil.ReadFile(filePath)
		if err != nil {
			return "", errors.Wrap(err, "failed to read current file")
		}
		if string(currentRevision) == string(out) {
			return "", nil
		}
	} else if os.IsNotExist(err) { // create subdirectory if not exist
		err := os.MkdirAll(filepath.Join(workDir, gitOpsConfig.Path), 0644)
		if err != nil {
			return "", errors.Wrap(err, "failed to mkdir for file")
		}
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

func generateKeyPair() (*KeyPair, error) {
	privateKey, err := getPrivateKey()
	if err != nil {
		return nil, err
	}

	var publicKey *rsa.PublicKey
	publicKey = &privateKey.PublicKey

	privateKeyBytes := x509.MarshalPKCS1PrivateKey(privateKey)
	PrivateKeyPEM := pem.EncodeToMemory(
		&pem.Block{
			Type:  "RSA PRIVATE KEY",
			Bytes: privateKeyBytes,
		},
	)

	publicKeySSH, err := ssh.NewPublicKey(publicKey)
	if err != nil {
		return nil, err
	}
	pubKeySSHBytes := ssh.MarshalAuthorizedKey(publicKeySSH)

	return &KeyPair{PrivateKeyPEM: string(PrivateKeyPEM), PublicKeySSH: string(pubKeySSHBytes)}, nil
}
