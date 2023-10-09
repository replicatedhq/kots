package operator

import (
	"os"
	"path/filepath"

	"github.com/pkg/errors"
	"github.com/replicatedhq/kots/pkg/util"
)

func getImagePullSecrets(deployedVersionArchive string) ([]string, error) {
	imagePullSecrets := []string{}

	secretFilename := filepath.Join(deployedVersionArchive, "overlays", "midstream", "secret.yaml")
	_, err := os.Stat(secretFilename)
	if err != nil && !os.IsNotExist(err) {
		return nil, errors.Wrap(err, "failed to os stat image pull secret file")
	}
	if err == nil {
		b, err := os.ReadFile(secretFilename)
		if err != nil {
			return nil, errors.Wrap(err, "failed to read image pull secret file")
		}
		secrets := util.ConvertToSingleDocs(b)
		for _, secret := range secrets {
			imagePullSecrets = append(imagePullSecrets, string(secret))
		}
	}

	chartPullSecrets, err := getChartsImagePullSecrets(deployedVersionArchive)
	if err != nil {
		return nil, errors.Wrap(err, "failed to read image pull secret files from charts")
	}
	imagePullSecrets = append(imagePullSecrets, chartPullSecrets...)
	imagePullSecrets = deduplicateSecrets(imagePullSecrets)

	return imagePullSecrets, nil
}

func getChartsImagePullSecrets(deployedVersionArchive string) ([]string, error) {
	archiveChartDir := filepath.Join(deployedVersionArchive, "overlays", "midstream", "charts")
	chartDirs, err := os.ReadDir(archiveChartDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, errors.Wrap(err, "failed to read charts directory")
	}

	imagePullSecrets := []string{}
	for _, chartDir := range chartDirs {
		if !chartDir.IsDir() {
			continue
		}

		secretFilename := filepath.Join(archiveChartDir, chartDir.Name(), "secret.yaml")
		secretData, err := os.ReadFile(secretFilename)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return nil, errors.Wrap(err, "failed to read helm tar.gz file")
		}

		secrets := util.ConvertToSingleDocs(secretData)
		for _, secret := range secrets {
			imagePullSecrets = append(imagePullSecrets, string(secret))
		}
	}

	return imagePullSecrets, nil
}
