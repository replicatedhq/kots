package helm

import (
	"bytes"
	"compress/gzip"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"sort"
	"strconv"
	"sync"
	"text/template"
	"time"

	"github.com/blang/semver"
	"github.com/pkg/errors"
	apptypes "github.com/replicatedhq/kots/pkg/app/types"
	"github.com/replicatedhq/kots/pkg/k8sutil"
	"github.com/replicatedhq/kots/pkg/kotsutil"
	"github.com/replicatedhq/kots/pkg/logger"
	"github.com/replicatedhq/kots/pkg/replicatedapp"
	"github.com/replicatedhq/kots/pkg/util"
	kotsv1beta1 "github.com/replicatedhq/kotskinds/apis/kots/v1beta1"
	kotsscheme "github.com/replicatedhq/kotskinds/client/kotsclientset/scheme"
	helmrelease "helm.sh/helm/v3/pkg/release"
	corev1 "k8s.io/api/core/v1"
	kuberneteserrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/kubernetes/scheme"
)

var (
	configSecretMutex sync.Mutex
)

// Secret labels from Helm v3 code:
//
// lbs.set("name", rls.Name)
// lbs.set("owner", owner)
// lbs.set("status", rls.Info.Status.String())
// lbs.set("version", strconv.Itoa(rls.Version))
type InstalledRelease struct {
	ReleaseName string
	Revision    int
	Version     string
	Semver      *semver.Version
	Status      helmrelease.Status
	DeployedOn  *time.Time
	ReleasedOn  *time.Time
}

type InstalledReleases []InstalledRelease

func (v InstalledReleases) Len() int {
	return len(v)
}

func (v InstalledReleases) Swap(i, j int) {
	v[i], v[j] = v[j], v[i]
}

func (v InstalledReleases) Less(i, j int) bool {
	return v[i].Version < v[j].Version
}

func GetChartSecret(releaseName, namespace, version string) (*helmrelease.Release, error) {
	clientSet, err := k8sutil.GetClientset()
	if err != nil {
		return nil, errors.Wrap(err, "failed to get clientset")
	}

	selectorLabels := map[string]string{
		"owner":   "helm",
		"name":    releaseName,
		"version": version,
	}
	listOpts := metav1.ListOptions{
		LabelSelector: labels.SelectorFromSet(selectorLabels).String(),
	}

	secrets, err := clientSet.CoreV1().Secrets(namespace).List(context.TODO(), listOpts)
	if err != nil {
		if kuberneteserrors.IsNotFound(err) {
			return nil, nil
		}
		return nil, errors.Wrap(err, "failed to list secrets")
	}
	if len(secrets.Items) > 1 {
		return nil, errors.New("found multiple secrets for single release revision")
	}

	helmRelease, err := HelmReleaseFromSecretData(secrets.Items[0].Data["release"])
	if err != nil {
		return nil, errors.Wrap(err, "failed to parse release info from secret")
	}

	return helmRelease, nil
}

func ListChartVersions(releaseName string, namespace string) ([]InstalledRelease, error) {
	clientSet, err := k8sutil.GetClientset()
	if err != nil {
		return nil, errors.Wrap(err, "failed to get clientset")
	}

	selectorLabels := map[string]string{
		"owner": "helm",
		"name":  releaseName,
	}
	listOpts := metav1.ListOptions{
		LabelSelector: labels.SelectorFromSet(selectorLabels).String(),
	}

	secrets, err := clientSet.CoreV1().Secrets(namespace).List(context.TODO(), listOpts)
	if err != nil {
		if kuberneteserrors.IsNotFound(err) {
			return InstalledReleases{}, nil
		}
		return nil, errors.Wrap(err, "failed to list secrets")
	}

	releases := InstalledReleases{}
	for _, secret := range secrets.Items {
		release, err := getChartVersionFromSecretData(&secret)
		if err != nil {
			logger.Warnf("failed to create release from secret chart %s revision number %v: %v", releaseName, secret.Labels["version"], err)
			continue
		}

		releases = append(releases, *release)
	}

	sort.Sort(sort.Reverse(releases))

	return releases, nil
}

func GetChartVersion(releaseName string, revision int64, namespace string) (*InstalledRelease, error) {
	clientSet, err := k8sutil.GetClientset()
	if err != nil {
		return nil, errors.Wrap(err, "failed to get clientset")
	}

	selectorLabels := map[string]string{
		"owner":   "helm",
		"name":    releaseName,
		"version": fmt.Sprintf("%d", revision),
	}
	listOpts := metav1.ListOptions{
		LabelSelector: labels.SelectorFromSet(selectorLabels).String(),
	}

	secrets, err := clientSet.CoreV1().Secrets(namespace).List(context.TODO(), listOpts)
	if err != nil {
		return nil, errors.Wrap(err, "failed to list secrets")
	}

	if len(secrets.Items) == 0 {
		return nil, nil
	}

	if len(secrets.Items) != 1 {
		return nil, errors.Errorf("found %d secrets but expected 1", len(secrets.Items))
	}

	release, err := getChartVersionFromSecretData(&secrets.Items[0])
	if err != nil {
		return nil, errors.Wrap(err, "failed to create release from secret")
	}

	return release, nil
}

