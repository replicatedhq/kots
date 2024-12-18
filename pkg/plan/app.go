package plan

import (
	"fmt"
	"os"
	"strconv"

	"github.com/phayes/freeport"
	"github.com/pkg/errors"
	"github.com/replicatedhq/kots/pkg/filestore"
	"github.com/replicatedhq/kots/pkg/kotsutil"
	"github.com/replicatedhq/kots/pkg/operator"
	"github.com/replicatedhq/kots/pkg/plan/types"
	"github.com/replicatedhq/kots/pkg/reporting"
	"github.com/replicatedhq/kots/pkg/store"
	"github.com/replicatedhq/kots/pkg/update"
	"github.com/replicatedhq/kots/pkg/upgradeservice"
	upgradeservicetypes "github.com/replicatedhq/kots/pkg/upgradeservice/types"
	"github.com/replicatedhq/kots/pkg/util"
)

func executeAppUpgradeService(s store.Store, p *types.Plan, step *types.PlanStep) (finalError error) {
	if err := UpdateStep(s, UpdateStepOptions{
		AppSlug:           p.AppSlug,
		VersionLabel:      p.VersionLabel,
		StepID:            step.ID,
		Status:            types.StepStatusStarting,
		StatusDescription: "Preparing...",
	}); err != nil {
		return errors.Wrap(err, "update step status")
	}

	// TODO (@salah): don't run as separate process if kots version did not change?
	params, err := getAppUpgradeServiceParams(s, p, step.ID)
	if err != nil {
		return err
	}
	if err := upgradeservice.Start(*params); err != nil {
		return errors.Wrap(err, "start app upgrade service")
	}

	if err := UpdateStep(s, UpdateStepOptions{
		AppSlug:      p.AppSlug,
		VersionLabel: p.VersionLabel,
		StepID:       step.ID,
		Status:       types.StepStatusRunning,
	}); err != nil {
		return errors.Wrap(err, "update step status")
	}

	return nil
}

func executeAppUpgrade(p *types.Plan, step *types.PlanStep) error {
	ausOutput, err := getAppUpgradeServiceOutput(p)
	if err != nil {
		return errors.Wrap(err, "get app upgrade service output")
	}
	appArchive, err := getAppArchive(ausOutput["app-version-archive"])
	if err != nil {
		return errors.Wrap(err, "get app archive")
	}
	defer os.RemoveAll(appArchive)

	baseSequence, err := strconv.ParseInt(ausOutput["base-sequence"], 10, 64)
	if err != nil {
		return errors.Wrap(err, "parse base sequence")
	}

	if err := operator.MustGetOperator().DeployEC2App(operator.DeployEC2AppOptions{
		AppID:           ausOutput["app-id"],
		AppSlug:         ausOutput["app-slug"],
		AppArchive:      appArchive,
		BaseSequence:    baseSequence,
		Source:          ausOutput["source"],
		IsAirgap:        ausOutput["is-airgap"] == "true",
		ChannelID:       ausOutput["channel-id"],
		UpdateCursor:    ausOutput["update-cursor"],
		SkipPreflights:  ausOutput["skip-preflights"] == "true",
		PreflightResult: ausOutput["preflight-result"],
	}); err != nil {
		return errors.Wrap(err, "deploy app")
	}

	if err := filestore.GetStore().DeleteArchive(ausOutput["app-version-archive"]); err != nil {
		return errors.Wrap(err, "delete archive")
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
	} else {
		// TODO (@salah): revert this
		// TODO (@salah): no need to download if the kots version did not change?
		// TODO (@salah): how to know if the kots version did not change? (i think there's a replicated.app endpoint for this)
		// kb, err := replicatedapp.DownloadKOTSBinary(license, p.VersionLabel)
		// if err != nil {
		// 	return nil, errors.Wrap(err, "download kots binary")
		// }
		updateKOTSBin = kotsutil.GetKOTSBinPath()
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
		UpdateECVersion:    p.ECVersion,
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
