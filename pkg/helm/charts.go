package helm

import (
	"bufio"
	"bytes"
	"compress/gzip"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"sort"
	"strconv"
	"sync"

	"github.com/blang/semver"
	"github.com/pkg/errors"
	kotsv1beta1 "github.com/replicatedhq/kots/kotskinds/apis/kots/v1beta1"
	kotsscheme "github.com/replicatedhq/kots/kotskinds/client/kotsclientset/scheme"
	apptypes "github.com/replicatedhq/kots/pkg/app/types"
	"github.com/replicatedhq/kots/pkg/k8sutil"
	"github.com/replicatedhq/kots/pkg/kotsutil"
	kotslicense "github.com/replicatedhq/kots/pkg/license"
	"github.com/replicatedhq/kots/pkg/logger"
	"github.com/replicatedhq/kots/pkg/util"
	"gopkg.in/yaml.v2"
	helmrelease "helm.sh/helm/v3/pkg/release"
	corev1 "k8s.io/api/core/v1"
	kuberneteserrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
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
		revision, err := strconv.Atoi(secret.Labels["version"])
		if err != nil {
			logger.Warnf("failed to parse chart %s revision number %v: %v", releaseName, secret.Labels["version"], err)
			continue
		}

		helmRelease, err := HelmReleaseFromSecretData(secret.Data["release"])
		if err != nil {
			logger.Warnf("failed to parse chart %s release info: %v", releaseName, err)
			continue
		}

		release := InstalledRelease{
			ReleaseName: releaseName,
			Revision:    revision,
			Status:      helmrelease.Status(secret.Labels["status"]),
		}

		if helmRelease.Chart != nil && helmRelease.Chart.Metadata != nil {
			release.Version = helmRelease.Chart.Metadata.Version
		}

		sv, err := semver.ParseTolerant(release.Version)
		if err != nil {
			logger.Warnf("failed to parse chart %s version %s: %v", releaseName, release.Version, err)
			continue
		}
		release.Semver = &sv

		releases = append(releases, release)
	}

	sort.Sort(sort.Reverse(releases))

	return releases, nil
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

func HelmReleaseToSecretData(release *helmrelease.Release) ([]byte, error) {
	jsonData, err := json.Marshal(release)
	if err != nil {
		return nil, errors.Wrap(err, "failed to marshal release data")
	}

	var data bytes.Buffer
	dataWriter := bufio.NewWriter(&data)

	gzwriter := gzip.NewWriter(dataWriter)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create gzip reader")
	}
	defer gzwriter.Close()

	base64Writer := base64.NewEncoder(base64.StdEncoding, gzwriter)

	_, err = io.Copy(base64Writer, bytes.NewReader(jsonData))
	if err != nil {
		return nil, errors.Wrap(err, "failed to copy data")
	}

	return data.Bytes(), nil
}

func GetChartConfigSecret(helmApp *apptypes.HelmApp) (*corev1.Secret, error) {
	clientSet, err := k8sutil.GetClientset()
	if err != nil {
		return nil, errors.Wrap(err, "failed to get clientset")
	}

	// Note that this must be chart name, not release name
	secretName := fmt.Sprintf("kots-%s-config", helmApp.Release.Chart.Name())
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

	licenseData, err := kotslicense.GetLatestLicenseForHelm(licenseID)
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

func GetKotsKinds(helmApp *apptypes.HelmApp) (kotsutil.KotsKinds, error) {
	kotsKinds := kotsutil.EmptyKotsKinds()

	secret, err := GetChartConfigSecret(helmApp)
	if err != nil {
		return kotsKinds, errors.Wrap(err, "failed to get helm config secret")
	}

	if secret == nil {
		return kotsKinds, nil
	}

	licenseData := secret.Data["license"]
	if len(licenseData) == 0 {
		license, err := downloadAppLicense(helmApp)
		if err != nil {
			return kotsKinds, errors.Wrap(err, "failed to download license")
		}
		kotsKinds.License = license
	} else {
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

	return kotsKinds, nil
}

func GetKotsKindsForRevision(releaseName string, revision int64) (kotsutil.KotsKinds, error) {
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

	secrets, err := clientSet.CoreV1().Secrets(util.PodNamespace).List(context.TODO(), listOpts)
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

		// Can't use client-go decoder here because raw data contains templates.
		secret := map[string]interface{}{}
		if err := yaml.Unmarshal(template.Data, &secret); err != nil {
			return kotsKinds, errors.Wrap(err, "failed to unmarshal secret data")
		}
		encodedConfig := util.GetValueFromMapPath(secret, []string{"data", "config"})

		configData, err := util.Base64DecodeInterface(encodedConfig)
		if err != nil {
			return kotsKinds, errors.Wrap(err, "failed to base64 decode config from chart templates")
		}
		kotsKinds.Config, err = kotsutil.LoadConfigFromBytes(configData)
		if err != nil {
			return kotsKinds, errors.Wrap(err, "failed to get config from chart templates")
		}

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
