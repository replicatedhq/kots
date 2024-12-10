package registry

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/url"
	"strings"
	"sync"

	"github.com/pkg/errors"
	kotsadmtypes "github.com/replicatedhq/kots/pkg/kotsadm/types"
	"github.com/replicatedhq/kots/pkg/util"
	kotsv1beta1 "github.com/replicatedhq/kotskinds/apis/kots/v1beta1"
	corev1 "k8s.io/api/core/v1"
	kuberneteserrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

type RegistryProxyInfo struct {
	Registry string
	Proxy    string
	Upstream string
}

type DockercfgAuth struct {
	Auth string `json:"auth,omitempty"`
}

type DockerCfgJSON struct {
	Auths map[string]DockercfgAuth `json:"auths"`
}

type Credentials struct {
	Username string
	Password string
}

type ImagePullSecrets struct {
	AdminConsoleSecret *corev1.Secret
	AppSecret          *corev1.Secret
	DockerHubSecret    *corev1.Secret
}

const DockerHubRegistryName = "index.docker.io"
const DockerHubSecretName = "kotsadm-dockerhub"

var ErrDockerHubCredentialsExist = errors.New("dockerhub credentials exists")
var dockerHubSecretMutex sync.Mutex

// try to ensure secrets are created first if using deployment phases or a helm install
var secretAnnotations = map[string]string{
	"kots.io/creation-phase": "-9999",
	"helm.sh/hook":           "pre-install,pre-upgrade",
	"helm.sh/hook-weight":    "-9999",
}

func GetRegistryProxyInfo(license *kotsv1beta1.License, installation *kotsv1beta1.Installation, app *kotsv1beta1.Application) *RegistryProxyInfo {
	registryProxyInfo := getRegistryProxyInfoFromLicense(license)

	// DEPRECATED: getting the registry and proxy from the application spec is deprecated and should be removed in a future release
	proxyEndpoint, registryEndpoint := getRegistryProxyEndpointFromKotsApplication(app)
	if proxyEndpoint != "" {
		registryProxyInfo.Proxy = proxyEndpoint
	}
	if registryEndpoint != "" {
		registryProxyInfo.Registry = registryEndpoint
	}

	// getting the registry and proxy from the installation spec takes precedence over the application spec
	proxyEndpoint, registryEndpoint = getRegistryProxyEndpointFromKotsInstallation(installation)
	if proxyEndpoint != "" {
		registryProxyInfo.Proxy = proxyEndpoint
	}
	if registryEndpoint != "" {
		registryProxyInfo.Registry = registryEndpoint
	}

	return registryProxyInfo
}

func getRegistryProxyEndpointFromKotsInstallation(installation *kotsv1beta1.Installation) (proxyEndpoint string, registryEndpoint string) {
	if installation == nil {
		return "", ""
	}

	if installation.Spec.ReplicatedProxyDomain != "" {
		proxyEndpoint = installation.Spec.ReplicatedProxyDomain
	}

	if installation.Spec.ReplicatedRegistryDomain != "" {
		registryEndpoint = installation.Spec.ReplicatedRegistryDomain
	}

	return proxyEndpoint, registryEndpoint
}

func getRegistryProxyEndpointFromKotsApplication(kotsApplication *kotsv1beta1.Application) (proxyEndpoint string, registryEndpoint string) {
	if kotsApplication == nil {
		return "", ""
	}

	if kotsApplication.Spec.ProxyRegistryDomain != "" {
		proxyEndpoint = kotsApplication.Spec.ProxyRegistryDomain
	}

	if kotsApplication.Spec.ReplicatedRegistryDomain != "" {
		registryEndpoint = kotsApplication.Spec.ReplicatedRegistryDomain
	}

	return proxyEndpoint, registryEndpoint
}

func getRegistryProxyInfoFromLicense(license *kotsv1beta1.License) *RegistryProxyInfo {
	defaultInfo := &RegistryProxyInfo{
		Upstream: "registry.replicated.com",
		Registry: "registry.replicated.com",
		Proxy:    "proxy.replicated.com",
	}

	if license == nil {
		return defaultInfo
	}

	u, err := url.Parse(license.Spec.Endpoint)
	if err != nil {
		return defaultInfo
	}

	if u.Hostname() == "staging.replicated.app" {
		return &RegistryProxyInfo{
			Upstream: "registry.staging.replicated.com",
			Registry: "registry.staging.replicated.com",
			Proxy:    "proxy.staging.replicated.com",
		}
	}

	if strings.HasSuffix(u.Hostname(), ".okteto.repldev.com") {
		hostnameParts := strings.Split(u.Hostname(), ".")
		if len(hostnameParts) == 4 {
			parts := strings.Split(hostnameParts[0], "-")
			if len(parts) == 3 {
				namespace := parts[2]
				return &RegistryProxyInfo{
					Upstream: fmt.Sprintf("vendor-registry-v2-%s.okteto.repldev.com", namespace),
					Registry: fmt.Sprintf("vendor-registry-v2-%s.okteto.repldev.com", namespace),
					Proxy:    fmt.Sprintf("registry-proxy-%s.okteto.repldev.com", namespace),
				}
			}
		}
	}

	return defaultInfo
}

