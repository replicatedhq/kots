package updateworker

import (
	"context"

	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"github.com/replicatedhq/ship-cluster/worker/pkg/ship"
	"github.com/replicatedhq/ship-cluster/worker/pkg/types"
)

func (w *Worker) deployUpdate(shipUpdate *types.UpdateSession) error {
	debug := level.Debug(log.With(w.Logger, "method", "updateworker.deployUpdate"))

	debug.Log("event", "set upload url", "id", shipUpdate.ID)
	uploadURL, err := w.Store.GetS3StoreURL(shipUpdate)
	if err != nil {
		level.Error(w.Logger).Log("getUpdateUploadURL", err)
		return err
	}
	shipUpdate.UploadURL = uploadURL

	debug.Log("event", "set output filepath", "watchId", shipUpdate.WatchID, "sequence", shipUpdate.UploadSequence)
	err = w.Store.SetOutputFilepath(context.TODO(), shipUpdate)
	if err != nil {
		level.Error(w.Logger).Log("setUpdateOutputFilepath", err)
		return err
	}

	debug.Log("event", "get namespace", "id", shipUpdate.ID)
	namespace := ship.GetNamespace(context.TODO(), shipUpdate)
	if err := w.ensureNamespace(context.TODO(), namespace); err != nil {
		level.Error(w.Logger).Log("ensureNamespace", err)
		return err
	}

	networkPolicy := ship.GetNetworkPolicySpec(context.TODO(), shipUpdate)
	if err := w.ensureNetworkPolicy(context.TODO(), networkPolicy); err != nil {
		level.Error(w.Logger).Log("networkPolicy", err)
		return err
	}

	secret := ship.GetSecretSpec(context.TODO(), shipUpdate, shipUpdate.StateJSON)
	if err := w.ensureSecret(context.TODO(), secret); err != nil {
		level.Error(w.Logger).Log("ensureSecret", err)
		return err
	}

	serviceAccount := ship.GetServiceAccountSpec(context.TODO(), shipUpdate)
	if err := w.ensureServiceAccount(context.TODO(), serviceAccount); err != nil {
		level.Error(w.Logger).Log("ensureSecret", err)
		return err
	}

	role := ship.GetRoleSpec(context.TODO(), shipUpdate)
	if err := w.ensureRole(context.TODO(), role); err != nil {
		level.Error(w.Logger).Log("ensureRole", err)
		return err
	}

	rolebinding := ship.GetRoleBindingSpec(context.TODO(), shipUpdate)
	if err := w.ensureRoleBinding(context.TODO(), rolebinding); err != nil {
		level.Error(w.Logger).Log("ensureRoleBinding", err)
		return err
	}

	pod := ship.GetPodSpec(context.TODO(), w.Config.LogLevel, w.Config.ShipImage, w.Config.ShipTag, w.Config.ShipPullPolicy, secret.Name, serviceAccount.Name, shipUpdate, w.Config.GithubToken)
	if err := w.ensurePod(context.TODO(), pod); err != nil {
		level.Error(w.Logger).Log("ensurePod", err)
		return err
	}

	// Wait for the pod to be ready here, or clean up and return an error

	service := ship.GetServiceSpec(context.TODO(), shipUpdate)
	if err := w.ensureService(context.TODO(), service); err != nil {
		level.Error(w.Logger).Log("ensureService", err)
		return err
	}
	return nil
}