func getChartVersionFromSecretData(secret *corev1.Secret) (*InstalledRelease, error) {
	revision, err := strconv.Atoi(secret.Labels["version"])
	if err != nil {
		return nil, errors.Wrap(err, "failed to parse release version")
	}

	helmRelease, err := HelmReleaseFromSecretData(secret.Data["release"])
	if err != nil {
		return nil, errors.Wrap(err, "failed to parse release data")
	}

	release := InstalledRelease{
		ReleaseName: secret.Labels["releaseName"],
		Revision:    revision,
		Status:      helmrelease.Status(secret.Labels["status"]),
		DeployedOn:  &secret.CreationTimestamp.Time,
	}

	createdAt := util.GetValueFromMapPath(helmRelease.Chart.Values, []string{"replicated", "created_at"})
	if s, ok := createdAt.(string); ok {
		t, err := time.Parse(time.RFC3339, s)
		if err == nil {
			release.ReleasedOn = &t
		}
	}

	if helmRelease.Chart != nil && helmRelease.Chart.Metadata != nil {
		release.Version = helmRelease.Chart.Metadata.Version
	}

	sv, err := semver.ParseTolerant(release.Version)
	if err != nil {
		return nil, errors.Wrap(err, "failed to parse release version")
	}
	release.Semver = &sv

	return &release, nil
}

func HelmReleaseFromSecretData(data []byte) (*helmrelease.Release, error) {
	base64Reader := base64.NewDecoder(base64.StdEncoding, bytes.NewReader(data))
	gzreader, err := gzip.NewReader(base64Reader)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create gzip reader")
	}
	defer gzreader.Close()

	releaseData, err := ioutil.ReadAll(gzreader)
	if err != nil {
		return nil, errors.Wrap(err, "failed to read from gzip reader")
	}

	release := &helmrelease.Release{}
	err = json.Unmarshal(releaseData, &release)
	if err != nil {
		return nil, errors.Wrap(err, "failed to unmarshal release data")
	}

	return release, nil
}

func GetChartConfigSecret(helmApp *apptypes.HelmApp) (*corev1.Secret, error) {
	clientSet, err := k8sutil.GetClientset()
	if err != nil {
		return nil, errors.Wrap(err, "failed to get clientset")
	}

	// Note that this is release name - chart name to support deploying multiple instances
	secretName := fmt.Sprintf("kots-%s-%s-config", helmApp.Release.Chart.Name(), helmApp.Release.Name)
	secret, err := clientSet.CoreV1().Secrets(helmApp.Namespace).Get(context.TODO(), secretName, metav1.GetOptions{})
	if err != nil {
		if kuberneteserrors.IsNotFound(err) {
			return nil, nil
		}
		return nil, errors.Wrap(err, "failed to get secret")
	}

	return secret, nil
}

func GetChartLicenseFromSecretOrDownload(helmApp *apptypes.HelmApp) (*kotsv1beta1.License, error) {
	configSecret, err := GetChartConfigSecret(helmApp)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get helm config secret")
	}

	if configSecret == nil {
		return nil, errors.Errorf("no config secret found for release %s", helmApp.Release.Name)
	}

	if licenseData := configSecret.Data["license"]; len(licenseData) > 0 {
		decode := kotsscheme.Codecs.UniversalDeserializer().Decode
		obj, gvk, err := decode(licenseData, nil, nil)
		if err != nil {
			return nil, errors.Wrap(err, "failed to decode license data")
		}

		if gvk.Group != "kots.io" || gvk.Version != "v1beta1" || gvk.Kind != "License" {
			return nil, errors.Errorf("unexpected GVK: %s", gvk.String())
		}

		return obj.(*kotsv1beta1.License), nil
	}

	license, err := downloadAppLicense(helmApp)
	if err != nil {
		return nil, errors.Wrap(err, "failed to download app license")
	}

	return license, nil
}

func downloadAppLicense(helmApp *apptypes.HelmApp) (*kotsv1beta1.License, error) {
	licenseID := GetKotsLicenseID(&helmApp.Release)
	if licenseID == "" {
		return nil, errors.Errorf("no license and no license ID found for release %s", helmApp.Release.Name)
	}

	licenseData, err := replicatedapp.GetLatestLicenseForHelm(licenseID)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get license for helm chart %s", helmApp.Release.Name)
	}

	if err := SaveChartLicenseInSecret(helmApp, licenseData.LicenseBytes); err != nil {
		return nil, errors.Wrapf(err, "failed save license in config for chart %s", helmApp.Release.Name)
	}

	return licenseData.License, nil
}

