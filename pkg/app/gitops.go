package app

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/pkg/errors"
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

				gitOpsConfig := GitOpsConfig{
					Provider:   provider,
					PublicKey:  publicKey,
					PrivateKey: privateKey,
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
