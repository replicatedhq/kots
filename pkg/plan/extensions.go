package plan

import (
	"context"
	"reflect"

	"github.com/pkg/errors"
	ecv1beta1 "github.com/replicatedhq/embedded-cluster/kinds/apis/v1beta1"
	"github.com/replicatedhq/kots/pkg/embeddedcluster"
	"github.com/replicatedhq/kots/pkg/plan/types"
	"github.com/replicatedhq/kots/pkg/store"
	"github.com/replicatedhq/kots/pkg/websocket"
	"github.com/segmentio/ksuid"
	kbclient "sigs.k8s.io/controller-runtime/pkg/client"
)

type ExtensionsDiffResult struct {
	Added    []ecv1beta1.Chart
	Removed  []ecv1beta1.Chart
	Modified []ecv1beta1.Chart
}

func planECExtensions(kcli kbclient.Client, newSpec *ecv1beta1.ConfigSpec) ([]*types.PlanStep, error) {
	steps := []*types.PlanStep{}

	currECExts, newECExts, err := getECExtensions(kcli, newSpec)
	if err != nil {
		return nil, errors.Wrap(err, "get extensions")
	}

	ecExtsDiff := diffECExtensions(currECExts, newECExts)
	newRepos := newECExts.Helm.Repositories

	// added extensions
	for _, chart := range ecExtsDiff.Added {
		steps = append(steps, &types.PlanStep{
			ID:                ksuid.New().String(),
			Name:              "Extension Add",
			Type:              types.StepTypeECExtensionAdd,
			Status:            types.StepStatusPending,
			StatusDescription: "Pending extension addition",
			Input: types.PlanStepInputECExtension{
				Repos: newRepos,
				Chart: chart,
			},
			Owner: types.StepOwnerECManager,
		})
	}

	// modified extensions
	for _, chart := range ecExtsDiff.Modified {
		steps = append(steps, &types.PlanStep{
			ID:                ksuid.New().String(),
			Name:              "Extension Upgrade",
			Type:              types.StepTypeECExtensionUpgrade,
			Status:            types.StepStatusPending,
			StatusDescription: "Pending extension upgrade",
			Input: types.PlanStepInputECExtension{
				Repos: newRepos,
				Chart: chart,
			},
			Owner: types.StepOwnerECManager,
		})
	}

	// removed extensions
	for _, chart := range ecExtsDiff.Removed {
		steps = append(steps, &types.PlanStep{
			ID:                ksuid.New().String(),
			Name:              "Extension Remove",
			Type:              types.StepTypeECExtensionRemove,
			Status:            types.StepStatusPending,
			StatusDescription: "Pending extension removal",
			Input: types.PlanStepInputECExtension{
				Repos: newRepos,
				Chart: chart,
			},
			Owner: types.StepOwnerECManager,
		})
	}

	return steps, nil
}

func executeECExtensionAdd(s store.Store, ws *websocket.ConnectionManager, p *types.Plan, step *types.PlanStep) error {
	in, ok := step.Input.(types.PlanStepInputECExtension)
	if !ok {
		return errors.New("invalid input for embedded cluster extension add step")
	}

	if step.Status == types.StepStatusPending {
		if err := websocket.AddExtension(ws, in.Repos, in.Chart, p.AppSlug, p.VersionLabel, step.ID); err != nil {
			return errors.Wrap(err, "add extension")
		}
	}

	if err := waitForStep(s, p, step.ID); err != nil {
		return errors.Wrap(err, "wait for embedded cluster extension add")
	}

	return nil
}

func executeECExtensionUpgrade(s store.Store, ws *websocket.ConnectionManager, p *types.Plan, step *types.PlanStep) error {
	in, ok := step.Input.(types.PlanStepInputECExtension)
	if !ok {
		return errors.New("invalid input for embedded cluster extension upgrade step")
	}

	if step.Status == types.StepStatusPending {
		if err := websocket.UpgradeExtension(ws, in.Repos, in.Chart, p.AppSlug, p.VersionLabel, step.ID); err != nil {
			return errors.Wrap(err, "upgrade extension")
		}
	}

	if err := waitForStep(s, p, step.ID); err != nil {
		return errors.Wrap(err, "wait for embedded cluster extension upgrade")
	}

	return nil
}

func executeECExtensionRemove(s store.Store, ws *websocket.ConnectionManager, p *types.Plan, step *types.PlanStep) error {
	in, ok := step.Input.(types.PlanStepInputECExtension)
	if !ok {
		return errors.New("invalid input for embedded cluster extension remove step")
	}

	if step.Status == types.StepStatusPending {
		if err := websocket.RemoveExtension(ws, in.Repos, in.Chart, p.AppSlug, p.VersionLabel, step.ID); err != nil {
			return errors.Wrap(err, "remove extension")
		}
	}

	if err := waitForStep(s, p, step.ID); err != nil {
		return errors.Wrap(err, "wait for embedded cluster extension remove")
	}

	return nil
}

func getECExtensions(kcli kbclient.Client, newSpec *ecv1beta1.ConfigSpec) (ecv1beta1.Extensions, ecv1beta1.Extensions, error) {
	currInstall, err := embeddedcluster.GetCurrentInstallation(context.Background(), kcli)
	if err != nil {
		return ecv1beta1.Extensions{}, ecv1beta1.Extensions{}, errors.Wrap(err, "get current embedded cluster installation")
	}
	return currInstall.Spec.Config.Extensions, newSpec.Extensions, nil
}

func diffECExtensions(oldExts, newExts ecv1beta1.Extensions) ExtensionsDiffResult {
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
