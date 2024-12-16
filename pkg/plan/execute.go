package plan

import (
	"context"
	"fmt"
	"time"

	"github.com/phayes/freeport"
	"github.com/pkg/errors"
	ecv1beta1 "github.com/replicatedhq/embedded-cluster/kinds/apis/v1beta1"
	"github.com/replicatedhq/kots/pkg/embeddedcluster"
	"github.com/replicatedhq/kots/pkg/k8sutil"
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
	"github.com/replicatedhq/kots/pkg/websocket"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func Execute(ctx context.Context, p *types.Plan) error {
	if p == nil {
		return nil
	}

	for _, step := range p.Steps {
		switch step.Type {
		case types.StepTypeUpgradeService:
			if err := executeUpgradeService(ctx, step); err != nil {
				return err
			}
		case types.StepTypeECUpgrade:
			if err := executeECUpgrade(ctx, step); err != nil {
				return err
			}
		case types.StepTypeAppUpgrade:
			if err := executeAppUpgrade(step); err != nil {
				return err
			}
		default:
			return errors.Errorf("unknown step type %q", step.Type)
		}
	}

	return nil
}

func executeUpgradeService(ctx context.Context, step types.PlanStep) (finalError error) {
	in, ok := step.Input.(types.PlanStepInputUpgradeService)
	if !ok {
		return errors.New("invalid input for upgrade service step")
	}

	if err := startUpgradeService(in); err != nil {
		return err
	}

	// block until the upgrade service process exits
	if err := upgradeservice.Wait(in.AppSlug); err != nil {
		return errors.Wrap(err, "wait for upgrade service")
	}

	return nil
}

func startUpgradeService(in types.PlanStepInputUpgradeService) (finalError error) {
	// TODO NOW: is this set status needed?
	if err := upgradeservicetask.SetStatusStarting(in.AppSlug, "Preparing..."); err != nil {
		return errors.Wrap(err, "set upgrade service task status")
	}

	finishedChan := make(chan error)
	defer close(finishedChan)

	tasks.StartTaskMonitor(upgradeservicetask.GetID(in.AppSlug), finishedChan)
	defer func() {
		if finalError != nil {
			logger.Error(finalError)
		}
		finishedChan <- finalError
	}()

	params, err := getUpgradeServiceParams(store.GetStore(), in)
	if err != nil {
		return err
	}
	if err := upgradeservice.Start(*params); err != nil {
		return errors.Wrap(err, "start upgrade service")
	}

	return nil
}

func getUpgradeServiceParams(s store.Store, in types.PlanStepInputUpgradeService) (*upgradeservicetypes.UpgradeServiceParams, error) {
	a, err := store.GetStore().GetAppFromSlug(in.AppSlug)
	if err != nil {
		return nil, errors.Wrap(err, "get app from slug")
	}

	registrySettings, err := s.GetRegistryDetailsForApp(a.ID)
	if err != nil {
		return nil, errors.Wrap(err, "get registry details for app")
	}

	baseArchive, baseSequence, err := s.GetAppVersionBaseArchive(a.ID, in.VersionLabel)
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
		au, err := update.GetAirgapUpdate(a.Slug, in.ChannelID, in.UpdateCursor)
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
		kb, err := replicatedapp.DownloadKOTSBinary(license, in.VersionLabel)
		if err != nil {
			return nil, errors.Wrap(err, "download kots binary")
		}
		updateKOTSBin = kb
		ecv, err := replicatedapp.GetECVersionForRelease(license, in.VersionLabel)
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
		Port: fmt.Sprintf("%d", port),

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

		UpdateVersionLabel: in.VersionLabel,
		UpdateCursor:       in.UpdateCursor,
		UpdateChannelID:    in.ChannelID,
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

func executeECUpgrade(ctx context.Context, step types.PlanStep) error {
	in, ok := step.Input.(types.PlanStepInputECUpgrade)
	if !ok {
		return errors.New("invalid input for ec upgrade step")
	}

	a, err := store.GetStore().GetAppFromSlug(in.AppSlug)
	if err != nil {
		return errors.Wrap(err, "get app from slug")
	}

	license, err := kotsutil.LoadLicenseFromBytes([]byte(a.License))
	if err != nil {
		return errors.Wrap(err, "parse app license")
	}

	kbClient, err := k8sutil.GetKubeClient(ctx)
	if err != nil {
		return errors.Wrap(err, "get kubeclient")
	}

	current, err := embeddedcluster.GetCurrentInstallation(ctx, kbClient)
	if err != nil {
		return errors.Wrap(err, "get current installation")
	}

	newInstall := &ecv1beta1.Installation{
		TypeMeta: metav1.TypeMeta{
			APIVersion: ecv1beta1.GroupVersion.String(),
			Kind:       "Installation",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: time.Now().Format("20060102150405"),
			Labels: map[string]string{
				"replicated.com/disaster-recovery": "ec-install",
			},
		},
		Spec: current.Spec,
	}
	newInstall.Spec.Artifacts = in.Artifacts
	newInstall.Spec.Config = in.ECConfig
	newInstall.Spec.LicenseInfo = &ecv1beta1.LicenseInfo{IsDisasterRecoverySupported: license.Spec.IsDisasterRecoverySupported}

	if err := websocket.UpgradeCluster(newInstall); err != nil {
		return errors.Wrap(err, "upgrade cluster")
	}

	return nil
}

func executeAppUpgrade(step types.PlanStep) error {
	in, ok := step.Input.(types.PlanStepInputAppUpgrade)
	if !ok {
		return errors.New("invalid input for app upgrade step")
	}

	a, err := store.GetStore().GetAppFromSlug(in.AppSlug)
	if err != nil {
		return errors.Wrap(err, "get app from slug")
	}

	source := "Upstream Update"
	if a.IsAirgap {
		source = "Airgap Update"
	}

	deployOpts := operator.DeployEC2AppOptions{
		AppID:                        a.ID,
		AppSlug:                      a.Slug,
		AppVersionArchive:            in.AppArchive,
		BaseSequence:                 in.BaseSequence,
		VersionLabel:                 in.VersionLabel,
		Source:                       source,
		IsAirgap:                     a.IsAirgap,
		ChannelID:                    in.ChannelID,
		UpdateCursor:                 in.UpdateCursor,
		SkipPreflights:               false, // TODO (@salah)
		ContinueWithFailedPreflights: false, // TODO (@salah)
		PreflightResult:              "",    // TODO (@salah)
	}

	if err := operator.MustGetOperator().DeployEC2App(deployOpts); err != nil {
		return errors.Wrap(err, "deploy app")
	}

	return nil
}
