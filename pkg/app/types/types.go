package types

type UndeployStatus string

const (
	UndeployInProcess UndeployStatus = "in_process"
	UndeployCompleted UndeployStatus = "completed"
	UndeployFailed    UndeployStatus = "failed"
	UndeployReset     UndeployStatus = ""
)

type AutoDeploy string

const (
	AutoDeployDisabled              AutoDeploy = "disabled"
	AutoDeploySemverPatch           AutoDeploy = "semver-patch"
	AutoDeploySemverMinorPatch      AutoDeploy = "semver-minor-patch"
	AutoDeploySemverMajorMinorPatch AutoDeploy = "semver-major-minor-patch"
	AutoDeploySequence              AutoDeploy = "sequence"
)
