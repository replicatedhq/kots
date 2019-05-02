package version

import (
	"time"
)

var (
	build Build
)

// Build holds details about this build of the Ship binary
type Build struct {
	Version      string
	GitSHA       string
	BuildTime    time.Time
	TimeFallback string `json:"time_fallback,omitempty"`
}

// Init sets up the version info from build args
func Init() {
	build.Version = version
	if len(gitSHA) >= 7 {
		build.GitSHA = gitSHA[:7]
	}
	var err error
	build.BuildTime, err = time.Parse(time.RFC3339, buildTime)
	if err != nil {
		build.TimeFallback = buildTime
	}

	exportBuild(build)
}

// GetBuild gets the build
func GetBuild() Build {
	return build
}

// Version gets the version
func Version() string {
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
