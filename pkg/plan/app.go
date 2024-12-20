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
	"github.com/replicatedhq/kots/pkg/render"
	"github.com/replicatedhq/kots/pkg/reporting"
	"github.com/replicatedhq/kots/pkg/store"
	"github.com/replicatedhq/kots/pkg/update"
	"github.com/replicatedhq/kots/pkg/upgradeservice"
	upgradeservicetypes "github.com/replicatedhq/kots/pkg/upgradeservice/types"
	"github.com/replicatedhq/kots/pkg/util"
)

func executeAppUpgradeService(s store.Store, p *types.Plan, step *types.PlanStep) (finalError error) {
	in, ok := step.Input.(types.PlanStepInputAppUpgradeService)
	if !ok {
		return errors.New("invalid input for app upgrade service step")
	}

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
	if err := upgradeservice.Start(in.Params); err != nil {
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

func executeAppUpgrade(s store.Store, p *types.Plan, step *types.PlanStep) error {
	if err := UpdateStep(s, UpdateStepOptions{
		AppSlug:      p.AppSlug,
		VersionLabel: p.VersionLabel,
		StepID:       step.ID,
		Status:       types.StepStatusRunning,
	}); err != nil {
		return errors.Wrap(err, "update step status")
	}

	ausOutput, err := getAppUpgradeServiceOutput(p)
	if err != nil {
		return errors.Wrap(err, "get app upgrade service output")
	}
	appArchive, err := getAppArchive(ausOutput["app-version-archive"])
	if err != nil {
		return errors.Wrap(err, "get app archive")
	}
	defer os.RemoveAll(appArchive)

	skipPreflights, err := strconv.ParseBool(ausOutput["skip-preflights"])
	if err != nil {
		return errors.Wrap(err, "failed to parse is skip preflights")
	}

	sequence, err := s.CreateAppVersion(p.AppID, &p.BaseSequence, appArchive, ausOutput["source"], false, false, "", skipPreflights, render.Renderer{})
	if err != nil {
		return errors.Wrap(err, "create app version")
	}

	if p.IsAirgap {
		if err := update.RemoveAirgapUpdate(p.AppSlug, p.ChannelID, p.UpdateCursor); err != nil {
			return errors.Wrap(err, "remove airgap update")
		}
	}

	if err := filestore.GetStore().DeleteArchive(ausOutput["app-version-archive"]); err != nil {
		return errors.Wrap(err, "delete archive")
	}

	if ausOutput["preflight-result"] != "" {
		if err := s.SetPreflightResults(p.AppID, sequence, []byte(ausOutput["preflight-result"])); err != nil {
			return errors.Wrap(err, "set preflight results")
		}
	}

	if err := s.SetAppChannelChanged(p.AppID, false); err != nil {
		return errors.Wrap(err, "reset channel changed flag")
	}

	if err := s.MarkAsCurrentDownstreamVersion(p.AppID, sequence); err != nil {
		return errors.Wrap(err, "mark as current downstream version")
	}

	go operator.MustGetOperator().DeployApp(p.AppID, sequence)

	if err := UpdateStep(s, UpdateStepOptions{
		AppSlug:      p.AppSlug,
		VersionLabel: p.VersionLabel,
		StepID:       step.ID,
		Status:       types.StepStatusComplete,
	}); err != nil {
		return errors.Wrap(err, "update step status")
	}

	return nil
}

func getAppUpgradeServiceInput(s store.Store, p *types.Plan, stepID string) (*types.PlanStepInputAppUpgradeService, error) {
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

	ausParams := upgradeservicetypes.UpgradeServiceParams{
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
		UpdateECVersion:    p.NewECVersion,
		UpdateKOTSBin:      updateKOTSBin,
		UpdateAirgapBundle: updateAirgapBundle,

		CurrentECVersion: util.EmbeddedClusterVersion(),

		RegistryEndpoint:   registrySettings.Hostname,
		RegistryUsername:   registrySettings.Username,
		RegistryPassword:   registrySettings.Password,
		RegistryNamespace:  registrySettings.Namespace,
		RegistryIsReadOnly: registrySettings.IsReadOnly,

		ReportingInfo: reporting.GetReportingInfo(a.ID),
	}

	return &types.PlanStepInputAppUpgradeService{
		Params: ausParams,
	}, nil
}
