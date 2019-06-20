package constants

import "path"

const (
	// InstallerPrefixPath is the path prefix of installed assets
	InstallerPrefixPath = "installer"
	// KustomizeBasePath is the path to which assets to be kustomized are written
	KustomizeBasePath = "base"
	// GithubAssetSavePath is the path that github assets are initially fetched to
	GithubAssetSavePath = "tmp-github-asset"
)

func init() {
	SetShipRootDir(".ship")
}

var (
	// ShipPathInternal is the default folder path of Ship configuration
	ShipPathInternal string
	// ShipPathInternalTmp is a temporary folder that will get cleaned up on exit
	ShipPathInternalTmp string
	// ShipPathInternalLog is a log file that will be preserved on failure for troubleshooting
	ShipPathInternalLog string
	// InternalTempHelmHome is the path to a helm home directory
	InternalTempHelmHome string
	// StatePath is the default state file path
	StatePath string
	// ReleasePath is the default place to write a pulled release to the filesystem
	ReleasePath string
	// TempHelmValuesPath is the folder path used to store the updated values.yaml
	TempHelmValuesPath string
	// TempApplyOverlayPath is the folder path used to apply patch
	TempApplyOverlayPath string
	// HelmChartPath is the path used to store Helm chart contents
	HelmChartPath string
	// HelmChartForkedPath is the path used to store Helm chart contents of the fork
	HelmChartForkedPath string
	// UnforkForkedBasePath is the path that unfork will save the forked in when unforking
	UnforkForkedBasePath string
	// HelmLocalDependencyPath is the local temp path that local dependencies are initially saved to
	HelmLocalDependencyPath string
)

func SetShipRootDir(dir string) {
	ShipPathInternal = dir
	ShipPathInternalTmp = path.Join(ShipPathInternal, "tmp")
	ShipPathInternalLog = path.Join(ShipPathInternal, "debug.log")
	InternalTempHelmHome = path.Join(ShipPathInternalTmp, ".helm")
	StatePath = path.Join(ShipPathInternal, "state.json")
	ReleasePath = path.Join(ShipPathInternal, "release.yml")
	TempHelmValuesPath = path.Join(HelmChartPath, "tmp")
	TempApplyOverlayPath = path.Join("overlays", "tmp-apply")
	HelmChartPath = path.Join(ShipPathInternalTmp, "chart")
	HelmChartForkedPath = path.Join(ShipPathInternalTmp, "chart-forked")
	UnforkForkedBasePath = path.Join(ShipPathInternalTmp, "fork", "base")
	HelmLocalDependencyPath = path.Join(ShipPathInternalTmp, "dependencies")
}
