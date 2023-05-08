package diff

import (
	"os"
	"path/filepath"

	"github.com/pkg/errors"
	"github.com/replicatedhq/kots/pkg/util"
)

// GetRenderedV1Beta2FileMap returns a map of the rendered v1beta2 charts to be deployed
func GetRenderedV1Beta2FileMap(deployedVersionArchive, downstream string) (map[string][]byte, error) {
	chartsDir := filepath.Join(deployedVersionArchive, "rendered", downstream, "helm")
	if _, err := os.Stat(chartsDir); err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, errors.Wrap(err, "failed to stat charts dir")
	}

	filesMap, err := util.GetFilesMap(chartsDir)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get files map")
	}

	return filesMap, nil
}
