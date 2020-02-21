package upstream

import (
	"io/ioutil"
	"os"
	"path"
	"path/filepath"

	"github.com/pkg/errors"
	kotsv1beta1 "github.com/replicatedhq/kots/kotskinds/apis/kots/v1beta1"
	"k8s.io/client-go/kubernetes/scheme"
)

func LoadApplication(upstreamDir string) (*kotsv1beta1.Application, error) {
	var application *kotsv1beta1.Application

	err := filepath.Walk(upstreamDir,
		func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}

			if info.IsDir() {
				return nil
			}

			content, err := ioutil.ReadFile(path)
			if err != nil {
				return err
			}

			decode := scheme.Codecs.UniversalDeserializer().Decode
			obj, gvk, err := decode(content, nil, nil)
			if err != nil {
				return nil
			}

			if gvk.Group == "kots.io" && gvk.Version == "v1beta1" && gvk.Kind == "Application" {
				application = obj.(*kotsv1beta1.Application)
			}

			return nil
		})

	if err != nil {
		return nil, errors.Wrap(err, "failed to walk archive dir")
	}

	return application, nil
}

func LoadInstallation(upstreamDir string) (*kotsv1beta1.Installation, error) {
	content, err := ioutil.ReadFile(path.Join(upstreamDir, "userdata", "installation.yaml"))
	if err != nil {
		return nil, errors.Wrap(err, "failed to read existing installation")
	}

	decode := scheme.Codecs.UniversalDeserializer().Decode
	obj, gvk, err := decode(content, nil, nil)
	if err != nil {
		return nil, errors.Wrap(err, "failed to decode installation")
	}

	if gvk.Group == "kots.io" && gvk.Version == "v1beta1" && gvk.Kind == "Installation" {
		return obj.(*kotsv1beta1.Installation), nil
	}

	return nil, errors.Errorf("unexpected gvk in installation file: %s/%s/%s", gvk.Group, gvk.Version, gvk.Kind)
}

func SaveInstallation(installation *kotsv1beta1.Installation, upstreamDir string) error {
	filename := path.Join(upstreamDir, "userdata", "installation.yaml")
	err := ioutil.WriteFile(filename, mustMarshalInstallation(installation), 0644)
	if err != nil {
		return errors.Wrap(err, "failed to write installation")
	}
	return nil
}
