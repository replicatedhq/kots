package deploy

import (
	"context"
	"fmt"
	"time"

	"github.com/pkg/errors"
	"github.com/replicatedhq/kots/pkg/apparchive"
	"github.com/replicatedhq/kots/pkg/embeddedcluster"
	"github.com/replicatedhq/kots/pkg/k8sutil"
	kotsadmtypes "github.com/replicatedhq/kots/pkg/kotsadm/types"
	kotsadmconfig "github.com/replicatedhq/kots/pkg/kotsadmconfig"
	"github.com/replicatedhq/kots/pkg/kotsutil"
	"github.com/replicatedhq/kots/pkg/logger"
	registrytypes "github.com/replicatedhq/kots/pkg/registry/types"
	"github.com/replicatedhq/kots/pkg/tasks"
	upgradepreflight "github.com/replicatedhq/kots/pkg/upgradeservice/preflight"
	"github.com/replicatedhq/kots/pkg/upgradeservice/task"
	"github.com/replicatedhq/kots/pkg/upgradeservice/types"
	"github.com/replicatedhq/kots/pkg/util"
	corev1 "k8s.io/api/core/v1"
	kuberneteserrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type CanDeployAppOptions struct {
	Params           types.UpgradeServiceParams
	KotsKinds        *kotsutil.KotsKinds
	RegistrySettings registrytypes.RegistrySettings
}

func CanDeployApp(opts CanDeployAppOptions) (bool, string, error) {
	needsConfig, err := kotsadmconfig.NeedsConfiguration(
		opts.Params.AppSlug,
		opts.Params.NextSequence,
		opts.Params.AppIsAirgap,
		opts.KotsKinds,
		opts.RegistrySettings,
	)
	if err != nil {
		return false, "", errors.Wrap(err, "failed to check if version needs configuration")
	}
	if needsConfig {
		return false, "cannot deploy because version needs configuration", nil
	}

	pd, err := upgradepreflight.GetPreflightData()
	if err != nil {
		return false, "", errors.Wrap(err, "failed to get preflight data")
	}
	if pd.Result != nil && pd.Result.HasFailingStrictPreflights {
		return false, "cannot deploy because a strict preflight check has failed", nil
	}

	return true, "", nil
}

type DeployAppOptions struct {
	Ctx                          context.Context
	IsSkipPreflights             bool
	ContinueWithFailedPreflights bool
	Params                       types.UpgradeServiceParams
	KotsKinds                    *kotsutil.KotsKinds
	RegistrySettings             registrytypes.RegistrySettings
}

func DeployApp(opts DeployAppOptions) error {
	tgzArchiveKey := fmt.Sprintf(
		"deployments/%s/%s-%s.tar.gz",
		opts.Params.AppSlug,
		opts.Params.UpdateChannelID,
		opts.Params.UpdateCursor,
	)
	if err := apparchive.CreateAppVersionArchive(opts.Params.AppArchive, tgzArchiveKey); err != nil {
		return errors.Wrap(err, "failed to create app version archive")
	}

	kbClient, err := k8sutil.GetKubeClient(opts.Ctx)
	if err != nil {
		return fmt.Errorf("failed to get kubeclient: %w", err)
	}

	rcu, err := embeddedcluster.RequiresClusterUpgrade(opts.Ctx, kbClient, opts.KotsKinds)
	if err != nil {
		return errors.Wrap(err, "failed to check if cluster requires upgrade")
	}
	if !rcu {
		// a cluster upgrade is not required so we can proceed with deploying the app
		if err := createDeployment(createDeploymentOptions{
			ctx:                          opts.Ctx,
			isSkipPreflights:             opts.IsSkipPreflights,
			continueWithFailedPreflights: opts.ContinueWithFailedPreflights,
			params:                       opts.Params,
			tgzArchiveKey:                tgzArchiveKey,
			requiresClusterUpgrade:       false,
		}); err != nil {
			return errors.Wrap(err, "failed to create deployment")
		}
		// wait for deployment to be processed by the kots operator
		if err := waitForDeployment(opts.Ctx, opts.Params.AppSlug); err != nil {
			return errors.Wrap(err, "failed to wait for deployment")
		}
		return nil
	}

	// TODO NOW: no error is shown if pod restarts during cluster upgrade
	if err := task.SetStatusUpgradingCluster(opts.Params.AppSlug, "Upgrading cluster..."); err != nil {
		return errors.Wrap(err, "failed to set task status")
	}

	go func() (finalError error) {
		finishedChan := make(chan error)
		defer close(finishedChan)

		tasks.StartTaskMonitor(task.GetID(opts.Params.AppSlug), finishedChan)
		defer func() {
			if finalError != nil {
				logger.Error(finalError)
			}
			finishedChan <- finalError
		}()

		if err := embeddedcluster.StartClusterUpgrade(context.Background(), opts.KotsKinds, opts.RegistrySettings); err != nil {
			return errors.Wrap(err, "failed to start cluster upgrade")
		}

		if err := createDeployment(createDeploymentOptions{
			ctx:                          context.Background(),
			isSkipPreflights:             opts.IsSkipPreflights,
			continueWithFailedPreflights: opts.ContinueWithFailedPreflights,
			params:                       opts.Params,
			tgzArchiveKey:                tgzArchiveKey,
			requiresClusterUpgrade:       true,
		}); err != nil {
			return errors.Wrap(err, "failed to create deployment")
		}

		return nil
	}()

	return nil
}

type createDeploymentOptions struct {
	ctx                          context.Context
	isSkipPreflights             bool
	continueWithFailedPreflights bool
	params                       types.UpgradeServiceParams
	tgzArchiveKey                string
	requiresClusterUpgrade       bool
}

// createDeployment creates a configmap with the app version info which gets detected by the operator of the new kots version to deploy the app.
func createDeployment(opts createDeploymentOptions) error {
	clientset, err := k8sutil.GetClientset()
	if err != nil {
		return errors.Wrap(err, "failed to get clientset")
	}

	preflightData, err := upgradepreflight.GetPreflightData()
	if err != nil {
		return errors.Wrap(err, "failed to get preflight data")
	}

	preflightResult := ""
	if preflightData.Result != nil {
		preflightResult = preflightData.Result.Result
	}

	cm := &corev1.ConfigMap{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "ConfigMap",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: getDeploymentName(opts.params.AppSlug),
			Labels: map[string]string{
				// exclude from backup so this app version is not deployed on restore
				kotsadmtypes.ExcludeKey: kotsadmtypes.ExcludeValue,
				"kots.io/deployment":    "true",
				"kots.io/processed":     "false",
			},
		},
		Data: map[string]string{
			"app-id":                          opts.params.AppID,
			"app-slug":                        opts.params.AppSlug,
			"app-version-archive":             opts.tgzArchiveKey,
			"base-sequence":                   fmt.Sprintf("%d", opts.params.BaseSequence),
			"version-label":                   opts.params.UpdateVersionLabel,
			"source":                          opts.params.Source,
			"is-airgap":                       fmt.Sprintf("%t", opts.params.AppIsAirgap),
			"channel-id":                      opts.params.UpdateChannelID,
			"update-cursor":                   opts.params.UpdateCursor,
			"skip-preflights":                 fmt.Sprintf("%t", opts.isSkipPreflights),
			"continue-with-failed-preflights": fmt.Sprintf("%t", opts.continueWithFailedPreflights),
			"preflight-result":                preflightResult,
			"kots-version":                    opts.params.UpdateKOTSVersion,
			"requires-cluster-upgrade":        fmt.Sprintf("%t", opts.requiresClusterUpgrade),
		},
	}

	err = clientset.CoreV1().ConfigMaps(util.PodNamespace).Delete(context.TODO(), cm.Name, metav1.DeleteOptions{})
	if err != nil && !kuberneteserrors.IsNotFound(err) {
		return errors.Wrap(err, "failed to delete configmap")
	}

	_, err = clientset.CoreV1().ConfigMaps(util.PodNamespace).Create(context.TODO(), cm, metav1.CreateOptions{})
	if err != nil {
		return errors.Wrap(err, "failed to create configmap")
	}

	return nil
}

