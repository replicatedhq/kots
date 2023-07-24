package helm

import (
	"context"
	"io/ioutil"
	"strconv"
	"sync"
	"time"

	"github.com/pkg/errors"
	apptypes "github.com/replicatedhq/kots/pkg/app/types"
	"github.com/replicatedhq/kots/pkg/k8sutil"
	"github.com/replicatedhq/kots/pkg/kotsadmconfig"
	"github.com/replicatedhq/kots/pkg/logger"
	registrytypes "github.com/replicatedhq/kots/pkg/registry/types"
	storetypes "github.com/replicatedhq/kots/pkg/store/types"
	"github.com/replicatedhq/kots/pkg/util"
	kotsv1beta1 "github.com/replicatedhq/kotskinds/apis/kots/v1beta1"
	helmrelease "helm.sh/helm/v3/pkg/release"
	corev1 "k8s.io/api/core/v1"
	kuberneteserrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/watch"
)

// TODO: Support same releases names in different namespaces
var (
	helmAppCache  = map[string]*apptypes.HelmApp{}
	tmpValuesRoot string
	appCacheLock  sync.Mutex
)

func Init(ctx context.Context) error {
	tmpDir, err := ioutil.TempDir("", "helm-values-")
	if err != nil {
		return errors.Wrap(err, "failed to create temp dir")
	}
	tmpValuesRoot = tmpDir

	clientSet, err := k8sutil.GetClientset()
	if err != nil {
		return errors.Wrap(err, "failed to get clientset")
	}

	namespacesToWatch := []string{}
	if k8sutil.IsKotsadmClusterScoped(ctx, clientSet, util.PodNamespace) {
		namespaces, err := clientSet.CoreV1().Namespaces().List(ctx, metav1.ListOptions{})
		if err != nil {
			return errors.Wrap(err, "failed to get namespaces")
		}
		for _, ns := range namespaces.Items {
			namespacesToWatch = append(namespacesToWatch, ns.Name)
		}
	} else {
		namespacesToWatch = []string{util.PodNamespace}
	}

	for _, namespace := range namespacesToWatch {
		secretsSelector := labels.SelectorFromSet(map[string]string{"owner": "helm"}).String()
		secrets, err := clientSet.CoreV1().Secrets(namespace).List(context.TODO(), metav1.ListOptions{
			LabelSelector: secretsSelector,
		})
		if err != nil {
			if !kuberneteserrors.IsForbidden(err) && !kuberneteserrors.IsNotFound(err) {
				logger.Warnf("failed to list secrets for namespace: %s", namespace)
			}
			continue
		}

		initMonitor(clientSet, namespace)
		for _, s := range secrets.Items {
			if s.Labels == nil || s.Labels["status"] != helmrelease.StatusDeployed.String() {
				continue
			}

			releaseInfo, err := helmAppFromSecret(&s)
			if err != nil {
				logger.Errorf("failed to get helm release from secret %s: %v", s.Name, err)
				continue
			}
			if releaseInfo == nil {
				continue
			}
			if releaseInfo.Release.Chart.Values["replicated"] == nil {
				continue
			}

			AddHelmApp(releaseInfo.Release.Name, releaseInfo)
			resumeHelmStatusInformers(releaseInfo.Release.Name)
		}

		go func(namespace string) {
			err := watchSecrets(ctx, namespace, secretsSelector)
			if err != nil {
				logger.Errorf("Faied to watch secrets in ns %s and application cache will not be updated: %v", err)
			}
		}(namespace)
	}

	return nil
}

func GetHelmApp(releaseName string) *apptypes.HelmApp {
	appCacheLock.Lock()
	defer appCacheLock.Unlock()

	return helmAppCache[releaseName]
}

func GetCachedHelmApps() []string {
	appCacheLock.Lock()
	defer appCacheLock.Unlock()

	releases := []string{}
	for k, _ := range helmAppCache {
		releases = append(releases, k)
	}
	return releases
}

func AddHelmApp(releaseName string, helmApp *apptypes.HelmApp) {
	appCacheLock.Lock()
	defer appCacheLock.Unlock()
	helmAppCache[releaseName] = helmApp
}

func RemoveHelmApp(releaseName string) {
	appCacheLock.Lock()
	defer appCacheLock.Unlock()

	delete(helmAppCache, releaseName)
}

func helmAppFromSecret(secret *corev1.Secret) (*apptypes.HelmApp, error) {
	helmRelease, err := HelmReleaseFromSecretData(secret.Data["release"])
	if err != nil {
		return nil, errors.Wrap(err, "failed to get helm release from secret")
	}

	version, err := strconv.ParseInt(secret.Labels["version"], 10, 64)
	if err != nil {
		return nil, errors.Wrap(err, "failed to parse release version")
	}

	helmApp := &apptypes.HelmApp{
		Release:           *helmRelease,
		Labels:            secret.Labels,
		Version:           version,
		CreationTimestamp: secret.CreationTimestamp.Time,
		Namespace:         secret.Namespace,
		TempConfigValues:  map[string]kotsv1beta1.ConfigValue{},
	}

	configSecret, err := GetChartConfigSecret(helmApp)
	if err != nil && !kuberneteserrors.IsNotFound(err) {
		return nil, errors.Wrap(err, "failed to get helm config secret")
	}

	if configSecret == nil {
		return helmApp, nil
	}

	_, helmApp.IsConfigurable = configSecret.Data["config"] // TODO: also check if there are any config items
	helmApp.ChartPath = string(configSecret.Data["chartPath"])

	return helmApp, nil
}

