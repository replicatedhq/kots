package midstream

import (
	"os"
	"path/filepath"

	"github.com/pkg/errors"
	"github.com/replicatedhq/kots/pkg/docker/registry"
	registrytypes "github.com/replicatedhq/kots/pkg/registry/types"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes/scheme"
)

func LoadPrivateRegistryInfo(archivePath string) (*registrytypes.RegistrySettings, error) {
	filename := filepath.Join(archivePath, "overlays", "midstream", secretFilename)
	secretData, err := os.ReadFile(filename)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, errors.Wrap(err, "failed to load pull secret file")
	}

	decode := scheme.Codecs.UniversalDeserializer().Decode
	obj, gvk, err := decode(secretData, nil, nil)
	if err != nil {
		return nil, errors.Wrap(err, "failed to decode image pull secret")
	}

	if gvk.Group != "" || gvk.Version != "v1" || gvk.Kind != "Secret" {
		return nil, errors.Errorf("unexpected secret GVK: %s", gvk.String())
	}

	secret := obj.(*corev1.Secret)
	if secret.Type != "kubernetes.io/dockerconfigjson" {
		return nil, errors.Errorf("unexpected secret type: %s", secret.Type)
	}

	hosts, err := registry.GetCredentialsFromConfigJSON(secret.Data[".dockerconfigjson"])
	if err != nil {
		return nil, errors.Wrap(err, "failed to parse docker config json")
	}

	// If there are 0 items, then there is no registry info (this should never happen)
	// If there are 2 items, then this is the default replicated and proxy registries.
	// If there is 1 item, then this was set on the registry settings page
	if len(hosts) != 1 {
		return nil, nil
	}

	rs := &registrytypes.RegistrySettings{}
	for host, creds := range hosts {
		rs.Hostname = host
		rs.Username = creds.Username
		rs.Password = creds.Password
	}
	return rs, nil
}
