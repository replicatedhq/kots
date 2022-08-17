package helm

import (
	"bytes"
	"context"
	"io/ioutil"
	"strconv"
	"sync"
	"time"

	"github.com/pkg/errors"
	apptypes "github.com/replicatedhq/kots/pkg/app/types"
	"github.com/replicatedhq/kots/pkg/k8sutil"
	kotslicense "github.com/replicatedhq/kots/pkg/license"
	"github.com/replicatedhq/kots/pkg/logger"
	"github.com/replicatedhq/kots/pkg/util"
	helmrelease "helm.sh/helm/v3/pkg/release"
	corev1 "k8s.io/api/core/v1"
	kuberneteserrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	serializer "k8s.io/apimachinery/pkg/runtime/serializer/json"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/kubernetes/scheme"
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

	licenseID := GetKotsLicenseID(helmRelease)
	if licenseID == "" { // not a kots managed chart
		return nil, nil
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

				removeFromCachedUpdates(helmApp.ChartPath, helmApp.Release.Chart.Metadata.Version)

				if err := finalizeChartConfig(helmApp); err != nil {
					logger.Errorf("failed to copy chart config from temp secret into helm release: %v", err)
				}

				logger.Debugf("adding secret %s to cache", secret.Name)
				AddHelmApp(helmApp.Release.Name, helmApp)

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

// Find replicated secret in helm release's templates, decode it, add config values to it, save helm release
func finalizeChartConfig(helmApp *apptypes.HelmApp) error {
	configValues, err := GetTempConfigValues(helmApp)
	if err != nil {
		if kuberneteserrors.IsNotFound(errors.Cause(err)) {
			return nil
		}
		return errors.Wrap(err, "failed to get temp config values")
	}

	s := serializer.NewYAMLSerializer(serializer.DefaultMetaFactory, scheme.Scheme, scheme.Scheme)
	var configValuesBuffer bytes.Buffer
	if err := s.Encode(configValues, &configValuesBuffer); err != nil {
		return errors.Wrap(err, "failed to encode config values")
	}
	configValuesData := configValuesBuffer.Bytes()

	for _, template := range helmApp.Release.Chart.Templates {
		if template.Name != "templates/_replicated/secret.yaml" {
			continue
		}

		decode := scheme.Codecs.UniversalDeserializer().Decode
		decoded, gvk, err := decode(template.Data, nil, nil)
		if err != nil {
			return errors.Wrap(err, "failed to decode replicated secret template")
		}

		if gvk.Group != "" || gvk.Version != "v1" || gvk.Kind != "Secret" {
			return errors.Errorf("%q is not a valid Secret GVK", gvk.String())
		}

		replicatedSecret := decoded.(*corev1.Secret)
		if _, ok := replicatedSecret.Data["configValues"]; ok {
			return errors.Errorf("replicated secret for chart %s in ns %s already has configValues", helmApp.Release.Name, helmApp.Namespace)
		}

		replicatedSecret.Data["configValues"] = configValuesData

		var replicatedSecretData bytes.Buffer
		if err := s.Encode(replicatedSecret, &replicatedSecretData); err != nil {
			return errors.Wrap(err, "failed to encode config values")
		}

		template.Data = replicatedSecretData.Bytes()
		break
	}

	// Deleting first because saving Helm secret will send another update event here
	if err := deleteTempConfigValues(helmApp); err != nil {
		return errors.Wrap(err, "failed to delete temp config values")
	}

	if err := saveHelmApp(helmApp); err != nil {
		return errors.Wrap(err, "failed to save helm release")
	}

	configSecret, err := GetChartConfigSecret(helmApp)
	if err != nil {
		return errors.Wrap(err, "failed to get config secret for license update")
	}

	if configSecret == nil {
		return errors.Errorf("secret not found")
	}

	configSecret.Data["configValues"] = configValuesData
	if err := UpdateChartConfig(configSecret); err != nil {
		return errors.Wrap(err, "failed to update config secret")
	}

	return nil
}