func waitForDeployment(ctx context.Context, appSlug string) error {
	clientset, err := k8sutil.GetClientset()
	if err != nil {
		return errors.Wrap(err, "failed to get clientset")
	}
	for {
		s, err := clientset.CoreV1().ConfigMaps(util.PodNamespace).Get(ctx, getDeploymentName(appSlug), metav1.GetOptions{})
		if err != nil {
			return errors.Wrap(err, "failed to get configmap")
		}
		if s.Labels["kots.io/processed"] == "true" {
			return nil
		}
		time.Sleep(1 * time.Second)
	}
}

func getDeploymentName(appSlug string) string {
	return fmt.Sprintf("kotsadm-%s-deployment", appSlug)
}

// IsClusterUpgrading returns true if:
// - the upgrade service task status is upgrading cluster OR
// - the deployment requires a cluster upgrade and has not been processed yet
func IsClusterUpgrading(ctx context.Context, appSlug string) (bool, error) {
	isUpgrading, err := task.IsStatusUpgradingCluster(appSlug)
	if err != nil {
		return false, errors.Wrap(err, "failed to get task status")
	}
	if isUpgrading {
		return true, nil
	}
	clientset, err := k8sutil.GetClientset()
	if err != nil {
		return false, errors.Wrap(err, "failed to get clientset")
	}
	cm, err := clientset.CoreV1().ConfigMaps(util.PodNamespace).Get(ctx, getDeploymentName(appSlug), metav1.GetOptions{})
	if err != nil {
		if kuberneteserrors.IsNotFound(err) {
			return false, nil
		}
		return false, errors.Wrap(err, "failed to get configmap")
	}
	if cm.Labels["kots.io/processed"] == "true" {
		return false, nil
	}
	if cm.Data["requires-cluster-upgrade"] == "true" {
		return true, nil
	}
	return false, nil
}
