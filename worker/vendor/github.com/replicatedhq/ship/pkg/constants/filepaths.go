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
	// DefaultOverlaysPath is the folder path used for the default k8s patches removing helm and tiller labels
	DefaultOverlaysPath string
	// TempApplyOverlayPath is the folder path used to apply patch
	TempApplyOverlayPath string
	// HelmChartPath is the path used to store Helm chart contents
	HelmChartPath string
	// HelmChartForkedPath is the path used to store Helm chart contents of the fork
	HelmChartForkedPath string
	// UnforkForkedBasePath is the path that unfork will save the forked in when unforking
	UnforkForkedBasePath string
	// HelmLocalDependencyPath is the local temp path that local dependencies are initially saved to
	HelmLocalDependencyPath = path.Join(ShipPathInternalTmp, "dependencies")
	// Kustomize render path is the local path that kustomize steps will use to render yaml for display
	KustomizeRenderPath string
	// Helm values path is the path in which the helm values file and original helm values file will be stored
	HelmValuesPath string
	// Upstream Contents path is the path in which the upstream is stored
	UpstreamContentsPath string
	// Upstream App Release path is the path in which the app release yaml will be stored
	UpstreamAppReleasePath string
)

func SetShipRootDir(dir string) {
	ShipPathInternal = dir
	ShipPathInternalTmp = path.Join(ShipPathInternal, "tmp")
	ShipPathInternalLog = path.Join(ShipPathInternal, "debug.log")
	InternalTempHelmHome = path.Join(ShipPathInternalTmp, ".helm")
	StatePath = path.Join(ShipPathInternal, "state.json")
	ReleasePath = path.Join(ShipPathInternal, "release.yml")
	TempHelmValuesPath = path.Join(HelmChartPath, "tmp")
	DefaultOverlaysPath = path.Join("overlays", "defaults")
	TempApplyOverlayPath = path.Join("overlays", "tmp-apply")
	HelmChartPath = path.Join(ShipPathInternalTmp, "chart")
	HelmChartForkedPath = path.Join(ShipPathInternalTmp, "chart-forked")
	UnforkForkedBasePath = path.Join(ShipPathInternalTmp, "fork", "base")
	HelmLocalDependencyPath = path.Join(ShipPathInternalTmp, "dependencies")
	KustomizeRenderPath = path.Join(ShipPathInternalTmp, "kustomize")
	HelmValuesPath = path.Join(ShipPathInternal, "helm")
	UpstreamContentsPath = path.Join(ShipPathInternal, "upstream")
	UpstreamAppReleasePath = path.Join(UpstreamContentsPath, "appRelease.json")
}
