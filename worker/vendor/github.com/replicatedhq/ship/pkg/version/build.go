package version

// NOTE: these variables are injected at build time

var (
	version, gitSHA, buildTime string
	helm, kustomize, terraform string
)
