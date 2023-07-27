package upstream

import (
	"os"
	"path/filepath"

	"github.com/pkg/errors"
	kotsv1beta1 "github.com/replicatedhq/kotskinds/apis/kots/v1beta1"
	"k8s.io/client-go/kubernetes/scheme"
)

func LoadIdentity(upstreamDir string) (*kotsv1beta1.Identity, error) {
	var identitySpec *kotsv1beta1.Identity

	err := filepath.Walk(upstreamDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() {
			return nil
		}

		content, err := os.ReadFile(path)
		if err != nil {
			return err
		}

		decode := scheme.Codecs.UniversalDeserializer().Decode
		obj, gvk, err := decode(content, nil, nil)
		if err != nil {
			return nil
		}

		if gvk.Group == "kots.io" && gvk.Version == "v1beta1" && gvk.Kind == "Identity" {
			identitySpec = obj.(*kotsv1beta1.Identity)
		}

		return nil
	})
	if err != nil {
		return nil, errors.Wrap(err, "failed to walk archive dir")
	}

	return identitySpec, nil
}

func LoadIdentityConfig(upstreamDir string) (*kotsv1beta1.IdentityConfig, error) {
	filename := filepath.Join(upstreamDir, "userdata", "identityconfig.yaml")
	if _, err := os.Stat(filename); os.IsNotExist(err) {
		return nil, nil
	} else if err != nil {
		return nil, errors.Wrap(err, "failed to stat identity Config")
	}

	content, err := os.ReadFile(filename)
	if err != nil {
		return nil, errors.Wrap(err, "failed to read existing identity Config")
	}

	decode := scheme.Codecs.UniversalDeserializer().Decode
	obj, gvk, err := decode(content, nil, nil)
	if err != nil {
		return nil, errors.Wrap(err, "failed to decode identity Config")
	}

	if gvk.Group == "kots.io" && gvk.Version == "v1beta1" && gvk.Kind == "IdentityConfig" {
		return obj.(*kotsv1beta1.IdentityConfig), nil
	}

	return nil, errors.Errorf("unexpected gvk in identity Config file: %s/%s/%s", gvk.Group, gvk.Version, gvk.Kind)
}

func SaveIdentityConfig(identityConfig *kotsv1beta1.IdentityConfig, upstreamDir string) error {
	filename := filepath.Join(upstreamDir, "userdata", "identityconfig.yaml")
	err := os.WriteFile(filename, mustMarshalIdentityConfig(identityConfig), 0644)
	if err != nil {
		return errors.Wrap(err, "failed to write identity config")
	}
	return nil
}
