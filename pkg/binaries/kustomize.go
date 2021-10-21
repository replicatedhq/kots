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
	knownKustomizeVersions kustomizeVersions
)

// InitKustomize will discover kustomize versions from the environment and populate a list of known
// kustomize versions for later use.
func InitKustomize() (err error) {
	knownKustomizeVersions, err = discoverKustomizeVersions(os.DirFS("/"))
	if err != nil {
		return errors.Wrap(err, "discover kustomize versions")
	}
	logger.Infof("Found kustomize binary versions %s", knownKustomizeVersions)
	return nil
}

// GetKustomizePathForVersion gets the path to a known kustomize version that matches the provided
// semver or semver range.
func GetKustomizePathForVersion(userString string) (string, error) {
	for _, knownVersion := range knownKustomizeVersions {
		if knownVersion.Match(userString) {
			return knownVersion.Path, nil
		}
	}

	return "", errors.New("kustomize binary not found")
}

type kustomizeVersions []kustomizeFuzzyVersion

func (a kustomizeVersions) Len() int {
	return len(a)
}

func (a kustomizeVersions) Less(i, j int) bool {
	return a[i].Version.GT(a[j].Version) // sort in descending order
}

func (a kustomizeVersions) Swap(i, j int) {
	a[i], a[j] = a[j], a[i]
}

type kustomizeFuzzyVersion struct {
	Version semver.Version
	Path    string
}

func newKustomizeFuzzyVersion(major uint64, path string) kustomizeFuzzyVersion {
	return kustomizeFuzzyVersion{
		Version: semver.Version{
			Major: major,
			Minor: 0,
			Patch: 0,
		},
		Path: path,
	}
}

func (v kustomizeFuzzyVersion) String() string {
	if v.Version.Equals(semver.Version{}) {
		return ""
	}
	return fmt.Sprintf("%d", v.Version.Major)
}

func (v kustomizeFuzzyVersion) Match(userString string) bool {
	if v.Version.Equals(semver.Version{}) {
		return true // catch all
	}

	if userString == "" || userString == "latest" {
		return false
	}

	if userString == v.String() {
		return true
	}

	// ignore error here as this could be a semver range
	if exactVer, err := semver.Parse(userString); err == nil {
		// fuzzy match major, not minor and patch
		return exactVer.Major == v.Version.Major
	}

	rangeVer, err := semver.ParseRange(userString)
	if err != nil {
		return false
	}

	return rangeVer(v.Version)
}

func discoverKustomizeVersions(fileSystem fs.FS) ([]kustomizeFuzzyVersion, error) {
	// in the kots run workflow, binaries exist under {kotsdatadir}/binaries
	if persistence.IsSQlite() {
		version := newKustomizeFuzzyVersion(0, filepath.Join(os.Getenv("KOTS_DATA_DIR"), "binaries/kustomize"))
		return []kustomizeFuzzyVersion{version}, nil
	}

	if binDirPath := os.Getenv("KOTS_KUSTOMIZE_BIN_DIR"); binDirPath != "" {
		versions, err := discoverKustomizeVersionsFromDir(fileSystem, binDirPath)
		return versions, errors.Wrap(err, "discover kustomize versions from dir")
	}

	versions, err := discoverKustomizeVersionsFromPath(fileSystem)
	return versions, errors.Wrap(err, "discover kustomize versions from path")
}

var kustomizeVerRegexp = regexp.MustCompile(`^kustomize(\d+)?$`)

func discoverKustomizeVersionsFromDir(fileSystem fs.FS, dirPath string) ([]kustomizeFuzzyVersion, error) {
	versions := []kustomizeFuzzyVersion{}

	entries, err := fs.ReadDir(fileSystem, strings.TrimLeft(dirPath, "/"))
	if err != nil {
		return versions, errors.Wrap(err, "read dir")
	}

	for _, entry := range entries {
		matches := kustomizeVerRegexp.FindStringSubmatch(entry.Name())
		if len(matches) > 0 {
			var major uint64
			// default version has no version suffix
			if matches[1] != "" {
				var err error
				major, err = strconv.ParseUint(matches[1], 10, 64)
				if err != nil {
					continue
				}
			}
			versions = append(versions, newKustomizeFuzzyVersion(major, filepath.Join(dirPath, entry.Name())))
		}
	}

	sort.Sort(kustomizeVersions(versions))

	return versions, nil
}

func discoverKustomizeVersionsFromPath(fileSystem fs.FS) ([]kustomizeFuzzyVersion, error) {
	// NOTE: exec.LookPath does not yet support io/fs
	binPath, err := exec.LookPath("kustomize")
	if err != nil {
		return nil, errors.Wrap(err, "look path")
	}

	return []kustomizeFuzzyVersion{newKustomizeFuzzyVersion(0, binPath)}, nil
}
