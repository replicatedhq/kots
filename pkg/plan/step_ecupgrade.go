package plan

import (
	"time"

	"github.com/pkg/errors"
	ecv1beta1 "github.com/replicatedhq/embedded-cluster/kinds/apis/v1beta1"
	apptypes "github.com/replicatedhq/kots/pkg/app/types"
	"github.com/replicatedhq/kots/pkg/embeddedcluster"
	"github.com/replicatedhq/kots/pkg/plan/types"
	"github.com/replicatedhq/kots/pkg/store"
	"github.com/replicatedhq/kots/pkg/websocket"
	"github.com/segmentio/ksuid"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	kbclient "sigs.k8s.io/controller-runtime/pkg/client"
)

var (
	_ IPlanStep = (*PlanStepECUpgrade)(nil)
)

type PlanStepECUpgrade struct {
	types.PlanStep `json:",inline" yaml:",inline"`
	Input          types.PlanStepInputECUpgrade `json:"input" yaml:"input"`
}

func NewPlanStepECUpgrade(s store.Store, kcli kbclient.Client, app *apptypes.App, newECConfigSpec *ecv1beta1.ConfigSpec, opts PlanUpgradeOptions) (*PlanStepECUpgrade, error) {
	in, err := getECUpgradeInput(s, kcli, app, opts.VersionLabel, newECConfigSpec)
	if err != nil {
		return nil, errors.Wrap(err, "get ec upgrade input")
	}
	return &PlanStepECUpgrade{
		PlanStep: types.PlanStep{
			ID:                ksuid.New().String(),
			Name:              "Embedded Cluster Upgrade",
			Type:              types.StepTypeECUpgrade,
			Status:            types.StepStatusPending,
			StatusDescription: "Pending embedded cluster upgrade",
			Input:             *in,
			Owner:             types.StepOwnerECManager,
		},
		Input: *in,
	}, nil
}

func (ps *PlanStepECUpgrade) Execute(s store.Store, p *types.Plan) error {
	if ps.Status == types.StepStatusPending {
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
			Spec: ps.Input.CurrentECInstallation.Spec,
		}
		newInstall.Spec.Artifacts = embeddedcluster.GetArtifactsFromInstallation(ps.Input.CurrentKOTSInstallation)
		newInstall.Spec.Config = &ps.Input.NewECConfigSpec
		newInstall.Spec.LicenseInfo = &ecv1beta1.LicenseInfo{IsDisasterRecoverySupported: ps.Input.IsDisasterRecoverySupported}

		if err := websocket.UpgradeCluster(newInstall, p.AppSlug, p.VersionLabel, ps.ID); err != nil {
			return errors.Wrap(err, "upgrade cluster")
		}

		return nil
	}
	if err := waitForStep(s, p, ps.ID); err != nil {
		return errors.Wrap(err, "wait for embedded cluster upgrade")
	}
	return nil
}
