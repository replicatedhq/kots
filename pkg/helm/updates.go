package helm

import (
	"bytes"
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/blang/semver"
	"github.com/containers/image/v5/docker"
	imagetypes "github.com/containers/image/v5/types"
	"github.com/pkg/errors"
	apptypes "github.com/replicatedhq/kots/pkg/app/types"
	"github.com/replicatedhq/kots/pkg/k8sutil"
	kotsadmtypes "github.com/replicatedhq/kots/pkg/kotsadm/types"
	"github.com/replicatedhq/kots/pkg/kotsutil"
	"github.com/replicatedhq/kots/pkg/logger"
	storetypes "github.com/replicatedhq/kots/pkg/store/types"
	"github.com/replicatedhq/kots/pkg/util"
	kotsv1beta1 "github.com/replicatedhq/kotskinds/apis/kots/v1beta1"
	"go.uber.org/zap"
	"gopkg.in/yaml.v3"
	helmgetter "helm.sh/helm/v3/pkg/getter"
	corev1 "k8s.io/api/core/v1"
	kuberneteserrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/scheme"
)

type ChartUpdate struct {
	Tag          string
	Version      semver.Version
	Status       storetypes.DownstreamVersionStatus
	CreatedOn    *time.Time
	IsDownloaded bool
}

type ReplicatedMeta struct {
	LicenseID string     `yaml:"license_id"`
	Username  string     `yaml:"username"`
	CreatedAt *time.Time `yaml:"created_at"`
	// TODO "app": map[string][]byte
}

type ChartUpdates []ChartUpdate

var (
	updateCacheMutex sync.Mutex
	updateCache      map[string]ChartUpdates // available updates sorted in descending order for each chart
)

func init() {
	updateCache = make(map[string]ChartUpdates)
}

func (v ChartUpdates) Len() int {
	return len(v)
}

func (v ChartUpdates) Swap(i, j int) {
	v[i], v[j] = v[j], v[i]
}

func (v ChartUpdates) Less(i, j int) bool {
	return v[i].Version.LT(v[j].Version)
}

func (u ChartUpdates) ToTagList() []string {
	tags := []string{}
	for _, update := range u {
		tags = append(tags, update.Tag)
	}
	return tags
}

func GetCachedUpdates(chartPath string) ChartUpdates {
	updateCacheMutex.Lock()
	defer updateCacheMutex.Unlock()

	return updateCache[chartPath]
}

func GetDownloadedUpdates(chartPath string) ChartUpdates {
	updateCacheMutex.Lock()
	defer updateCacheMutex.Unlock()

	downloadedUpdates := ChartUpdates{}
	for _, u := range updateCache[chartPath] {
		if u.IsDownloaded {
			downloadedUpdates = append(downloadedUpdates, u)
		}
	}
	return downloadedUpdates
}

func SetCachedUpdateStatus(chartPath string, tag string, status storetypes.DownstreamVersionStatus) {
	updates := GetCachedUpdates(chartPath)
	for i, u := range updates {
		if u.Tag == tag {
			updates[i].Status = status
			break
		}
	}
}

func SetCachedUpdateMetadata(chartPath string, tag string, meta *ReplicatedMeta) {
	updates := GetCachedUpdates(chartPath)
	for i, u := range updates {
		if u.Tag == tag {
			updates[i].IsDownloaded = true // Metadata is only known when version is downloaded
			if meta != nil {
				updates[i].CreatedOn = meta.CreatedAt
			}
			break
		}
	}
}

func setCachedUpdates(chartPath string, updates ChartUpdates) {
	updateCacheMutex.Lock()
	defer updateCacheMutex.Unlock()

	updateCache[chartPath] = updates
}

// Removes this tag from cache and also every tag that is less than this one according to semver ordering
func removeFromCachedUpdates(chartPath string, tag string) {
	updateCacheMutex.Lock()
	defer updateCacheMutex.Unlock()

	version, parseErr := semver.ParseTolerant(tag)

	existingList := updateCache[chartPath]
	newList := ChartUpdates{}
	for _, update := range existingList {
		// If tag cannot be parsed, fall back on string comparison.
		// This should never happen for versions that are on the list because we only include valid semvers and Helm chart versions are valid semvers.
		if parseErr != nil {
			if update.Tag != tag {
				newList = append(newList, update)
			}
		} else if update.Version.GT(version) {
			newList = append(newList, update)
		}
	}
	updateCache[chartPath] = newList
}

func deleteUpdateCacheForChart(chartPath string) {
	updateCacheMutex.Lock()
	defer updateCacheMutex.Unlock()

	delete(updateCache, chartPath)
}

