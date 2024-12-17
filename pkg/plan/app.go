package plan

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/phayes/freeport"
	"github.com/pkg/errors"
	"github.com/replicatedhq/kots/pkg/filestore"
	"github.com/replicatedhq/kots/pkg/kotsutil"
	"github.com/replicatedhq/kots/pkg/logger"
	"github.com/replicatedhq/kots/pkg/operator"
	"github.com/replicatedhq/kots/pkg/plan/types"
	"github.com/replicatedhq/kots/pkg/replicatedapp"
	"github.com/replicatedhq/kots/pkg/reporting"
	"github.com/replicatedhq/kots/pkg/store"
	"github.com/replicatedhq/kots/pkg/tasks"
	"github.com/replicatedhq/kots/pkg/update"
	"github.com/replicatedhq/kots/pkg/upgradeservice"
	upgradeservicetask "github.com/replicatedhq/kots/pkg/upgradeservice/task"
	upgradeservicetypes "github.com/replicatedhq/kots/pkg/upgradeservice/types"
	"github.com/replicatedhq/kots/pkg/util"
)

func executeAppUpgradeService(s store.Store, p *types.Plan, step *types.PlanStep) (finalError error) {
	// TODO NOW: is this step needed? use plan step status instead? upgrade service can update status via api
	if err := upgradeservicetask.SetStatusStarting(p.AppSlug, "Preparing..."); err != nil {
		return errors.Wrap(err, "set app upgrade service task status")
	}

	finishedChan := make(chan error)
	defer close(finishedChan)

	tasks.StartTaskMonitor(upgradeservicetask.GetID(p.AppSlug), finishedChan)
	defer func() {
		if finalError != nil {
			logger.Error(finalError)
		}
		finishedChan <- finalError
	}()

	// TODO (@salah): don't run as separate process if kots version did not change?
	params, err := getAppUpgradeServiceParams(s, p, step.ID)
	if err != nil {
		return err
	}
	if err := upgradeservice.Start(*params); err != nil {
		return errors.Wrap(err, "start app upgrade service")
	}

	return nil
}

func executeAppUpgrade(p *types.Plan, step *types.PlanStep) error {
	ausOutput, err := getAppUpgradeServiceOutput(p)
	if err != nil {
		return errors.Wrap(err, "get app upgrade service output")
	}

	if err := operator.MustGetOperator().DeployEC2App(ausOutput); err != nil {
		return errors.Wrap(err, "deploy app")
	}
	return nil
}

func getAppUpgradeServiceParams(s store.Store, p *types.Plan, stepID string) (*upgradeservicetypes.UpgradeServiceParams, error) {
	a, err := s.GetAppFromSlug(p.AppSlug)
	if err != nil {
		return nil, errors.Wrap(err, "get app from slug")
	}

	registrySettings, err := s.GetRegistryDetailsForApp(a.ID)
	if err != nil {
		return nil, errors.Wrap(err, "get registry details for app")
	}

	baseArchive, baseSequence, err := s.GetAppVersionBaseArchive(a.ID, p.VersionLabel)
	if err != nil {
		return nil, errors.Wrap(err, "get app version base archive")
	}

	nextSequence, err := s.GetNextAppSequence(a.ID)
	if err != nil {
		return nil, errors.Wrap(err, "get next app sequence")
	}

	source := "Upstream Update"
	if a.IsAirgap {
		source = "Airgap Update"
	}

	license, err := kotsutil.LoadLicenseFromBytes([]byte(a.License))
	if err != nil {
		return nil, errors.Wrap(err, "parse app license")
	}

	var updateECVersion string
	var updateKOTSBin string
	var updateAirgapBundle string

	if a.IsAirgap {
		au, err := update.GetAirgapUpdate(a.Slug, p.ChannelID, p.UpdateCursor)
		if err != nil {
			return nil, errors.Wrap(err, "get airgap update")
		}
		updateAirgapBundle = au
		kb, err := kotsutil.GetKOTSBinFromAirgapBundle(au)
		if err != nil {
			return nil, errors.Wrap(err, "get kots binary from airgap bundle")
		}
		updateKOTSBin = kb
		ecv, err := kotsutil.GetECVersionFromAirgapBundle(au)
		if err != nil {
			return nil, errors.Wrap(err, "get kots version from binary")
		}
		updateECVersion = ecv
	} else {
		// TODO (@salah): revert this
		// TODO (@salah): no need to download if the kots version did not change?
		// TODO (@salah): how to know if the kots version did not change? (i think there's a replicated.app endpoint for this)
		// kb, err := replicatedapp.DownloadKOTSBinary(license, p.VersionLabel)
		// if err != nil {
		// 	return nil, errors.Wrap(err, "download kots binary")
		// }
		updateKOTSBin = kotsutil.GetKOTSBinPath()
		ecv, err := replicatedapp.GetECVersionForRelease(license, p.VersionLabel)
		if err != nil {
			return nil, errors.Wrap(err, "get kots version for release")
		}
		updateECVersion = ecv
	}

	port, err := freeport.GetFreePort()
	if err != nil {
		return nil, errors.Wrap(err, "get free port")
	}

	return &upgradeservicetypes.UpgradeServiceParams{
		Port:       fmt.Sprintf("%d", port),
		PlanStepID: stepID,

		AppID:       a.ID,
		AppSlug:     a.Slug,
		AppName:     a.Name,
		AppIsAirgap: a.IsAirgap,
		AppIsGitOps: a.IsGitOps,
		AppLicense:  a.License,
		AppArchive:  baseArchive,

		Source:       source,
		BaseSequence: baseSequence,
		NextSequence: nextSequence,

		UpdateVersionLabel: p.VersionLabel,
		UpdateCursor:       p.UpdateCursor,
		UpdateChannelID:    p.ChannelID,
		UpdateECVersion:    updateECVersion,
		UpdateKOTSBin:      updateKOTSBin,
		UpdateAirgapBundle: updateAirgapBundle,

		CurrentECVersion: util.EmbeddedClusterVersion(),

		RegistryEndpoint:   registrySettings.Hostname,
		RegistryUsername:   registrySettings.Username,
		RegistryPassword:   registrySettings.Password,
		RegistryNamespace:  registrySettings.Namespace,
		RegistryIsReadOnly: registrySettings.IsReadOnly,

		ReportingInfo: reporting.GetReportingInfo(a.ID),
	}, nil
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

// getAppArchive returns the archive created by the app upgrade service step.
// the caller is responsible for deleting the archive.
func getAppArchive(p *types.Plan) (string, error) {
	ausOutput, err := getAppUpgradeServiceOutput(p)
	if err != nil {
		return "", errors.Wrap(err, "get app upgrade service output")
	}

	path, ok := ausOutput["app-version-archive"]
	if !ok || path == "" {
		return "", errors.New("app version archive not found in app upgrade service step output")
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
