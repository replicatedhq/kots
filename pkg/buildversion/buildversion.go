package buildversion

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"runtime"
	"time"

	semver "github.com/Masterminds/semver/v3"
	"github.com/pkg/errors"
)

var (
	build Build
)

// Build holds details about this build of the Ship binary
type Build struct {
	Version      string     `json:"version,omitempty"`
	GitSHA       string     `json:"git,omitempty"`
	BuildTime    time.Time  `json:"buildTime,omitempty"`
	TimeFallback string     `json:"buildTimeFallback,omitempty"`
	GoInfo       GoInfo     `json:"go,omitempty"`
	RunAt        *time.Time `json:"runAt,omitempty"`
}

type GoInfo struct {
	Version  string `json:"version,omitempty"`
	Compiler string `json:"compiler,omitempty"`
	OS       string `json:"os,omitempty"`
	Arch     string `json:"arch,omitempty"`
}

// initBuild sets up the version info from build args
func initBuild() {
	build.Version = version
	if len(gitSHA) >= 7 {
		build.GitSHA = gitSHA[:7]
	}
	var err error
	build.BuildTime, err = time.Parse(time.RFC3339, buildTime)
	if err != nil {
		build.TimeFallback = buildTime
	}

	build.GoInfo = getGoInfo()
	build.RunAt = &RunAt
}

// GetBuild gets the build
func GetBuild() Build {
	return build
}

// Version gets the version
func Version() string {
	if build.Version == "" {
		return "v0.0.0-unknown"
	}
	return build.Version
}

// GitSHA gets the gitsha
func GitSHA() string {
	return build.GitSHA
}

// BuildTime gets the build time
func BuildTime() time.Time {
	return build.BuildTime
}

func getGoInfo() GoInfo {
	return GoInfo{
		Version:  runtime.Version(),
		Compiler: runtime.Compiler,
		OS:       runtime.GOOS,
		Arch:     runtime.GOARCH,
	}
}

func GetUserAgent() string {
	return fmt.Sprintf("KOTS/%s", Version())
}

// IsLatestRelease queries github for the latest release in the project repo. If that release has a semver greater
// than the current release, it returns false and the new latest release semver. Otherwise, it returns true or error
func IsLatestRelease() (bool, string, error) {
	fastClient := &http.Client{
		Timeout: time.Second * 1,
	}
	return isLatestRelease(fastClient, "https://kots.io")
}

func isLatestRelease(client *http.Client, upstream string) (bool, string, error) {
	resp, err := client.Get(upstream + "/install?version")
	if err != nil {
		return false, "", errors.Wrapf(err, "find latest release")
	}
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return false, "", errors.Wrapf(err, "read latest release body")
	}

	currentSemver, err := semver.NewVersion(Version())
	if err != nil {
		return false, "", errors.Wrapf(err, "current release %s does not parse as semver", Version())
	}

	latestSemver, err := semver.NewVersion(string(body))
	if err != nil {
		return false, "", errors.Wrapf(err, "latest release %s does not parse as semver", string(body))
	}

	if currentSemver.LessThan(latestSemver) {
		return false, string(body), nil
	}

	return true, "", nil
}