// Always save original data returned from the server without remarshaling.
func SaveChartLicenseInSecret(helmApp *apptypes.HelmApp, licenseData []byte) error {
	configSecretMutex.Lock()
	defer configSecretMutex.Unlock()

	secret, err := GetChartConfigSecret(helmApp)
	if err != nil {
		return errors.Wrap(err, "failed to get config secret for license update")
	}

	if secret == nil {
		return errors.Errorf("secret not found")
	}

	secret.Data["license"] = licenseData

	clientSet, err := k8sutil.GetClientset()
	if err != nil {
		return errors.Wrap(err, "failed to get clientset")
	}

	_, err = clientSet.CoreV1().Secrets(secret.Namespace).Update(context.TODO(), secret, metav1.UpdateOptions{})
	if err != nil {
		// TODO: retry on IsConflict
		return errors.Wrap(err, "failed to update secret")
	}

	return nil
}

func GetKotsKindsFromHelmApp(helmApp *apptypes.HelmApp) (kotsutil.KotsKinds, error) {
	kotsKinds := kotsutil.EmptyKotsKinds()

	secret, err := GetChartConfigSecret(helmApp)
	if err != nil {
		return kotsKinds, errors.Wrap(err, "failed to get helm config secret")
	}

	if secret == nil {
		return kotsKinds, nil
	}

	kotsKinds, err = GetKotsKindsFromReplicatedSecret(secret)
	if err != nil {
		return kotsKinds, errors.Wrap(err, "failed to get kots kinds from secret")
	}

	if kotsKinds.License == nil {
		license, err := downloadAppLicense(helmApp)
		if err != nil {
			return kotsKinds, errors.Wrap(err, "failed to download license")
		}
		kotsKinds.License = license
	}

	return kotsKinds, nil
}

func GetKotsKindsFromReplicatedSecret(secret *corev1.Secret) (kotsutil.KotsKinds, error) {
	kotsKinds := kotsutil.EmptyKotsKinds()

	licenseData := secret.Data["license"]
	if len(licenseData) != 0 {
		license, err := kotsutil.LoadLicenseFromBytes(licenseData)
		if err != nil {
			return kotsKinds, errors.Wrap(err, "failed to load license from data")
		}
		kotsKinds.License = license
	}

	configData := secret.Data["config"]
	if len(configData) != 0 {
		config, err := kotsutil.LoadConfigFromBytes(configData)
		if err != nil {
			return kotsKinds, errors.Wrap(err, "failed to load config from data")
		}
		kotsKinds.Config = config
	}

	configValuesData := secret.Data["configValues"]
	if len(configValuesData) != 0 {
		configValues, err := kotsutil.LoadConfigValuesFromBytes(configValuesData)
		if err != nil {
			return kotsKinds, errors.Wrap(err, "failed to load config values from data")
		}
		kotsKinds.ConfigValues = configValues
	}

	appData := secret.Data["application"]
	if len(appData) != 0 {
		app, err := kotsutil.LoadApplicationFromBytes(appData)
		if err != nil {
			return kotsKinds, errors.Wrap(err, "failed to load application from data")
		}
		kotsKinds.KotsApplication = *app
	}

	return kotsKinds, nil
}

