package constants

import "path"

const (
	// InstallerPrefixPath is the path prefix of installed assets
	InstallerPrefixPath = "installer"
	// ShipPathInternal is the default folder path of Ship configuration
	ShipPathInternal = ".ship"
	// KustomizeBasePath is the path to which assets to be kustomized are written
	KustomizeBasePath = "base"
	// GithubAssetSavePath is the path that github assets are initially fetched to
	GithubAssetSavePath = "tmp-github-asset"
)

var (
	// ShipPathInternalTmp is a temporary folder that will get cleaned up on exit
	ShipPathInternalTmp = path.Join(ShipPathInternal, "tmp")
	// ShipPathInternalLog is a log file that will be preserved on failure for troubleshooting
	ShipPathInternalLog = path.Join(ShipPathInternal, "debug.log")
	// InternalTempHelmHome is the path to a helm home directory
	InternalTempHelmHome = path.Join(ShipPathInternalTmp, ".helm")
	// StatePath is the default state file path
	StatePath = path.Join(ShipPathInternal, "state.json")
	// ReleasePath is the default place to write a pulled release to the filesystem
	ReleasePath = path.Join(ShipPathInternal, "release.yml")
	// TempHelmValuesPath is the folder path used to store the updated values.yaml
	TempHelmValuesPath = path.Join(HelmChartPath, "tmp")
	// TempApplyOverlayPath is the folder path used to apply patch
	TempApplyOverlayPath = path.Join("overlays", "tmp-apply")
	// HelmChartPath is the path used to store Helm chart contents
	HelmChartPath = path.Join(ShipPathInternalTmp, "chart")
	// HelmChartForkedPath is the path used to store Helm chart contents of the fork
	HelmChartForkedPath = path.Join(ShipPathInternalTmp, "chart-forked")
	// UnforkForkedBasePath is the path that unfork will save the forked in when unforking
	UnforkForkedBasePath = path.Join(ShipPathInternalTmp, "fork", "base")
	// HelmLocalDependencyPath is the local temp path that local dependencies are initially saved to
	HelmLocalDependencyPath = path.Join(ShipPathInternalTmp, "dependencies")
)
