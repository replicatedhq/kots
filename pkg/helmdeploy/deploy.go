package helmdeploy

import (
	"os"
	"path/filepath"

	"github.com/pkg/errors"
	"github.com/replicatedhq/kots/pkg/util"
)

// GetV1Beta2ChartsArchive returns an archive of the v1beta2 charts to be deployed
func GetV1Beta2ChartsArchive(deployedVersionArchive string) ([]byte, error) {
	chartsDir := filepath.Join(deployedVersionArchive, "helm")
	if _, err := os.Stat(chartsDir); err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, errors.Wrap(err, "failed to stat charts dir")
	}

	archive, err := util.TGZArchive(chartsDir)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create charts archive")
	}

	return archive, nil
}
