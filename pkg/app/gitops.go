package app

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/pkg/errors"
	"github.com/replicatedhq/kots/pkg/crypto"
	"github.com/replicatedhq/kotsadm/pkg/kotsutil"
	"golang.org/x/crypto/ssh"
	"gopkg.in/src-d/go-git.v4"
	go_git_config "gopkg.in/src-d/go-git.v4/config"
	"gopkg.in/src-d/go-git.v4/plumbing"
	"gopkg.in/src-d/go-git.v4/plumbing/object"
	"gopkg.in/src-d/go-git.v4/plumbing/transport"
	go_git_ssh "gopkg.in/src-d/go-git.v4/plumbing/transport/ssh"
	kuberneteserrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
)

type GitOpsConfig struct {
	Provider   string `json:"provider"`
	RepoURI    string `json:"repoUri"`
	Hostname   string `json:"hostname"`
	Path       string `json:"path"`
	Branch     string `json:"branch"`
	Format     string `json:"format"`
	Action     string `json:"action"`
	PublicKey  string `json:"publicKey"`
	PrivateKey string `json:"-"`
	LastError  string `json:"lastError"`
}

func (g *GitOpsConfig) CommitURL(hash string) string {
	switch g.Provider {
	case "github":
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

	secret, err := clientset.CoreV1().Secrets(os.Getenv("POD_NAMESPACE")).Get("kotsadm-gitops", metav1.GetOptions{})
	if kuberneteserrors.IsNotFound(err) {
		return nil, nil
	}

	configMap, err := clientset.CoreV1().ConfigMaps(os.Getenv("POD_NAMESPACE")).Get("kotsadm-gitops", metav1.GetOptions{})
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
		return nil, errors.Wrap(err, "faield to unmarshal configmap data")
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
				}
				provider, publicKey, privateKey, repoURI, err := gitOpsConfigFromSecretData(idx, secret.Data)

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
					Branch:     configMapData["branch"],
					Path:       configMapData["path"],
					Format:     configMapData["format"],
					Action:     configMapData["action"],
					LastError:  configMapData["lastError"],
				}

				return &gitOpsConfig, nil
			}
		}
	}

	return nil, nil
}

func gitOpsConfigFromSecretData(idx int64, secretData map[string][]byte) (string, string, string, string, error) {
	provider := ""
	publicKey := ""
	privateKey := ""
	repoURI := ""

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

	return provider, publicKey, privateKey, repoURI, nil
}

func createGitOpsCommit(gitOpsConfig *GitOpsConfig, appSlug string, appName string, newSequence int, archiveDir string, downstreamName string) (string, error) {
	kotsKinds, err := kotsutil.LoadKotsKindsFromPath(archiveDir)
	if err != nil {
		return "", errors.Wrap(err, "failed to load kots kinds")
	}

	kustomizeVersion := "3.5.4"
	if kotsKinds.KotsApplication.Spec.KustomizeVersion != "" {
		kustomizeVersion = kotsKinds.KotsApplication.Spec.KustomizeVersion
	}

	// we use the kustomize binary here...
	cmd := exec.Command(fmt.Sprintf("kustomize%s", kustomizeVersion), "build", filepath.Join(archiveDir, "overlays", "downstreams", downstreamName))
	out, err := cmd.CombinedOutput()
	if err != nil {
		return "", errors.Wrap(err, "failed to run kustomize")
	}

	// using the deploy key, create the commit in a new branch
	var auth transport.AuthMethod
	signer, err := ssh.ParsePrivateKey([]byte(gitOpsConfig.PrivateKey))
	if err != nil {
		return "", errors.Wrap(err, "failed to parse deploy key")
	}
	auth = &go_git_ssh.PublicKeys{User: "git", Signer: signer}
	auth.(*go_git_ssh.PublicKeys).HostKeyCallback = ssh.InsecureIgnoreHostKey()

	workDir, err := ioutil.TempDir("", "kotsadm")
	if err != nil {
		return "", errors.Wrap(err, "failed to create temp dir")
	}
	defer os.RemoveAll(workDir)

	cloned, err := git.PlainClone(workDir, false, &git.CloneOptions{
		URL:               gitOpsConfig.CloneURL(),
		RecurseSubmodules: git.DefaultSubmoduleRecursionDepth,
		Auth:              auth,
	})
	if err != nil {
		return "", errors.Wrap(err, "failed to clone repo")
	}

	workTree, err := cloned.Worktree()
	if err != nil {
		return "", errors.Wrap(err, "failed to get worktree")
	}

	err = cloned.Fetch(&git.FetchOptions{
		RefSpecs: []go_git_config.RefSpec{"refs/*:refs/*", "HEAD:refs/heads/HEAD"},
		Auth:     auth,
	})
	if err != nil {
		return "", errors.Wrap(err, "failed to fetch from repo")
	}

	// try to check out the branch if it exists
	err = workTree.Checkout(&git.CheckoutOptions{
		Create: false,
		Force:  false,
		Branch: plumbing.ReferenceName(fmt.Sprintf("refs/heads/%s", gitOpsConfig.Branch)),
	})
	if err != nil {
		err := workTree.Checkout(&git.CheckoutOptions{
			Create: true,
			Force:  false,
			Branch: plumbing.ReferenceName(fmt.Sprintf("refs/heads/%s", gitOpsConfig.Branch)),
		})
		if err != nil {
			return "", errors.Wrap(err, "failed to get or create branch")
		}
	}

	// if the file has not changed, end now
	currentRevision, err := ioutil.ReadFile(filepath.Join(workDir, gitOpsConfig.Path, fmt.Sprintf("%s.yaml", appSlug)))
	if err != nil {
		return "", errors.Wrap(err, "failed to read current file")
	}
	if string(currentRevision) == string(out) {
		return "", nil
	}

	err = ioutil.WriteFile(filepath.Join(workDir, gitOpsConfig.Path, fmt.Sprintf("%s.yaml", appSlug)), out, 0644)
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
		Auth: auth,
	})
	if err != nil {
		return "", errors.Wrap(err, "failed to push")
	}

	return gitOpsConfig.CommitURL(updatedHash.String()), nil
}