func (r *RegistryProxyInfo) ToSlice() []string {
	return []string{
		r.Proxy,
		r.Registry,
	}
}

func SecretNameFromPrefix(namePrefix string) string {
	if namePrefix == "" {
		return ""
	}

	return fmt.Sprintf("%s-registry", namePrefix)
}

func PullSecretForRegistries(registries []string, username, password string, appNamespace string, namePrefix string) (ImagePullSecrets, error) {
	dockercfgAuth := DockercfgAuth{
		Auth: base64.StdEncoding.EncodeToString([]byte(fmt.Sprintf("%s:%s", username, password))),
	}

	dockerCfgJSON := DockerCfgJSON{
		Auths: map[string]DockercfgAuth{},
	}

	for _, r := range registries {
		// we can get "host/namespace" here, which can break parts of kots that use hostname to lookup secret.
		host := strings.Split(r, "/")[0]
		dockerCfgJSON.Auths[host] = dockercfgAuth
	}

	secretData, err := json.Marshal(dockerCfgJSON)
	if err != nil {
		return ImagePullSecrets{}, errors.Wrap(err, "failed to marshal pull secret data")
	}

	secrets := ImagePullSecrets{
		AdminConsoleSecret: &corev1.Secret{
			TypeMeta: metav1.TypeMeta{
				APIVersion: "v1",
				Kind:       "Secret",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:        "kotsadm-replicated-registry",
				Namespace:   appNamespace,
				Annotations: secretAnnotations,
			},
			Type: corev1.SecretTypeDockerConfigJson,
			Data: map[string][]byte{
				".dockerconfigjson": secretData,
			},
		},
	}

	if namePrefix == "" {
		return secrets, nil
	}

	secrets.AppSecret = &corev1.Secret{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "Secret",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:        SecretNameFromPrefix(namePrefix),
			Namespace:   appNamespace,
			Annotations: secretAnnotations,
			Labels:      applicationPullSecretLabels(),
		},
		Type: corev1.SecretTypeDockerConfigJson,
		Data: map[string][]byte{
			".dockerconfigjson": secretData,
		},
	}

	return secrets, nil
}

// applicationPullSecretLabels returns the labels that should be applied to app-specific pull secrets.
// It will return nil if there are no labels to apply.
func applicationPullSecretLabels() map[string]string {
	var secretLabels map[string]string
	if util.IsEmbeddedCluster() {
		secretLabels = map[string]string{
			kotsadmtypes.DisasterRecoveryLabel:      kotsadmtypes.DisasterRecoveryLabelValueInfra,
			kotsadmtypes.DisasterRecoveryChartLabel: kotsadmtypes.DisasterRecoveryChartValue,
		}
	}

	return secretLabels
}

func EnsureDockerHubSecret(username string, password string, namespace string, clientset *kubernetes.Clientset) error {
	dockerHubSecretMutex.Lock()
	defer dockerHubSecretMutex.Unlock()

	newSecret, err := PullSecretForDockerHub(username, password, namespace)
	if err != nil {
		return errors.Wrap(err, "failed to build pull secret for dockerhub")
	}

	existingSecret, err := clientset.CoreV1().Secrets(namespace).Get(context.TODO(), DockerHubSecretName, metav1.GetOptions{})
	if err != nil {
		if !kuberneteserrors.IsNotFound(err) {
			return errors.Wrap(err, "failed to get pull secret for dockerhub")
		}

		_, err = clientset.CoreV1().Secrets(namespace).Create(context.TODO(), newSecret, metav1.CreateOptions{})
		if err != nil {
			return errors.Wrap(err, "failed to create dockerhub secret")
		}

		return nil
	}

	// Ignore error, if JSON is bad or missing, we should replace it anyway
	existingCreds, _ := GetCredentialsForRegistryFromConfigJSON(existingSecret.Data[".dockerconfigjson"], DockerHubRegistryName)
	if existingCreds.Username == username && existingCreds.Password == password {
		return ErrDockerHubCredentialsExist
	}

	existingSecret.Data = newSecret.DeepCopy().Data
	_, err = clientset.CoreV1().Secrets(namespace).Update(context.TODO(), newSecret, metav1.UpdateOptions{})
	if err != nil {
		return errors.Wrap(err, "failed to update dockerhub secret")
	}

	return nil
}