func CheckForUpdates(helmApp *apptypes.HelmApp, license *kotsv1beta1.License, currentVersion *semver.Version) (ChartUpdates, error) {
	chartPath := helmApp.ChartPath
	licenseID := license.Spec.LicenseID

	_, err := SyncLicense(helmApp)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to sync license before update check")
	}

	imageName := strings.TrimLeft(chartPath, "oci:")
	ref, err := docker.ParseReference(imageName)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to parse image ref %q", imageName)
	}

	sysCtx := &imagetypes.SystemContext{
		DockerInsecureSkipTLSVerify: imagetypes.OptionalBoolTrue,
		DockerDisableV1Ping:         true,
		DockerAuthConfig: &imagetypes.DockerAuthConfig{
			Username: licenseID,
			Password: licenseID,
		},
	}

	tags, err := docker.GetRepositoryTags(context.TODO(), sysCtx, ref)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get repo tags")
	}

	tags = removeDuplicates(tags) // registry should not be returning duplicate tags

	availableUpdates := ChartUpdates{}
	for _, tag := range tags {
		v, err := semver.ParseTolerant(tag)
		if err != nil {
			// TODO: log
			continue
		}

		if currentVersion != nil && v.LE(*currentVersion) {
			continue
		}

		availableUpdates = append(availableUpdates, ChartUpdate{
			Tag:          tag,
			Version:      v,
			IsDownloaded: false,
		})
	}

	sort.Sort(sort.Reverse(ChartUpdates(availableUpdates)))

	setCachedUpdates(chartPath, availableUpdates)

	return availableUpdates, nil
}

func removeDuplicates(tags []string) []string {
	m := map[string]struct{}{}
	for _, tag := range tags {
		m[tag] = struct{}{}
	}

	u := []string{}
	for k := range m {
		u = append(u, k)
	}

	return u
}

func GetKotsKindsFromUpstreamChartVersion(helmApp *apptypes.HelmApp, licenseID string, version string) (kotsutil.KotsKinds, error) {
	secret, err := GetReplicatedSecretFromUpstreamChartVersion(helmApp, licenseID, version)
	if err != nil {
		return kotsutil.KotsKinds{}, errors.Wrap(err, "failed to get secret upstream archive")
	}

	kotsKinds, err := GetKotsKindsFromReplicatedSecret(secret)
	if err != nil {
		return kotsKinds, errors.Wrap(err, "failed to get kots kinds from secret")
	}

	return kotsKinds, nil
}

func GetReplicatedSecretFromUpstreamChartVersion(helmApp *apptypes.HelmApp, licenseID string, version string) (*corev1.Secret, error) {
	chartData, err := downloadChartReleaseIfNeeded(helmApp, licenseID, version)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get chart data")
	}

	templatedData, err := util.GetFileFromTGZArchive(chartData, "**/templates/_replicated/secret.yaml")
	if err != nil {
		return nil, errors.Wrap(err, "failed to get secret file from chart archive")
	}

	secretData, err := removeHelmTemplate(templatedData.Bytes())
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

func GetReplicatedMetadataFromUpstreamChartVersion(helmApp *apptypes.HelmApp, licenseID string, version string) (*ReplicatedMeta, error) {
	chartData, err := downloadChartReleaseIfNeeded(helmApp, licenseID, version)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get chart data")
	}

	valuesData, err := util.GetFileFromTGZArchive(chartData, "**/values.yaml")
	if err != nil {
		return nil, errors.Wrap(err, "failed to get values.yaml from chart archive")
	}

	b := valuesData.Bytes()
	values := struct {
		Replicated *ReplicatedMeta `yaml:"replicated,omitempty"`
	}{}
	err = yaml.Unmarshal(b, &values)
	if err != nil {
		return nil, errors.Wrap(err, "failed to decode values.yaml")
	}

	return values.Replicated, nil
}

func downloadChartReleaseIfNeeded(helmApp *apptypes.HelmApp, licenseID string, version string) (*bytes.Buffer, error) {
	chartData, err := getUpdateChartFromCache(helmApp, version)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get release from cache")
	}

	if chartData != nil {
		return chartData, nil
	}

	err = CreateHelmRegistryCreds(licenseID, licenseID, helmApp.ChartPath)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create helm credentials file")
	}
	chartGetter, err := helmgetter.NewOCIGetter()
	if err != nil {
		return nil, errors.Wrap(err, "failed to create chart getter")
	}

	imageName := fmt.Sprintf("%s:%s", helmApp.ChartPath, version)
	chartData, err = chartGetter.Get(imageName)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get chart %q", imageName)
	}

	chartData, err = saveUpdateChartInCache(helmApp, version, chartData)
	if err != nil {
		logger.Info("failed to save chart in cache", zap.String("error", err.Error()))
	}

	return chartData, nil
}

