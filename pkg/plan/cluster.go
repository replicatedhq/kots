package plan

import (
	"context"
	"reflect"
	"time"

	"github.com/pkg/errors"
	ecv1beta1 "github.com/replicatedhq/embedded-cluster/kinds/apis/v1beta1"
	apptypes "github.com/replicatedhq/kots/pkg/app/types"
	"github.com/replicatedhq/kots/pkg/embeddedcluster"
	"github.com/replicatedhq/kots/pkg/kotsutil"
	"github.com/replicatedhq/kots/pkg/plan/types"
	"github.com/replicatedhq/kots/pkg/store"
	"github.com/replicatedhq/kots/pkg/websocket"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	kbclient "sigs.k8s.io/controller-runtime/pkg/client"
	k8syaml "sigs.k8s.io/yaml"
)

func executeECUpgrade(s store.Store, p *types.Plan, step *types.PlanStep) error {
	in, ok := step.Input.(types.PlanStepInputECUpgrade)
	if !ok {
		return errors.New("invalid input for embedded cluster upgrade step")
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
		Spec: in.CurrentECInstallation.Spec,
	}
	newInstall.Spec.Artifacts = embeddedcluster.GetArtifactsFromInstallation(in.CurrentKOTSInstallation)
	newInstall.Spec.Config = &in.NewECConfigSpec
	newInstall.Spec.LicenseInfo = &ecv1beta1.LicenseInfo{IsDisasterRecoverySupported: in.IsDisasterRecoverySupported}

	if err := websocket.UpgradeCluster(newInstall, p.AppSlug, p.VersionLabel, step.ID); err != nil {
		return errors.Wrap(err, "upgrade cluster")
	}

	return nil
}

func requiresECUpgrade(kcli kbclient.Client, newSpec *ecv1beta1.ConfigSpec) (bool, error) {
	currInstall, err := embeddedcluster.GetCurrentInstallation(context.Background(), kcli)
	if err != nil {
		return false, errors.Wrap(err, "get current embedded cluster installation")
	}
	currSpec := currInstall.Spec.Config

	if currSpec.Version != newSpec.Version {
		return true, nil
	}
	if currSpec.BinaryOverrideURL != newSpec.BinaryOverrideURL {
		return true, nil
	}
	if currSpec.MetadataOverrideURL != newSpec.MetadataOverrideURL {
		return true, nil
	}
	if !reflect.DeepEqual(currSpec.UnsupportedOverrides, newSpec.UnsupportedOverrides) {
		return true, nil
	}
	return false, nil
}

func getECUpgradeInput(s store.Store, kcli kbclient.Client, a *apptypes.App, versionLabel string, newSpec *ecv1beta1.ConfigSpec) (*types.PlanStepInputECUpgrade, error) {
	license, err := kotsutil.LoadLicenseFromBytes([]byte(a.License))
	if err != nil {
		return nil, errors.Wrap(err, "parse app license")
	}

	baseArchive, _, err := s.GetAppVersionBaseArchive(a.ID, versionLabel)
	if err != nil {
		return nil, errors.Wrap(err, "get app version base archive")
	}

	currKOTSInstall, err := kotsutil.FindInstallationInPath(baseArchive)
	if err != nil {
		return nil, errors.Wrap(err, "find kots installation in base archive")
	}

	currECInstall, err := embeddedcluster.GetCurrentInstallation(context.Background(), kcli)
	if err != nil {
		return nil, errors.Wrap(err, "get current embedded cluster installation")
	}

	return &types.PlanStepInputECUpgrade{
		CurrentECInstallation:       *currECInstall,
		CurrentKOTSInstallation:     *currKOTSInstall,
		NewECConfigSpec:             *newSpec,
		IsDisasterRecoverySupported: license.Spec.IsDisasterRecoverySupported,
	}, nil
}

func findECConfigSpecInRelease(manifests map[string][]byte) (*ecv1beta1.ConfigSpec, error) {
	for _, contents := range manifests {
		if !kotsutil.IsApiVersionKind(contents, "embeddedcluster.replicated.com/v1beta1", "Config") {
			continue
		}

		var cfg ecv1beta1.Config
		if err := k8syaml.Unmarshal(contents, &cfg); err != nil {
			return nil, errors.Wrap(err, "unmarshal")
		}
		return &cfg.Spec, nil
	}

	return nil, errors.New("not found")
}
