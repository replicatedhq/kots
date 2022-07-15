package helm

import (
	"bytes"
	"compress/gzip"
	"context"
	"encoding/base64"
	"encoding/json"
	"io/ioutil"
	"sort"
	"strconv"

	"github.com/blang/semver"
	"github.com/pkg/errors"
	"github.com/replicatedhq/kots/pkg/k8sutil"
	"github.com/replicatedhq/kots/pkg/logger"
	helmrelease "helm.sh/helm/v3/pkg/release"
	kuberneteserrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
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
