package analyzeworker

import (
	"context"
	"os"

	"github.com/replicatedhq/ship-cluster/worker/pkg/troubleshoot"
	"github.com/replicatedhq/ship-cluster/worker/pkg/types"
	"go.uber.org/zap"
)

func (w *Worker) deployAnalyzer(supportBundle *types.SupportBundle) error {
	namespace := troubleshoot.GetNamespace(context.TODO(), supportBundle)
	if err := w.ensureNamespace(context.TODO(), namespace); err != nil {
		w.Logger.Errorw("analyzeworker failed to create namespace", zap.Error(err))
		return err
	}

	networkPolicy := troubleshoot.GetNetworkPolicySpec(context.TODO(), supportBundle)
	if err := w.ensureNetworkPolicy(context.TODO(), networkPolicy); err != nil {
		w.Logger.Errorw("analyzeworker failed to create networkpolicy", zap.Error(err))
		return err
	}

	serviceAccount := troubleshoot.GetServiceAccountSpec(context.TODO(), supportBundle)
	if err := w.ensureServiceAccount(context.TODO(), serviceAccount); err != nil {
		w.Logger.Errorw("analyzeworker failed to create serviceaccount", zap.Error(err))
		return err
	}

	role := troubleshoot.GetRoleSpec(context.TODO(), supportBundle)
	if err := w.ensureRole(context.TODO(), role); err != nil {
		w.Logger.Errorw("analyzeworker failed to create role", zap.Error(err))
		return err
	}

	rolebinding := troubleshoot.GetRoleBindingSpec(context.TODO(), supportBundle)
	if err := w.ensureRoleBinding(context.TODO(), rolebinding); err != nil {
		w.Logger.Errorw("analyzeworker failed to create rolebinding", zap.Error(err))
		return err
	}

	analyzeSpec, err := w.Store.GetAnalyzeSpec(context.TODO(), supportBundle.WatchID)
	if err != nil {
		return err
	}

	configMap := troubleshoot.GetConfigMapSpec(context.TODO(), supportBundle, analyzeSpec)
	if err := w.ensureConfigMap(context.TODO(), configMap); err != nil {
		w.Logger.Errorw("analyzeworker failed to create configMap", zap.Error(err))
		return err
	}

	getBundlePresignedURI, err := w.Store.GetSupportBundleURL(supportBundle)
	if err != nil {
		return err
	}

	pod := troubleshoot.GetPodSpec(context.TODO(), w.Config.LogLevel, w.Config.AnalyzeImage, w.Config.AnalyzeTag, w.Config.AnalyzePullPolicy, serviceAccount.Name, supportBundle, getBundlePresignedURI, os.Getenv("ANALYZE_NODE_SELECTOR"))
	if err := w.ensurePod(context.TODO(), pod); err != nil {
		w.Logger.Errorw("analyzeworker failed to create pod", zap.Error(err))
		return err
	}

	return nil
}
