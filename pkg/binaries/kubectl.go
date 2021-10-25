package binaries

import (
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/blang/semver"
	"github.com/pkg/errors"
	"github.com/replicatedhq/kots/pkg/logger"
	"github.com/replicatedhq/kots/pkg/persistence"
)

var (
	knownKubectlVersions kubectlVersions
)

// InitKubectl will discover kubectl versions from the environment and populate a list of known
// kubectl versions for later use.
func InitKubectl() (err error) {
	knownKubectlVersions, err = discoverKubectlVersions(os.DirFS("/"))
	if err != nil {
		return errors.Wrap(err, "discover kubectl versions")
	}
	logger.Infof("Found kubectl binary versions %s", knownKubectlVersions)
	return nil
}

// GetKubectlPathForVersion gets the path to a known kubectl version that matches the provided
// semver or semver range.
func GetKubectlPathForVersion(userString string) (string, error) {
	for _, knownVersion := range knownKubectlVersions {
		if knownVersion.Match(userString) {
			return knownVersion.Path, nil
		}
	}

	return "", errors.New("kubectl binary not found")
}

type kubectlVersions []kubectlFuzzyVersion

func (a kubectlVersions) Len() int {
	return len(a)
}

func (a kubectlVersions) Less(i, j int) bool {
	return a[i].Version.GT(a[j].Version) // sort in descending order
}

func (a kubectlVersions) Swap(i, j int) {
	a[i], a[j] = a[j], a[i]
}

type kubectlFuzzyVersion struct {
	Version semver.Version
	Path    string
}

func newKubectlFuzzyVersion(major, minor uint64, path string) kubectlFuzzyVersion {
	return kubectlFuzzyVersion{
		Version: semver.Version{
			Major: major,
			Minor: minor,
			Patch: 0,
		},
		Path: path,
	}
}

func (v kubectlFuzzyVersion) String() string {
	if v.Version.Equals(semver.Version{}) {
		return ""
	}
	return fmt.Sprintf("v%d.%d", v.Version.Major, v.Version.Minor)
}

func (v kubectlFuzzyVersion) Match(userString string) bool {
	if v.Version.Equals(semver.Version{}) {
		return true // catch all
	}

	if userString == "" || userString == "latest" {
		return false
	}

	// match non-semver format 1.17 or v1.17
	if userString == v.String() || userString == strings.TrimLeft(v.String(), "v") {
		return true
	}

	// ignore error here as this could be a semver range
	if exactVer, err := semver.Parse(userString); err == nil {
		// fuzzy match major and minor, not patch
		return exactVer.Major == v.Version.Major && exactVer.Minor == v.Version.Minor
	}

	rangeVer, err := semver.ParseRange(userString)
	if err != nil {
		return false
	}

	return rangeVer(v.Version)
}

func discoverKubectlVersions(fileSystem fs.FS) ([]kubectlFuzzyVersion, error) {
	// in the kots run workflow, binaries exist under {kotsdatadir}/binaries
	if persistence.IsSQlite() {
		version := newKubectlFuzzyVersion(0, 0, filepath.Join(os.Getenv("KOTS_DATA_DIR"), "binaries/kubectl"))
		return []kubectlFuzzyVersion{version}, nil
	}

	if binDirPath := os.Getenv("KOTS_KUBECTL_BIN_DIR"); binDirPath != "" {
		versions, err := discoverKubectlVersionsFromDir(fileSystem, binDirPath)
		return versions, errors.Wrap(err, "discover kubectl versions from dir")
	}

	versions, err := discoverKubectlVersionsFromPath(fileSystem)
	return versions, errors.Wrap(err, "discover kubectl versions from path")
}

var kubectlVerRegexp = regexp.MustCompile(`^kubectl(-v(\d+)\.(\d+))?$`)

func discoverKubectlVersionsFromDir(fileSystem fs.FS, dirPath string) ([]kubectlFuzzyVersion, error) {
	versions := []kubectlFuzzyVersion{}

	entries, err := fs.ReadDir(fileSystem, strings.TrimLeft(dirPath, "/"))
	if err != nil {
		return versions, errors.Wrap(err, "read dir")
	}

	for _, entry := range entries {
		matches := kubectlVerRegexp.FindStringSubmatch(entry.Name())
		if len(matches) > 0 {
			var major, minor uint64
			// default version has no version suffix
			if matches[2] != "" && matches[3] != "" {
				var err error
				major, err = strconv.ParseUint(matches[2], 10, 64)
				if err != nil {
					continue
				}
				minor, err = strconv.ParseUint(matches[3], 10, 64)
				if err != nil {
					continue
				}
			}
			versions = append(versions, newKubectlFuzzyVersion(major, minor, filepath.Join(dirPath, entry.Name())))
		}
	}

	sort.Sort(kubectlVersions(versions))

	return versions, nil
}

func discoverKubectlVersionsFromPath(fileSystem fs.FS) ([]kubectlFuzzyVersion, error) {
	// NOTE: exec.LookPath does not yet support io/fs
	binPath, err := exec.LookPath("kubectl")
	if err != nil {
		return nil, errors.Wrap(err, "look path")
	}

	return []kubectlFuzzyVersion{newKubectlFuzzyVersion(0, 0, binPath)}, nil
}