func PullSecretForDockerHub(username string, password string, kuberneteNamespace string) (*corev1.Secret, error) {
	dockercfgAuth := DockercfgAuth{
		Auth: base64.StdEncoding.EncodeToString([]byte(fmt.Sprintf("%s:%s", username, password))),
	}

	dockerCfgJSON := DockerCfgJSON{
		Auths: map[string]DockercfgAuth{
			DockerHubRegistryName: dockercfgAuth,
		},
	}

	secretData, err := json.Marshal(dockerCfgJSON)
	if err != nil {
		return nil, errors.Wrap(err, "failed to marshal pull secret data")
	}

	secret := &corev1.Secret{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "Secret",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      DockerHubSecretName,
			Namespace: kuberneteNamespace,
		},
		Type: corev1.SecretTypeDockerConfigJson,
		Data: map[string][]byte{
			".dockerconfigjson": secretData,
		},
	}

	return secret, nil
}

func GetDockerHubCredentials(clientset kubernetes.Interface, namespace string) (Credentials, error) {
	imagePullSecret, err := clientset.CoreV1().Secrets(namespace).Get(context.TODO(), DockerHubSecretName, metav1.GetOptions{})
	if err != nil {
		if kuberneteserrors.IsNotFound(err) {
			return Credentials{}, nil
		}
		return Credentials{}, errors.Wrap(err, "failed to get existing dockerhub secret")
	}

	dockerConfigJson := imagePullSecret.Data[".dockerconfigjson"]
	if len(dockerConfigJson) == 0 {
		return Credentials{}, nil
	}

	return GetCredentialsForRegistryFromConfigJSON(dockerConfigJson, DockerHubRegistryName)
}

func GetDockerHubPullSecret(clientset kubernetes.Interface, namespace string, appNamespace string, namePrefix string) (*corev1.Secret, error) {
	if namePrefix == "" {
		return nil, nil
	}

	secret, err := clientset.CoreV1().Secrets(namespace).Get(context.TODO(), DockerHubSecretName, metav1.GetOptions{})
	if err != nil {
		if kuberneteserrors.IsNotFound(err) {
			return nil, nil
		}
		return nil, errors.Wrap(err, "failed to get existing dockerhub secret")
	}

	cleanSecret := &corev1.Secret{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "Secret",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:        fmt.Sprintf("%s-%s", namePrefix, DockerHubSecretName),
			Namespace:   appNamespace,
			Annotations: secretAnnotations,
			Labels:      applicationPullSecretLabels(),
		},
		Type: corev1.SecretTypeDockerConfigJson,
		Data: secret.DeepCopy().Data,
	}

	return cleanSecret, nil
}

func GetCredentialsForRegistryFromConfigJSON(configJson []byte, registry string) (Credentials, error) {
	creds, err := GetCredentialsFromConfigJSON(configJson)
	if err != nil {
		return Credentials{}, errors.Wrap(err, "failed parse config json")
	}

	c, ok := creds[registry]
	if !ok {
		return Credentials{}, nil
	}

	return c, nil
}

func GetCredentialsFromConfigJSON(configJson []byte) (map[string]Credentials, error) {
	if len(configJson) == 0 {
		return nil, errors.New("docker config JSON data is empty")
	}

	dockerCfgJSON := DockerCfgJSON{}
	err := json.Unmarshal(configJson, &dockerCfgJSON)
	if err != nil {
		return nil, errors.Wrap(err, "failed to unmarshal config json")
	}

	result := map[string]Credentials{}
	for registry, auth := range dockerCfgJSON.Auths {
		decoded, err := base64.StdEncoding.DecodeString(auth.Auth)
		if err != nil {
			return nil, errors.Wrap(err, "failed to decode auth string")
		}

		parts := strings.SplitN(string(decoded), ":", 2)
		if len(parts) != 2 {
			return nil, errors.Errorf("expected 2 parts in the string, but found %d", len(parts))
		}

		result[registry] = Credentials{
			Username: parts[0],
			Password: parts[1],
		}
	}

	return result, nil
}
