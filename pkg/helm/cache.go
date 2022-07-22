package helm

import (
	"context"
	"io/ioutil"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/pkg/errors"
	"github.com/replicatedhq/kots/pkg/k8sutil"
	kotslicense "github.com/replicatedhq/kots/pkg/license"
	"github.com/replicatedhq/kots/pkg/logger"
	"github.com/replicatedhq/kots/pkg/util"
	helmrelease "helm.sh/helm/v3/pkg/release"
	corev1 "k8s.io/api/core/v1"
	kuberneteserrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/watch"
)

type HelmApp struct {
	Release           helmrelease.Release
	Labels            map[string]string
	Namespace         string
	IsConfigurable    bool
	ChartPath         string
	CreationTimestamp time.Time
	PathToValuesFile  string
}

// TODO: Support same releases names in different namespaces
var (
	helmAppCache  = map[string]*HelmApp{}
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

			AddHelmApp(releaseInfo.Release.Name, releaseInfo)
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

func GetHelmApp(releaseName string) *HelmApp {
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

func AddHelmApp(releaseName string, helmApp *HelmApp) {
	appCacheLock.Lock()
	defer appCacheLock.Unlock()

	helmAppCache[releaseName] = helmApp
}

func RemoveHelmApp(releaseName string) {
	appCacheLock.Lock()
	defer appCacheLock.Unlock()

	delete(helmAppCache, releaseName)
}

func SaveConfigValuesToFile(helmApp *HelmApp, data []byte) error {
	err := os.MkdirAll(filepath.Dir(helmApp.PathToValuesFile), 0744)
	if err != nil {
		return errors.Wrap(err, "failed to create directory")
	}

	err = ioutil.WriteFile(helmApp.PathToValuesFile, data, 0644)
	if err != nil {
		return errors.Wrap(err, "failed to save values to file")
	}

	return nil
}

func helmAppFromSecret(secret *corev1.Secret) (*HelmApp, error) {
	helmRelease, err := HelmReleaseFromSecretData(secret.Data["release"])
	if err != nil {
		return nil, errors.Wrap(err, "failed to get helm release from secret")
	}

	licenseID := GetKotsLicenseID(helmRelease)
	if licenseID == "" { // not a kots managed chart
		return nil, nil
	}

	helmApp := &HelmApp{
		Release:           *helmRelease,
		Labels:            secret.Labels,
		CreationTimestamp: secret.CreationTimestamp.Time,
		Namespace:         secret.Namespace,
		PathToValuesFile:  filepath.Join(tmpValuesRoot, "helm", helmRelease.Name, "values.yaml"),
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

	if configSecret.Data["license"] == nil {
		// This block does not return an error.
		// This allows the app cache to be populated and app be accessible via Admin Console.
		// If there is no license, the license card will be empty.
		// License data can be healed by syncing the license from Admin Console.
		licenseData, err := kotslicense.GetLatestLicenseForHelm(licenseID)
		if err != nil {
			logger.Warnf("failed to get license for helm chart %s: %v", helmRelease.Name, err)
		} else {
			configSecret.Data["license"] = licenseData.LicenseBytes
			err := UpdateChartConfig(configSecret)
			if err != nil {
				logger.Warnf("failed to save license for helm chart %s: %v", helmRelease.Name, err)
			}
		}
	}

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
				if secret.Labels == nil || secret.Labels["status"] != helmrelease.StatusDeployed.String() {
					continue
				}
				releaseInfo, err := helmAppFromSecret(secret)
				if err != nil {
					logger.Errorf("failed to create helm release info from secret %s in namespace %s: %s", secret.Name, namespace)
					break
				}
				if releaseInfo == nil {
					break
				}

				removeFromCachedUpdates(releaseInfo.ChartPath, releaseInfo.Release.Chart.Metadata.Version)

				logger.Debugf("adding secret %s to cache", secret.Name)
				AddHelmApp(releaseInfo.Release.Name, releaseInfo)

			case watch.Deleted:
				secret, ok := e.Object.(*corev1.Secret)
				if !ok {
					break
				}

				releaseInfo, err := helmAppFromSecret(secret)
				if err != nil {
					logger.Errorf("failed to create helm release info from secret %s in namespace %s: %s", secret.Name, namespace)
					break
				}
				if releaseInfo == nil {
					break
				}

				deleteUpdateCacheForChart(releaseInfo.ChartPath)

				RemoveHelmApp(releaseInfo.Release.Name)

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
