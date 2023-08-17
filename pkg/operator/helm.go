package operator

import (
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/pkg/errors"
	"github.com/replicatedhq/kots/pkg/util"
)

func getChartsImagePullSecrets(deployedVersionArchive string) ([]string, error) {
	archiveChartDir := filepath.Join(deployedVersionArchive, "overlays", "midstream", "charts")
	chartDirs, err := ioutil.ReadDir(archiveChartDir)
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
		secretData, err := ioutil.ReadFile(secretFilename)
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
