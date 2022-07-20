package helm

import (
	"context"
	"sort"
	"strings"
	"sync"

	"github.com/blang/semver"
	"github.com/containers/image/v5/docker"
	imagetypes "github.com/containers/image/v5/types"
	"github.com/pkg/errors"
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
