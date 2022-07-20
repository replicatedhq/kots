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

	"github.com/blang/semver"
	"github.com/pkg/errors"
	kotsv1beta1 "github.com/replicatedhq/kots/kotskinds/apis/kots/v1beta1"
	kotsscheme "github.com/replicatedhq/kots/kotskinds/client/kotsclientset/scheme"
	"github.com/replicatedhq/kots/pkg/k8sutil"
	"github.com/replicatedhq/kots/pkg/logger"
	helmrelease "helm.sh/helm/v3/pkg/release"
	corev1 "k8s.io/api/core/v1"
	kuberneteserrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	serializer "k8s.io/apimachinery/pkg/runtime/serializer/json"
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

func GetChartConfigSecret(helmApp *HelmApp) (*corev1.Secret, error) {
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

func UpdateChartConfig(secret *corev1.Secret) error {
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

func GetChartLicenseFromSecret(helmApp *HelmApp) (*kotsv1beta1.License, error) {
	secret, err := GetChartConfigSecret(helmApp)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get helm config secret")
	}

	if secret == nil {
		return nil, nil
	}

	licenseData := secret.Data["license"]
	if len(licenseData) == 0 {
		return nil, nil
	}

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

func SaveChartLicenseInSecret(helmApp *HelmApp, license *kotsv1beta1.License) error {
	configSecretMutex.Lock()
	defer configSecretMutex.Unlock()

	secret, err := GetChartConfigSecret(helmApp)
	if err != nil {
		return errors.Wrap(err, "failed to get config secret for license update")
	}

	if secret == nil {
		return errors.Errorf("secret not found")
	}

	s := serializer.NewYAMLSerializer(serializer.DefaultMetaFactory, kotsscheme.Scheme, kotsscheme.Scheme)
	var b bytes.Buffer
	if err := s.Encode(license, &b); err != nil {
		return errors.Wrap(err, "failed to encode kots license")
	}

	secret.Data["license"] = b.Bytes()

	if err := UpdateChartConfig(secret); err != nil {
		return errors.Wrap(err, "failed to update config secret with new license")
	}

	return nil
}
