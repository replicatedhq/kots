package version

import (
	"runtime"
	"time"
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
