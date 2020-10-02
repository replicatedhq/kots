package registry

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/url"
	"strings"

	"github.com/pkg/errors"
	kotsv1beta1 "github.com/replicatedhq/kots/kotskinds/apis/kots/v1beta1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type RegistryProxyInfo struct {
	Registry string
	Proxy    string
}

type DockercfgAuth struct {
	Auth string `json:"auth,omitempty"`
}

type DockerCfgJSON struct {
	Auths map[string]DockercfgAuth `json:"auths"`
}

func ProxyEndpointFromLicense(license *kotsv1beta1.License) *RegistryProxyInfo {
	defaultInfo := &RegistryProxyInfo{
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

	switch u.Hostname() {
	case "staging.replicated.app":
		return &RegistryProxyInfo{
			Registry: "registry.staging.replicated.com",
			Proxy:    "proxy.staging.replicated.com",
		}
	case "replicated-app":
		return &RegistryProxyInfo{
			Registry: "registry:3000", // TODO: not real info
			Proxy:    "registry-proxy:3000",
		}
	default:
		return defaultInfo
	}
}

func (r *RegistryProxyInfo) ToSlice() []string {
	return []string{
		r.Proxy,
		r.Registry,
	}
}

func PullSecretForRegistries(registries []string, username, password string, kuberneteNamespace string) (*corev1.Secret, error) {
	dockercfgAuth := DockercfgAuth{
		Auth: base64.StdEncoding.EncodeToString([]byte(fmt.Sprintf("%s:%s", username, password))),
	}

	dockerCfgJSON := DockerCfgJSON{
		Auths: map[string]DockercfgAuth{},
	}

	for _, r := range registries {
		dockerCfgJSON.Auths[r] = dockercfgAuth
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
			Name:      "kotsadm-replicated-registry",
			Namespace: kuberneteNamespace,
		},
		Type: corev1.SecretTypeDockerConfigJson,
		Data: map[string][]byte{
			".dockerconfigjson": secretData,
		},
	}

	return secret, nil
}

func GetCredentialsForRegistry(configJson string, registry string) (string, string, error) {
	dockerCfgJSON := DockerCfgJSON{}
	err := json.Unmarshal([]byte(configJson), &dockerCfgJSON)
	if err != nil {
		return "", "", errors.Wrap(err, "failed to unmarshal config json")
	}

	auth, ok := dockerCfgJSON.Auths[registry]
	if !ok {
		return "", "", nil
	}

	decoded, err := base64.StdEncoding.DecodeString(auth.Auth)
	if err != nil {
		return "", "", errors.Wrap(err, "failed to decode auth string")
	}

	parts := strings.Split(string(decoded), ":")
	if len(parts) != 2 {
		return "", "", errors.Errorf("expected 2 parts in the string, but found %d", len(parts))
	}

	return parts[0], parts[1], nil
}