var (
	updateCacheDir = ""
)

func getUpdateChartFromCache(helmApp *apptypes.HelmApp, version string) (*bytes.Buffer, error) {
	fileName := getUpdateChacheFileName(helmApp, version)
	b, err := ioutil.ReadFile(fileName)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, errors.Wrap(err, "failed to read file")
	}

	return bytes.NewBuffer(b), nil
}

func saveUpdateChartInCache(helmApp *apptypes.HelmApp, version string, data *bytes.Buffer) (*bytes.Buffer, error) {
	b := data.Bytes()
	newBuff := bytes.NewBuffer(b)

	if updateCacheDir == "" {
		dirName, err := ioutil.TempDir("", "chart-updates-")
		if err != nil {
			return newBuff, errors.Wrap(err, "failed to create temp dir")
		}
		updateCacheDir = dirName
	}

	fileName := getUpdateChacheFileName(helmApp, version)

	err := os.MkdirAll(filepath.Dir(fileName), 0755)
	if err != nil {
		return newBuff, errors.Wrap(err, "failed to create cache dir")
	}

	err = ioutil.WriteFile(fileName, b, 0744)
	if err != nil {
		return newBuff, errors.Wrap(err, "failed to save cache file")
	}

	return newBuff, nil
}

func getUpdateChacheFileName(helmApp *apptypes.HelmApp, version string) string {
	return filepath.Join(updateCacheDir, strings.TrimPrefix(helmApp.ChartPath, "oci://"), fmt.Sprintf("%s.tgz", version))
}

func GetUpdateCheckSpec(helmApp *apptypes.HelmApp) (string, error) {
	clientset, err := k8sutil.GetClientset()
	if err != nil {
		return "", errors.Wrap(err, "failed to get clientset")
	}

	spec := "@default"

	cm, err := clientset.CoreV1().ConfigMaps(util.PodNamespace).Get(context.TODO(), kotsadmtypes.KotsadmConfigMap, metav1.GetOptions{})
	if err != nil && !kuberneteserrors.IsNotFound(err) {
		return "", errors.Wrap(err, "failed to get configmap")
	} else if kuberneteserrors.IsNotFound(err) {
		return spec, nil
	}

	if cm.Data == nil {
		return spec, nil
	}

	key := fmt.Sprintf("update-schedule-%s", helmApp.GetID())
	if s := cm.Data[key]; s != "" {
		spec = s
	}

	return spec, nil
}

var configMapMutex sync.Mutex

func SetUpdateCheckSpec(helmApp *apptypes.HelmApp, updateSpec string) error {
	configMapMutex.Lock()
	defer configMapMutex.Unlock()

	clientset, err := k8sutil.GetClientset()
	if err != nil {
		return errors.Wrap(err, "failed to get clientset")
	}

	key := fmt.Sprintf("update-schedule-%s", helmApp.GetID())

	cm, err := clientset.CoreV1().ConfigMaps(util.PodNamespace).Get(context.TODO(), kotsadmtypes.KotsadmConfigMap, metav1.GetOptions{})
	if err != nil && !kuberneteserrors.IsNotFound(err) {
		return errors.Wrap(err, "failed to get configmap")
	} else if kuberneteserrors.IsNotFound(err) {
		cm := &corev1.ConfigMap{
			TypeMeta: metav1.TypeMeta{
				APIVersion: "v1",
				Kind:       "ConfigMap",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      kotsadmtypes.KotsadmConfigMap,
				Namespace: util.PodNamespace,
				Labels:    kotsadmtypes.GetKotsadmLabels(),
			},
			Data: map[string]string{
				key: updateSpec,
			},
		}
		_, err = clientset.CoreV1().ConfigMaps(util.PodNamespace).Create(context.Background(), cm, metav1.CreateOptions{})
		if err != nil {
			return errors.Wrap(err, "failed to update config map")
		}
	}

	if cm.Data == nil {
		cm.Data = map[string]string{}
	}
	cm.Data[key] = updateSpec

	_, err = clientset.CoreV1().ConfigMaps(util.PodNamespace).Update(context.Background(), cm, metav1.UpdateOptions{})
	if err != nil {
		return errors.Wrap(err, "failed to update config map")
	}

	return nil
}
