package version

import "time"

var (
	build    Build
	hasBuilt = false
)

// Build holds details about this build of the Ship binary
type Build struct {
	Version      string            `json:"version,omitempty"`
	GitSHA       string            `json:"git,omitempty"`
	BuildTime    time.Time         `json:"buildTime,omitempty"`
	TimeFallback string            `json:"buildTimeFallback,omitempty"`
	Dependencies BuildDependencies `json:"dependencies,omitempty"`
}

type BuildDependencies struct {
	Helm      string `json:"helm,omitempty"`
	Kustomize string `json:"kustomize,omitempty"`
	Terraform string `json:"terraform,omitempty"`
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

	deps := BuildDependencies{
		Helm:      helm,
		Kustomize: kustomize,
		Terraform: terraform,
	}
	build.Dependencies = deps
	hasBuilt = true
}

// GetBuild gets the build
func GetBuild() Build {
	if !hasBuilt {
		Init()
	}
	return build
}

// Version gets the version
func Version() string {
	if !hasBuilt {
		Init()
	}
	return build.Version
}

// GitSHA gets the gitsha
func GitSHA() string {
	if !hasBuilt {
		Init()
	}
	return build.GitSHA
}

// BuildTime gets the build time
func BuildTime() time.Time {
	if !hasBuilt {
		Init()
	}
	return build.BuildTime
}