func GetKotsLicenseID(release *helmrelease.Release) string {
	if release == nil {
		return ""
	}

	replValuesInterface := release.Chart.Values["replicated"]
	if replValuesInterface == nil {
		return ""
	}

	replValues, ok := replValuesInterface.(map[string]interface{})
	if !ok {
		return ""
	}

	licenseIDInterface, ok := replValues["license_id"]
	if !ok {
		return ""
	}

	licenseID, ok := licenseIDInterface.(string)
	if !ok {
		return ""
	}

	return licenseID
}

func watchSecrets(ctx context.Context, namespace string, labelSelector string) error {
	clientSet, err := k8sutil.GetClientset()
	if err != nil {
		return errors.Wrap(err, "failed to get clientset")
	}

	opts := metav1.ListOptions{
		LabelSelector: labelSelector,
		Watch:         true,
	}

	secrets := clientSet.CoreV1().Secrets(namespace)
	for {
		w, err := secrets.Watch(ctx, opts)
		if err != nil {
			logger.Warnf("failed to list secrets %s for namespace %s: %v", labelSelector, namespace, err)
			time.Sleep(time.Second * 20)
			continue
		}
		logger.Debugf("watching for changes to secrets in ns %s", namespace)
		for e := range w.ResultChan() {
			switch e.Type {
			case watch.Added, watch.Modified:
				secret, ok := e.Object.(*corev1.Secret)
				if !ok {
					break
				}
				logger.Debugf("got event %s for secret %s in ns %s", e.Type, secret.Name, namespace)
				if secret.Labels == nil || secret.Labels["status"] != helmrelease.StatusDeployed.String() {
					continue
				}
				helmApp, err := helmAppFromSecret(secret)
				if err != nil {
					logger.Errorf("failed to create helm release info from secret %s in namespace %s: %s", secret.Name, namespace)
					break
				}
				if helmApp == nil {
					break
				}
				if helmApp.Release.Chart.Values["replicated"] == nil {
					break
				}

				if err := recalculateCachedUpdates(helmApp); err != nil {
					logger.Errorf("failed to recalculate updates for helm release info from secret %s in namespace %s: %s", secret.Name, namespace, err)
					// Continue here.  Release has been installed and should show up up in Admin Console.
				}

				logger.Debugf("adding secret %s to cache", secret.Name)
				AddHelmApp(helmApp.Release.Name, helmApp)
				resumeHelmStatusInformers(helmApp.Release.Name)
			case watch.Deleted:
				secret, ok := e.Object.(*corev1.Secret)
				if !ok {
					break
				}

				helmRelease, err := HelmReleaseFromSecretData(secret.Data["release"])
				if err != nil {
					logger.Errorf("failed to get helm release from secret in delete event", err)
					break
				}

				// Get app from cache because the config secret is likely gone now, and we can't construct this data from cluster
				helmApp := GetHelmApp(helmRelease.Name)
				if helmApp == nil {
					break
				}

				deleteUpdateCacheForChart(helmApp.ChartPath)

				RemoveHelmApp(helmApp.Release.Name)
			default:
				secret, ok := e.Object.(*corev1.Secret)
				if !ok {
					break
				}
				logger.Debugf("%v event ignored for secret %s in namespace %s", e.Type, secret.Name, namespace)
			}
		}
		logger.Infof("watch of secrets in ns %s unexpectedly terminated. Reconnecting...\n", namespace)
		time.Sleep(time.Second * 5)
	}
}

func recalculateCachedUpdates(newHelmApp *apptypes.HelmApp) error {
	removeFromCachedUpdates(newHelmApp.ChartPath, newHelmApp.Release.Chart.Metadata.Version)

	currentHelmApp := GetHelmApp(newHelmApp.Release.Name)
	if currentHelmApp == nil {
		// no app installed yet
		return nil
	}
	updates := GetCachedUpdates(currentHelmApp.ChartPath)
	currentKotsKinds, err := GetKotsKindsFromHelmApp(currentHelmApp)
	if err != nil {
		return errors.Wrapf(err, "failed to get current config values")
	}

	// Check if new configuration has been applied and now required items have been set in pending updates.
	for _, update := range updates {
		if update.Status != storetypes.VersionPendingConfig {
			continue
		}

		kotsKinds, err := GetKotsKindsFromUpstreamChartVersion(currentHelmApp, currentKotsKinds.License.Spec.LicenseID, update.Tag)
		if err != nil {
			return errors.Wrapf(err, "failed to pull update %s for chart", update.Version)
		}
		kotsKinds.ConfigValues = currentKotsKinds.ConfigValues.DeepCopy()

		sequence := int64(-1)                                // TODO: do something sensible, this value isn't used
		registrySettings := registrytypes.RegistrySettings{} // TODO: private registries aren't supported yet
		t, err := kotsadmconfig.NeedsConfiguration(currentHelmApp.GetSlug(), sequence, currentHelmApp.GetIsAirgap(), &kotsKinds, registrySettings)
		if err != nil {
			return errors.Wrap(err, "failed to check if version needs configuration")
		}
		if !t {
			SetCachedUpdateStatus(currentHelmApp.ChartPath, update.Tag, storetypes.VersionPending)
		}
	}

	return nil
}
