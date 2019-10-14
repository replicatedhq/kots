package updateworker

import (
	"context"

	"github.com/replicatedhq/kotsadm/worker/pkg/ship"
	"github.com/replicatedhq/kotsadm/worker/pkg/types"
	"go.uber.org/zap"
)

func (w *Worker) deployUpdate(shipUpdate *types.UpdateSession) error {
	shipState, err := ship.NewStateManager(w.Config)
	if err != nil {
		w.Logger.Errorw("updateworker failed to initialize state manager", zap.Error(err))
		return err
	}
	s3State, err := shipState.CreateS3State(shipUpdate.StateJSON)
	if err != nil {
		w.Logger.Errorw("updateworker failed to upload state to S3", zap.Error(err))
		return err
	}

	uploadURL, err := w.Store.GetS3StoreURL(shipUpdate)
	if err != nil {
		w.Logger.Errorw("updateworker failed to get s3 store url", zap.Error(err))
		return err
	}
	shipUpdate.UploadURL = uploadURL

	err = w.Store.SetOutputFilepath(context.TODO(), shipUpdate)
	if err != nil {
		w.Logger.Errorw("updateworker failed to set output file path", zap.Error(err))
		return err
	}

	namespace := ship.GetNamespace(context.TODO(), shipUpdate)
	if err := w.ensureNamespace(context.TODO(), namespace); err != nil {
		w.Logger.Errorw("updateworker failed to create namespace", zap.Error(err))
		return err
	}

	networkPolicy := ship.GetNetworkPolicySpec(context.TODO(), shipUpdate)
	if err := w.ensureNetworkPolicy(context.TODO(), networkPolicy); err != nil {
		w.Logger.Errorw("updateworker failed to create networkpolicy", zap.Error(err))
		return err
	}

	serviceAccount := ship.GetServiceAccountSpec(context.TODO(), shipUpdate)
	if err := w.ensureServiceAccount(context.TODO(), serviceAccount); err != nil {
		w.Logger.Errorw("updateworker failed to create serviceaccount", zap.Error(err))
		return err
	}

	role := ship.GetRoleSpec(context.TODO(), shipUpdate)
	if err := w.ensureRole(context.TODO(), role); err != nil {
		w.Logger.Errorw("updateworker failed to create role", zap.Error(err))
		return err
	}

	rolebinding := ship.GetRoleBindingSpec(context.TODO(), shipUpdate)
	if err := w.ensureRoleBinding(context.TODO(), rolebinding); err != nil {
		w.Logger.Errorw("updateworker failed to create rolebinding", zap.Error(err))
		return err
	}

	pod := ship.GetPodSpec(context.TODO(), w.Config.LogLevel, w.Config.ShipImage, w.Config.ShipTag, w.Config.ShipPullPolicy, s3State, serviceAccount.Name, shipUpdate, w.Config.GithubToken)
	if err := w.ensurePod(context.TODO(), pod); err != nil {
		w.Logger.Errorw("updateworker failed to create pod", zap.Error(err))
		return err
	}

	// Wait for the pod to be ready here, or clean up and return an error

	service := ship.GetServiceSpec(context.TODO(), shipUpdate)
	if err := w.ensureService(context.TODO(), service); err != nil {
		w.Logger.Errorw("updateworker failed to create service", zap.Error(err))
		return err
	}
	return nil
}