func GetKotsKindsForRevision(releaseName string, revision int64, namespace string) (kotsutil.KotsKinds, error) {
	kotsKinds := kotsutil.EmptyKotsKinds()

	clientSet, err := k8sutil.GetClientset()
	if err != nil {
		return kotsKinds, errors.Wrap(err, "failed to get clientset")
	}

	selectorLabels := map[string]string{
		"owner":   "helm",
		"version": fmt.Sprintf("%d", revision),
		"name":    releaseName,
	}
	listOpts := metav1.ListOptions{
		LabelSelector: labels.SelectorFromSet(selectorLabels).String(),
	}

	secrets, err := clientSet.CoreV1().Secrets(namespace).List(context.TODO(), listOpts)
	if err != nil {
		return kotsKinds, errors.Wrap(err, "failed to list secrets")
	}

	if len(secrets.Items) != 1 {
		return kotsKinds, errors.Errorf("expected to match 1 secret, but found %d", len(secrets.Items))
	}

	chartSecret := secrets.Items[0]
	helmApp, err := helmAppFromSecret(&chartSecret)
	if err != nil {
		return kotsKinds, errors.Wrap(err, "failed to convert secret to helm app")
	}

	license, err := GetChartLicenseFromSecretOrDownload(helmApp)
	if err != nil {
		return kotsKinds, errors.Wrap(err, "failed to get license")
	}
	kotsKinds.License = license

	// "Config" object is in the template secret.
	for _, template := range helmApp.Release.Chart.Templates {
		if template.Name != "templates/_replicated/secret.yaml" {
			continue
		}

		secretData, err := removeHelmTemplate(template.Data)
		if err != nil {
			return kotsKinds, errors.Wrap(err, "failed to remove helm templates from replicated secret file")
		}

		decode := scheme.Codecs.UniversalDeserializer().Decode
		obj, gvk, err := decode(secretData, nil, nil)
		if err != nil {
			return kotsKinds, errors.Wrap(err, "failed to decode secret data")
		}

		if gvk.Group != "" || gvk.Version != "v1" || gvk.Kind != "Secret" {
			return kotsKinds, errors.Errorf("unexpected secret GVK: %s", gvk.String())
		}

		k, err := GetKotsKindsFromReplicatedSecret(obj.(*corev1.Secret))
		if err != nil {
			return kotsKinds, errors.Wrap(err, "failed to get kots kinds from secret")
		}

		kotsKinds.Config = k.Config
		kotsKinds.Application = k.Application

		break
	}

	// If chart was deployed with --values, they will be in Config.  Otherwise, get the default values injected by the registry
	encodedConfigValues := util.GetValueFromMapPath(helmApp.Release.Config, []string{"replicated", "app", "configValues"})
	if encodedConfigValues == nil {
		encodedConfigValues = util.GetValueFromMapPath(helmApp.Release.Chart.Values, []string{"replicated", "app", "configValues"})
	}

	if encodedConfigValues == nil {
		return kotsKinds, errors.Errorf("failed to find configValues from release %s", helmApp.Release.Name)
	}

	configValuesData, err := util.Base64DecodeInterface(encodedConfigValues)
	if err != nil {
		return kotsKinds, errors.Wrap(err, "failed to base64 decode config values from chart release")
	}

	kotsKinds.ConfigValues, err = kotsutil.LoadConfigValuesFromBytes(configValuesData)
	if err != nil {
		return kotsKinds, errors.Wrap(err, "failed to get config values from chart values")
	}

	return kotsKinds, nil
}

func GetReplicatedSecretForRevision(releaseName string, revision int64, namespace string) (*corev1.Secret, error) {
	clientSet, err := k8sutil.GetClientset()
	if err != nil {
		return nil, errors.Wrap(err, "failed to get clientset")
	}

	selectorLabels := map[string]string{
		"owner":   "helm",
		"version": fmt.Sprintf("%d", revision),
		"name":    releaseName,
	}
	listOpts := metav1.ListOptions{
		LabelSelector: labels.SelectorFromSet(selectorLabels).String(),
	}

	secrets, err := clientSet.CoreV1().Secrets(namespace).List(context.TODO(), listOpts)
	if err != nil {
		return nil, errors.Wrap(err, "failed to list secrets")
	}

	if len(secrets.Items) != 1 {
		return nil, errors.Errorf("expected to match 1 secret, but found %d", len(secrets.Items))
	}

	chartSecret := secrets.Items[0]
	helmApp, err := helmAppFromSecret(&chartSecret)
	if err != nil {
		return nil, errors.Wrap(err, "failed to convert secret to helm app")
	}

	for _, template := range helmApp.Release.Chart.Templates {
		if template.Name != "templates/_replicated/secret.yaml" {
			continue
		}

		secretData, err := removeHelmTemplate(template.Data)
		if err != nil {
			return nil, errors.Wrap(err, "failed to remove helm templates from replicated secret file")
		}

		decode := scheme.Codecs.UniversalDeserializer().Decode
		obj, gvk, err := decode(secretData, nil, nil)
		if err != nil {
			return nil, errors.Wrap(err, "failed to decode secret data")
		}

		if gvk.Group != "" || gvk.Version != "v1" || gvk.Kind != "Secret" {
			return nil, errors.Errorf("unexpected secret GVK: %s", gvk.String())
		}

		return obj.(*corev1.Secret), nil
	}

	return nil, errors.Errorf("replicated secret template not found for chart %q, revision %d, in ns %q", releaseName, revision, namespace)
}

func removeHelmTemplate(doc []byte) ([]byte, error) {
	type Inventory struct {
		Material string
		Count    uint
	}
	replicatedValues := map[string]interface{}{
		"Values": map[string]interface{}{
			"replicated": map[string]interface{}{
				"app": map[string]interface{}{
					"configValues": base64.RawStdEncoding.EncodeToString(nil),
				},
			},
		},
	}
	tmpl, err := template.New("sanitize-helm").Parse(string(doc))
	if err != nil {
		return nil, errors.Wrap(err, "failed to parse doc as template")
	}

	b := bytes.NewBuffer(nil)
	err = tmpl.Execute(b, replicatedValues)
	if err != nil {
		return nil, errors.Wrap(err, "failed to execute template")
	}

	return b.Bytes(), nil
}
