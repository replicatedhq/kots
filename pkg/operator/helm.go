package operator

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/pkg/errors"
)

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

		secrets := strings.Split(string(secretData), "\n---\n")
		imagePullSecrets = append(imagePullSecrets, secrets...)
	}

	return imagePullSecrets, nil
}
