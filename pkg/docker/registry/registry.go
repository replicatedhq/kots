package registry

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/url"

	"github.com/pkg/errors"
	kotsv1beta1 "github.com/replicatedhq/kots/kotskinds/apis/kots/v1beta1"
	"github.com/replicatedhq/kots/pkg/kotsadm/types"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type RegistryProxyInfo struct {
	Registry string
	Proxy    string
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

func PullSecretForRegistries(registries []string, username, password string, namespace string) (*corev1.Secret, error) {
	dockercfgAuth := struct {
		Auth string `json:"auth,omitempty"`
	}{
		Auth: base64.StdEncoding.EncodeToString([]byte(fmt.Sprintf("%s:%s", username, password))),
	}

	dockerCfgJSON := struct {
		Auths map[string]interface{} `json:"auths"`
	}{
		Auths: map[string]interface{}{},
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
			Namespace: namespace,
			Labels:    types.GetKotsadmLabels(),
		},
		Type: corev1.SecretTypeDockerConfigJson,
		Data: map[string][]byte{
			".dockerconfigjson": secretData,
		},
	}

	return secret, nil
}
