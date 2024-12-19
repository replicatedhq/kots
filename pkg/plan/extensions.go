package plan

import (
	"context"
	"reflect"

	"github.com/pkg/errors"
	ecv1beta1 "github.com/replicatedhq/embedded-cluster/kinds/apis/v1beta1"
	"github.com/replicatedhq/kots/pkg/embeddedcluster"
	"github.com/replicatedhq/kots/pkg/plan/types"
	"github.com/replicatedhq/kots/pkg/websocket"
	kbclient "sigs.k8s.io/controller-runtime/pkg/client"
)

type ExtensionsDiffResult struct {
	Added    []ecv1beta1.Chart
	Removed  []ecv1beta1.Chart
	Modified []ecv1beta1.Chart
}

func executeECExtensionAdd(p *types.Plan, step *types.PlanStep) error {
	in, ok := step.Input.(types.PlanStepInputECExtension)
	if !ok {
		return errors.New("invalid input for embedded cluster extension add step")
	}
	if err := websocket.AddExtension(in.Repos, in.Chart, p.AppSlug, p.VersionLabel, step.ID); err != nil {
		return errors.Wrap(err, "add extension")
	}
	return nil
}

func executeECExtensionUpgrade(p *types.Plan, step *types.PlanStep) error {
	in, ok := step.Input.(types.PlanStepInputECExtension)
	if !ok {
		return errors.New("invalid input for embedded cluster extension upgrade step")
	}
	if err := websocket.UpgradeExtension(in.Repos, in.Chart, p.AppSlug, p.VersionLabel, step.ID); err != nil {
		return errors.Wrap(err, "upgrade extension")
	}
	return nil
}

func executeECExtensionRemove(p *types.Plan, step *types.PlanStep) error {
	in, ok := step.Input.(types.PlanStepInputECExtension)
	if !ok {
		return errors.New("invalid input for embedded cluster extension remove step")
	}
	if err := websocket.RemoveExtension(in.Repos, in.Chart, p.AppSlug, p.VersionLabel, step.ID); err != nil {
		return errors.Wrap(err, "remove extension")
	}
	return nil
}

func getExtensions(kcli kbclient.Client, newSpec *ecv1beta1.ConfigSpec) (ecv1beta1.Extensions, ecv1beta1.Extensions, error) {
	currInstall, err := embeddedcluster.GetCurrentInstallation(context.Background(), kcli)
	if err != nil {
		return ecv1beta1.Extensions{}, ecv1beta1.Extensions{}, errors.Wrap(err, "get current embedded cluster installation")
	}
	return currInstall.Spec.Config.Extensions, newSpec.Extensions, nil
}

func diffExtensions(oldExts, newExts ecv1beta1.Extensions) ExtensionsDiffResult {
	oldCharts := make(map[string]ecv1beta1.Chart)
	newCharts := make(map[string]ecv1beta1.Chart)

	if oldExts.Helm != nil {
		for _, chart := range oldExts.Helm.Charts {
			oldCharts[chart.Name] = chart
		}
	}
	if newExts.Helm != nil {
		for _, chart := range newExts.Helm.Charts {
			newCharts[chart.Name] = chart
		}
	}

	var added, removed, modified []ecv1beta1.Chart

	// find removed and modified charts.
	for name, oldChart := range oldCharts {
		newChart, exists := newCharts[name]
		if !exists {
			// chart was removed.
			removed = append(removed, oldChart)
		} else if !reflect.DeepEqual(oldChart, newChart) {
			// chart was modified.
			modified = append(modified, newChart)
		}
	}

	// find added charts.
	for name, newChart := range newCharts {
		if _, exists := oldCharts[name]; !exists {
			// chart was added.
			added = append(added, newChart)
		}
	}

	return ExtensionsDiffResult{
		Added:    added,
		Removed:  removed,
		Modified: modified,
	}
}
