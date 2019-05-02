package version

import "expvar"

func exportBuild(build Build) {
	buildVersion := expvar.NewString("build.version")
	buildVersion.Set(build.Version)

	buildGitSHA := expvar.NewString("build.git_sha")
	buildGitSHA.Set(build.GitSHA)

	buildBuildTime := expvar.NewString("build.time")
	buildBuildTime.Set(build.BuildTime.String())

	buildTimeFallback := expvar.NewString("build.time_fallback")
	buildTimeFallback.Set(build.TimeFallback)
}
