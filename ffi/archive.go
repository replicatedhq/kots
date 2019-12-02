package main

import (
	"io/ioutil"
	"os"

	"github.com/mholt/archiver"
	"github.com/pkg/errors"
	kotsv1beta1 "github.com/replicatedhq/kots/kotskinds/apis/kots/v1beta1"
	"k8s.io/client-go/kubernetes/scheme"
)

func extractArchive(rootPath, fromArchivePath string) (*archiver.TarGz, error) {
	// extract the current archive to this root
	tarGz := &archiver.TarGz{
		Tar: &archiver.Tar{
			ImplicitTopLevelFolder: false,
		},
	}
	if err := tarGz.Unarchive(fromArchivePath, rootPath); err != nil {
		return nil, err
	}

	return tarGz, nil
}

func readCursorFromPath(installationFilePath string) (string, error) {
	installation, err := loadInstallationFromPath(installationFilePath)
	if err != nil {
		if os.IsNotExist(err) {
			return "", nil
		}
		return "", errors.Wrap(err, "failed to read installation file")
	}
	return installation.Spec.UpdateCursor, nil
}

func loadInstallationFromPath(installationFilePath string) (*kotsv1beta1.Installation, error) {
	installationData, err := ioutil.ReadFile(installationFilePath)
	if err != nil {
		return nil, errors.Wrap(err, "failed to read installation file")
	}

	decode := scheme.Codecs.UniversalDeserializer().Decode
	obj, gvk, err := decode([]byte(installationData), nil, nil)
	if err != nil {
		return nil, errors.Wrap(err, "failed to devode installation data")
	}

	if gvk.Group != "kots.io" || gvk.Version != "v1beta1" || gvk.Kind != "Installation" {
		return nil, errors.Errorf("unexpected GVK: %s", gvk.String())
	}

	return obj.(*kotsv1beta1.Installation), nil
}
