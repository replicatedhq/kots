package deploy

import (
	"context"
	"fmt"
	"time"

	"github.com/pkg/errors"
	embeddedclusterv1beta1 "github.com/replicatedhq/embedded-cluster/kinds/apis/v1beta1"
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
	kbclient "sigs.k8s.io/controller-runtime/pkg/client"
)

type CanDeployOptions struct {
	Params           types.UpgradeServiceParams
	KotsKinds        *kotsutil.KotsKinds
	RegistrySettings registrytypes.RegistrySettings
}

func CanDeploy(opts CanDeployOptions) (bool, string, error) {
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

type DeployOptions struct {
	Ctx                          context.Context
	IsSkipPreflights             bool
	ContinueWithFailedPreflights bool
	Params                       types.UpgradeServiceParams
	KotsKinds                    *kotsutil.KotsKinds
	RegistrySettings             registrytypes.RegistrySettings
}

func Deploy(opts DeployOptions) error {
	// put the app version archive in the object store so the operator
	// of the new kots version can retrieve it to deploy the app
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

	// a cluster upgrade is required. that's a long running process, and there's a high chance
	// kots will be upgraded and restart during the process. so we run the upgrade in a goroutine
	// and report the status back to the ui for the user to see the progress.
	// the kots operator takes care of reporting the progress after the deployment gets created

	if err := task.SetStatusUpgradingCluster(opts.Params.AppSlug, embeddedclusterv1beta1.InstallationStateEnqueued); err != nil {
		return errors.Wrap(err, "failed to set task status")
	}

	go func() (finalError error) {
		defer func() {
			if finalError != nil {
				if err := notifyUpgradeFailed(context.Background(), kbClient, opts, finalError.Error()); err != nil {
					logger.Errorf("Failed to notify upgrade failed: %v", err)
				}
				if err := task.SetStatusUpgradeFailed(opts.Params.AppSlug, finalError.Error()); err != nil {
					logger.Error(errors.Wrap(err, "failed to set task status to upgrade failed"))
				}
			}
		}()

		finishedCh := make(chan struct{})
		defer close(finishedCh)
		tasks.StartTicker(task.GetID(opts.Params.AppSlug), finishedCh)

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

// notifyUpgradeFailed sends a metrics event to the api that the upgrade failed.
func notifyUpgradeFailed(ctx context.Context, kbClient kbclient.Client, opts DeployOptions, reason string) error {
	ins, err := embeddedcluster.GetCurrentInstallation(ctx, kbClient)
	if err != nil {
		return fmt.Errorf("failed to get current installation: %w", err)
	}

	if ins.Spec.AirGap {
		return nil
	}

	prev, err := embeddedcluster.GetPreviousInstallation(ctx, kbClient)
	if err != nil {
		return errors.Wrap(err, "failed to get previous installation")
	} else if prev == nil {
		return errors.New("previous installation not found")
	}

	err = embeddedcluster.NotifyUpgradeFailed(ctx, opts.KotsKinds.License.Spec.Endpoint, ins, prev, reason)
	if err != nil {
		return errors.Wrap(err, "failed to send event")
	}
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
			"embedded-cluster-version":        opts.params.UpdateECVersion,
			"requires-cluster-upgrade":        fmt.Sprintf("%t", opts.requiresClusterUpgrade),
		},
	}

	err = clientset.CoreV1().ConfigMaps(util.PodNamespace).Delete(opts.ctx, cm.Name, metav1.DeleteOptions{})
	if err != nil && !kuberneteserrors.IsNotFound(err) {
		return errors.Wrap(err, "failed to delete configmap")
	}

	_, err = clientset.CoreV1().ConfigMaps(util.PodNamespace).Create(opts.ctx, cm, metav1.CreateOptions{})
	if err != nil {
		return errors.Wrap(err, "failed to create configmap")
	}

	return nil
}

// waitForDeployment waits for the deployment to be processed by the kots operator.
// this is only used when a cluster upgrade is not required.
func waitForDeployment(ctx context.Context, appSlug string) error {
	clientset, err := k8sutil.GetClientset()
	if err != nil {
		return errors.Wrap(err, "failed to get clientset")
	}
	start := time.Now()
	for {
		cm, err := clientset.CoreV1().ConfigMaps(util.PodNamespace).Get(ctx, getDeploymentName(appSlug), metav1.GetOptions{})
		if err != nil {
			return errors.Wrap(err, "failed to get configmap")
		}
		if cm.Labels != nil && cm.Labels["kots.io/processed"] == "true" {
			return nil
		}
		if time.Sleep(1 * time.Second); time.Since(start) > 15*time.Second {
			return errors.New("timed out waiting for deployment to be processed")
		}
	}
}

func getDeploymentName(appSlug string) string {
	return fmt.Sprintf("kotsadm-%s-deployment", appSlug)
}
