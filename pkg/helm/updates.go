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

	"github.com/blang/semver"
	"github.com/containers/image/v5/docker"
	imagetypes "github.com/containers/image/v5/types"
	"github.com/pkg/errors"
	apptypes "github.com/replicatedhq/kots/pkg/app/types"
	"github.com/replicatedhq/kots/pkg/logger"
	"go.uber.org/zap"
	helmgetter "helm.sh/helm/v3/pkg/getter"
)

type ChartUpdate struct {
	Tag     string
	Version semver.Version
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

func CheckForUpdates(chartPath string, licenseID string, currentVersion *semver.Version) (ChartUpdates, error) {
	availableUpdates := ChartUpdates{}

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
			Tag:     tag,
			Version: v,
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

func PullChartVersion(helmApp *apptypes.HelmApp, licenseID string, version string) (*bytes.Buffer, error) {
	data, err := getUpdateChartFromCache(helmApp, version)
	if err != nil {
		logger.Info("failed to get chart release from cache", zap.String("error", err.Error()))
	}

	if data != nil {
		return data, nil
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
	data, err = chartGetter.Get(imageName)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get chart %q", imageName)
	}

	data, err = saveUpdateChartInCache(helmApp, version, data)
	if err != nil {
		logger.Info("failed to save chart in cache", zap.String("error", err.Error()))
	}

	return data, nil
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
