package plan

import (
	"context"
	"fmt"
	"time"

	"github.com/pkg/errors"
	apptypes "github.com/replicatedhq/kots/pkg/app/types"
	"github.com/replicatedhq/kots/pkg/kotsutil"
	"github.com/replicatedhq/kots/pkg/logger"
	"github.com/replicatedhq/kots/pkg/plan/types"
	"github.com/replicatedhq/kots/pkg/store"
	"github.com/replicatedhq/kots/pkg/websocket"
	"github.com/segmentio/ksuid"
	corev1 "k8s.io/api/core/v1"
	kbclient "sigs.k8s.io/controller-runtime/pkg/client"
)

func planECManagersUpgrade(ws *websocket.ConnectionManager, kcli kbclient.Client, a *apptypes.App, newECVersion string) ([]*types.PlanStep, error) {
	nodes := &corev1.NodeList{}
	if err := kcli.List(context.Background(), nodes, &kbclient.ListOptions{}); err != nil {
		return nil, errors.Wrap(err, "list nodes")
	}

	steps := []*types.PlanStep{}
	connectedManagers := ws.GetClients()

	for _, node := range nodes.Items {
		m, ok := connectedManagers[node.Name]
		if !ok {
			return nil, errors.Errorf("manager of node %s is not connected", node.Name)
		}
		if m.Version == newECVersion {
			continue
		}
		in, err := getECManagerUpgradeInput(node.Name, a)
		if err != nil {
			return nil, errors.Wrap(err, "get ec manager upgrade input")
		}
		steps = append(steps, &types.PlanStep{
			ID:                ksuid.New().String(),
			Name:              fmt.Sprintf("%s EC Manager Upgrade", node.Name),
			Type:              types.StepTypeECManagerUpgrade,
			Status:            types.StepStatusPending,
			StatusDescription: "Pending EC Manager Upgrade",
			Input:             *in,
			Owner:             types.StepOwnerECManager,
		})
	}

	return steps, nil
}

func executeECManagerUpgrade(s store.Store, ws *websocket.ConnectionManager, p *types.Plan, step *types.PlanStep) error {
	in, ok := step.Input.(types.PlanStepInputECManagerUpgrade)
	if !ok {
		return errors.New("invalid input for ec manager upgrade step")
	}

	if step.Status == types.StepStatusPending {
		if err := websocket.UpgradeECManager(ws, in.NodeName, in.LicenseID, in.LicenseEndpoint, p.NewECVersion, p.AppSlug, p.VersionLabel, step.ID); err != nil {
			return errors.Wrapf(err, "upgrade %s ec manager", in.NodeName)
		}
	}

	if err := waitForECManagerToConnect(ws, in.NodeName, p.NewECVersion); err != nil {
		return errors.Wrapf(err, "wait for %s ec manager to connect", in.NodeName)
	}

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

func getECManagerUpgradeInput(nodeName string, a *apptypes.App) (*types.PlanStepInputECManagerUpgrade, error) {
	license, err := kotsutil.LoadLicenseFromBytes([]byte(a.License))
	if err != nil {
		return nil, errors.Wrap(err, "parse app license")
	}

	return &types.PlanStepInputECManagerUpgrade{
		NodeName:        nodeName,
		LicenseID:       license.Spec.LicenseID,
		LicenseEndpoint: license.Spec.Endpoint,
	}, nil
}

func waitForECManagerToConnect(ws *websocket.ConnectionManager, nodeName string, version string) error {
	timeout := time.After(5 * time.Minute)
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-timeout:
			return errors.Errorf("timeout waiting for EC manager on node %s to connect with version %s", nodeName, version)

		case <-ticker.C:
			connectedManagers := ws.GetClients()
			if m, ok := connectedManagers[nodeName]; ok {
				if m.Version == version {
					logger.Debugf("EC manager on node %s connected successfully", nodeName)
					return nil
				}
				logger.Debugf("EC manager on node %s is connected but is running version %s not %s. Waiting...", nodeName, m.Version, version)
			} else {
				logger.Debugf("EC manager on node %s is not connected. Waiting...", nodeName)
			}
		}
	}
}
