package imageworker

import (
	"sort"
	"testing"
	"time"

	"github.com/replicatedhq/ship-cluster/worker/pkg/testing/logger"
	"github.com/stretchr/testify/require"

	semver "github.com/hashicorp/go-version"
)

func makeVersions(versions []string) []*semver.Version {
	allVersions := make([]*semver.Version, 0)
	for _, version := range versions {
		v, _ := semver.NewVersion(version)
		allVersions = append(allVersions, v)
	}
	return allVersions
}

func makeOriginal(versions []*semver.Version) []string {
	allVersions := make([]string, 0)
	for _, version := range versions {
		v := version.Original()
		allVersions = append(allVersions, v)
	}
	return allVersions
}

func TestTagCollectionSort(t *testing.T) {
	tests := []struct {
		name           string
		versions       []string
		expectVersions []string
	}{
		{
			name:           "same major versions",
			versions:       []string{"10", "10.4"},
			expectVersions: []string{"10.4", "10"},
		},
		{
			name:           "different major versions",
			versions:       []string{"10", "11.1"},
			expectVersions: []string{"10", "11.1"},
		},
		{
			name:           "same major and minor versions",
			versions:       []string{"9.1.3", "9.1.0", "9.1.4", "9.1"},
			expectVersions: []string{"9.1.0", "9.1.3", "9.1.4", "9.1"},
		},
		{
			name:           "different major and minor versions",
			versions:       []string{"10.1.2", "10.0", "10", "10.3.2", "11", "10.1.3", "10.1"},
			expectVersions: []string{"10.0", "10.1.2", "10.1.3", "10.1", "10.3.2", "10", "11"},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			vers := makeVersions(test.versions)
			sort.Sort(SemverTagCollection(vers))
			actualVers := makeOriginal(vers)

			require.Equal(t, test.expectVersions, actualVers)
		})
	}
}

func TestTagCollectionUnique(t *testing.T) {
	tests := []struct {
		name           string
		versions       []string
		expectVersions []string
	}{
		{
			name:           "tagged versions",
			versions:       []string{"1.0.1", "1.0.2", "1.0.1-alpine", "1.0.1-debian"},
			expectVersions: []string{"1.0.1", "1.0.2"},
		},
		{
			name:           "tagged major version",
			versions:       []string{"4-alpine", "4"},
			expectVersions: []string{"4"},
		},
		{
			name:           "different major and minor versions",
			versions:       []string{"10.4", "9"},
			expectVersions: []string{"9", "10.4"},
		},
		{
			name:           "different major only versions",
			versions:       []string{"10", "11"},
			expectVersions: []string{"10", "11"},
		},
		{
			name:           "less specific patch version",
			versions:       []string{"0.1.0", "0.1"},
			expectVersions: []string{"0.1.0", "0.1"},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			vers := makeVersions(test.versions)
			uniqueVers, err := SemverTagCollection(vers).Unique()
			require.NoError(t, err)
			actualVers := makeOriginal(uniqueVers)

			require.Equal(t, test.expectVersions, actualVers)
		})
	}
}

func TestTrueVersionsBehind(t *testing.T) {
	tests := []struct {
		name           string
		current        string
		versions       []string
		expectVersions []string
	}{
		{
			name:           "tagged version",
			current:        "1.0.1-alpine",
			versions:       []string{"1.0.1", "1.0.2", "1.0.1-alpine", "1.0.1-debian"},
			expectVersions: []string{"1.0.1", "1.0.2"},
		},
		{
			name:           "major version only",
			current:        "10",
			versions:       []string{"10", "10.0", "10.1"},
			expectVersions: []string{"10"},
		},
		{
			name:           "minor version only",
			current:        "9.1",
			versions:       []string{"9.1.3", "9.1.2", "9.1", "9.1.0"},
			expectVersions: []string{"9.1"},
		},
		{
			name:           "variety",
			current:        "0.19",
			versions:       []string{"0", "0.17", "0.18", "0.19", "0.20"},
			expectVersions: []string{"0.19", "0.20", "0"},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			vers := makeVersions(test.versions)
			currentVer, err := semver.NewVersion(test.current)
			require.NoError(t, err)

			versionsBehind, err := SemverTagCollection(vers).VersionsBehind(currentVer)
			require.NoError(t, err)
			actualVers := makeOriginal(versionsBehind)

			require.Equal(t, test.expectVersions, actualVers)
		})
	}
}

