package plan

import (
	"encoding/json"
	"os"

	"github.com/pkg/errors"
	apptypes "github.com/replicatedhq/kots/pkg/app/types"
	"github.com/replicatedhq/kots/pkg/filestore"
	"github.com/replicatedhq/kots/pkg/kotsutil"
	"github.com/replicatedhq/kots/pkg/plan/types"
	"github.com/replicatedhq/kots/pkg/replicatedapp"
	"github.com/replicatedhq/kots/pkg/update"
	"github.com/replicatedhq/kots/pkg/util"
)

func getECVersionForRelease(a *apptypes.App, versionLabel, channelID, updateCursor string) (string, error) {
	license, err := kotsutil.LoadLicenseFromBytes([]byte(a.License))
	if err != nil {
		return "", errors.Wrap(err, "parse app license")
	}

	var ecVersion string
	if a.IsAirgap {
		au, err := update.GetAirgapUpdate(a.Slug, channelID, updateCursor)
		if err != nil {
			return "", errors.Wrap(err, "get airgap update")
		}
		ecv, err := kotsutil.GetECVersionFromAirgapBundle(au)
		if err != nil {
			return "", errors.Wrap(err, "get kots version from binary")
		}
		ecVersion = ecv
	} else {
		ecv, err := replicatedapp.GetECVersionForRelease(license, versionLabel)
		if err != nil {
			return "", errors.Wrap(err, "get kots version for release")
		}
		ecVersion = ecv
	}

	return ecVersion, nil
}

func getAppUpgradeServiceOutput(p *types.Plan) (map[string]string, error) {
	var ausOutput map[string]string
	for _, s := range p.Steps {
		if s.Type != types.StepTypeAppUpgradeService {
			continue
		}
		output := s.Output.(string)
		if output == "" {
			return nil, errors.New("app upgrade service step output not found")
		}
		if err := json.Unmarshal([]byte(output), &ausOutput); err != nil {
			return nil, errors.Wrap(err, "unmarshal app upgrade service step output")
		}
		break
	}
	if ausOutput == nil {
		return nil, errors.New("app upgrade service step output not found")
	}
	return ausOutput, nil
}

func getAppArchive(path string) (string, error) {
	if path == "" {
		return "", errors.New("path is empty")
	}

	tgzArchive, err := filestore.GetStore().ReadArchive(path)
	if err != nil {
		return "", errors.Wrap(err, "read archive")
	}
	defer os.RemoveAll(tgzArchive)

	archiveDir, err := os.MkdirTemp("", "kotsadm")
	if err != nil {
		return "", errors.Wrap(err, "create temp dir")
	}

	if err := util.ExtractTGZArchive(tgzArchive, archiveDir); err != nil {
		return "", errors.Wrap(err, "extract app archive")
	}

	return archiveDir, nil
}