func TestCompareVersions(t *testing.T) {
	tests := []struct {
		name     string
		versions []string
		expect   int
	}{
		{
			name:     "major versions match",
			versions: []string{"10", "10"},
			expect:   0,
		},
		{
			name:     "major version is less than",
			versions: []string{"9", "10"},
			expect:   -1,
		},
		{
			name:     "minor versions is less than",
			versions: []string{"10.1", "10"},
			expect:   -1,
		},
		{
			name:     "minor versions is greater than",
			versions: []string{"10.3", "10.1"},
			expect:   1,
		},
		{
			name:     "minor versions match",
			versions: []string{"10.1", "10.1"},
			expect:   0,
		},
		{
			name:     "patch versions is less than",
			versions: []string{"10.1.2", "10.1.3"},
			expect:   -1,
		},
		{
			name:     "patch versions is greater than",
			versions: []string{"10.1.4", "10.1.3"},
			expect:   1,
		},
		{
			name:     "patch versions match",
			versions: []string{"10.1.2", "10.1.2"},
			expect:   0,
		},
		{
			name:     "major version only with patch",
			versions: []string{"10", "10.1.2"},
			expect:   1,
		},
		{
			name:     "minor version greater, patch version less",
			versions: []string{"10.2.3", "10.1.4"},
			expect:   1,
		},
		{
			name:     "major version less, minor version greater",
			versions: []string{"9.1.2", "10.0.1"},
			expect:   -1,
		},
		{
			name:     "major version less, also shorter",
			versions: []string{"9.1", "10.2.2"},
			expect:   -1,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			vers := makeVersions(test.versions)

			actual := compareVersions(vers[0], vers[1])
			require.Equal(t, test.expect, actual)

			reverse := compareVersions(vers[1], vers[0])
			require.Equal(t, -test.expect, reverse)
		})
	}
}

func TestRemoveLeastSpecific(t *testing.T) {
	tests := []struct {
		name           string
		versions       []string
		expectVersions []string
	}{
		{
			name:           "major",
			versions:       []string{"1.0", "1.1", "1.2", "1"},
			expectVersions: []string{"1.0", "1.1", "1.2"},
		},
		{
			name:           "zeros",
			versions:       []string{"1.0", "1"},
			expectVersions: []string{"1.0"},
		},
		{
			name:           "minor",
			versions:       []string{"0.1.1", "0.1.2", "0.1"},
			expectVersions: []string{"0.1.1", "0.1.2"},
		},
		{
			name:           "patch",
			versions:       []string{"0.2.1", "0.2.2", "0.2.3"},
			expectVersions: []string{"0.2.1", "0.2.2", "0.2.3"},
		},
		{
			name:           "similar version numbers",
			versions:       []string{"0.11.1", "0.11", "11.0"},
			expectVersions: []string{"0.11.1", "11.0"},
		},
		{
			name:           "different versions",
			versions:       []string{"0.1", "2.1"},
			expectVersions: []string{"0.1", "2.1"},
		},
		{
			name:           "include last",
			versions:       []string{"0.1", "0.2.1", "0.3.4", "0.4"},
			expectVersions: []string{"0.1", "0.2.1", "0.3.4", "0.4"},
		},
		{
			name:           "variety",
			versions:       []string{"0.1.0", "0.1", "0.2.0", "0.2", "0.10.0", "0.10", "0.11.0", "0.11", "0.13.0", "0.13.1", "0.13.2", "0.13.3", "0.13", "0.17.0", "0.17.1", "0.17", "0.18.0", "0.18", "0.21.0", "0.21", "0"},
			expectVersions: []string{"0.1.0", "0.2.0", "0.10.0", "0.11.0", "0.13.0", "0.13.1", "0.13.2", "0.13.3", "0.17.0", "0.17.1", "0.18.0", "0.21.0"},
		},
		{
			name:           "preserve major version",
			versions:       []string{"0.1", "0.2.1", "1", "2", "3.5", "3", "4"},
			expectVersions: []string{"0.1", "0.2.1", "1", "2", "3.5", "4"},
		},
		{
			name:           "variations",
			versions:       []string{"1.0.0", "1.0", "1"},
			expectVersions: []string{"1.0.0"},
		},
		{
			name:           "variations 2",
			versions:       []string{"0.0.0", "0.0", "0"},
			expectVersions: []string{"0.0.0"},
		},
		{
			name:           "more segments",
			versions:       []string{"3.5.1.1", "3.5.1", "4.5.1"},
			expectVersions: []string{"3.5.1.1", "4.5.1"},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			semverTags := makeVersions(test.versions)
			collection := SemverTagCollection(semverTags)
			tags := collection.RemoveLeastSpecific()

			originalRemoved := makeOriginal(tags)
			require.Equal(t, test.expectVersions, originalRemoved)
		})
	}
}

func TestResolveTagDates(t *testing.T) {
	hostname := "index.docker.io"
	imageName := "library/postgres"
	versions := []string{"10.0", "10.1", "10.2"}
	testLogger := logger.TestLogger{T: t}
	allVersions := makeVersions(versions)

	reg, err := initRegistryClient(hostname)
	require.NoError(t, err)

	versionTags, err := resolveTagDates(testLogger, reg, imageName, allVersions)
	require.NoError(t, err)

	for _, versionTag := range versionTags {
		_, err = time.Parse(time.RFC3339, versionTag.Date)
		require.NoError(t, err)
	}
}
