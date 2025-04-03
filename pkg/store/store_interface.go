package store

import (
	"context"
	"time"

	embeddedclusterv1beta1 "github.com/replicatedhq/embedded-cluster/kinds/apis/v1beta1"
	airgaptypes "github.com/replicatedhq/kots/pkg/airgap/types"
	downstreamtypes "github.com/replicatedhq/kots/pkg/api/downstream/types"
	reportingtypes "github.com/replicatedhq/kots/pkg/api/reporting/types"
	versiontypes "github.com/replicatedhq/kots/pkg/api/version/types"
	apptypes "github.com/replicatedhq/kots/pkg/app/types"
	appstatetypes "github.com/replicatedhq/kots/pkg/appstate/types"
	snapshottypes "github.com/replicatedhq/kots/pkg/kotsadmsnapshot/types"
	installationtypes "github.com/replicatedhq/kots/pkg/online/types"
	preflighttypes "github.com/replicatedhq/kots/pkg/preflight/types"
	registrytypes "github.com/replicatedhq/kots/pkg/registry/types"
	rendertypes "github.com/replicatedhq/kots/pkg/render/types"
	sessiontypes "github.com/replicatedhq/kots/pkg/session/types"
	"github.com/replicatedhq/kots/pkg/store/types"
	supportbundletypes "github.com/replicatedhq/kots/pkg/supportbundle/types"
	upstreamtypes "github.com/replicatedhq/kots/pkg/upstream/types"
	usertypes "github.com/replicatedhq/kots/pkg/user/types"
	kotsv1beta1 "github.com/replicatedhq/kotskinds/apis/kots/v1beta1"
	troubleshootredact "github.com/replicatedhq/troubleshoot/pkg/redact"
)

type Store interface {
	Migrations
	RegistryStore
	SupportBundleStore
	PreflightStore
	PrometheusStore
	AirgapStore
	SessionStore
	AppStatusStore
	AppStore
	DownstreamStore
	VersionStore
	LicenseStore
	UserStore
	ClusterStore
	SnapshotStore
	InstallationStore
	KotsadmParamsStore
	EmbeddedStore
	BrandingStore
	EmbeddedClusterStore

	Init() error // this may need options
	WaitForReady(ctx context.Context) error
	IsNotFound(err error) bool
}

type Migrations interface {
	RunMigrations()
}

type RegistryStore interface {
	GetRegistryDetailsForApp(appID string) (registrytypes.RegistrySettings, error)
	UpdateRegistry(appID string, hostname string, username string, password string, namespace string, isReadOnly bool) error
	GetAppIDsFromRegistry(hostname string) ([]string, error)
}

type SupportBundleStore interface {
	ListSupportBundles(appID string) ([]*supportbundletypes.SupportBundle, error)
	GetSupportBundle(bundleID string) (*supportbundletypes.SupportBundle, error)
	DeleteSupportBundle(bundleID string, appID string) error
	CreateSupportBundle(bundleID string, appID string, archivePath string, marshalledTree []byte) (*supportbundletypes.SupportBundle, error)
	GetSupportBundleArchive(bundleID string) (archivePath string, err error)
	GetSupportBundleAnalysis(bundleID string) (*supportbundletypes.SupportBundleAnalysis, error)
	SetSupportBundleAnalysis(bundleID string, insights []byte) error
	GetRedactions(bundleID string) (troubleshootredact.RedactionList, error)
	SetRedactions(bundleID string, redacts troubleshootredact.RedactionList) error
	CreateInProgressSupportBundle(supportBundle *supportbundletypes.SupportBundle) error
	UpdateSupportBundle(bundle *supportbundletypes.SupportBundle) error
	UploadSupportBundle(bundleID string, archivePath string, marshalledTree []byte) error
}

type PreflightStore interface {
	SetPreflightProgress(appID string, sequence int64, progress string) error
	GetPreflightProgress(appID string, sequence int64) (string, error)
	SetPreflightResults(appID string, sequence int64, results []byte) error
	GetPreflightResults(appID string, sequence int64) (*preflighttypes.PreflightResult, error)
	ResetPreflightResults(appID string, sequence int64) error
	SetIgnorePreflightPermissionErrors(appID string, sequence int64) error
}

type PrometheusStore interface {
	GetPrometheusAddress() (address string, err error)
	SetPrometheusAddress(address string) error
}

type AirgapStore interface {
	GetPendingAirgapUploadApp() (*airgaptypes.PendingApp, error)
	GetAirgapInstallStatus(appID string) (*airgaptypes.InstallStatus, error)
	ResetAirgapInstallInProgress(appID string) error
	SetAppIsAirgap(appID string, isAirgap bool) error
}

type SessionStore interface {
	CreateSession(user *usertypes.User, issuedAt time.Time, expiresAt time.Time, roles []string) (*sessiontypes.Session, error)
	DeleteSession(sessionID string) error
	GetSession(sessionID string) (*sessiontypes.Session, error)
	UpdateSessionExpiresAt(sessionID string, expiresAt time.Time) error
	DeleteExpiredSessions() error
}

type AppStatusStore interface {
	GetAppStatus(appID string) (*appstatetypes.AppStatus, error)
	SetAppStatus(appID string, resourceStates appstatetypes.ResourceStates, updatedAt time.Time, sequence int64) error
}

type AppStore interface {
	AddAppToAllDownstreams(appID string) error
	SetAppInstallState(appID string, state string) error
	ListInstalledApps() ([]*apptypes.App, error)
	ListInstalledAppSlugs() ([]string, error)
	ListFailedApps() ([]*apptypes.App, error)
	GetAppIDFromSlug(slug string) (appID string, err error)
	GetApp(appID string) (*apptypes.App, error)
	GetAppFromSlug(slug string) (*apptypes.App, error)
	CreateApp(name string, channelID string, upstreamURI string, licenseData string, isAirgapEnabled bool, skipImagePush bool, registryIsReadOnly bool) (*apptypes.App, error)
	ListDownstreamsForApp(appID string) ([]downstreamtypes.Downstream, error)
	ListAppsForDownstream(clusterID string) ([]*apptypes.App, error)
	GetDownstream(clusterID string) (*downstreamtypes.Downstream, error)
	IsGitOpsEnabledForApp(appID string) (bool, error)
	SetUpdateCheckerSpec(appID string, updateCheckerSpec string) error
	SetAutoDeploy(appID string, autoDeploy apptypes.AutoDeploy) error
	SetSnapshotTTL(appID string, snapshotTTL string) error
	SetSnapshotSchedule(appID string, snapshotSchedule string) error
	RemoveApp(appID string) error
	SetAppChannelChanged(appID string, channelChanged bool) error
	SetAppSelectedChannelID(appID string, channelID string) error
}

type DownstreamStore interface {
	GetCurrentDownstreamSequence(appID string, clusterID string) (int64, error)
	GetCurrentParentSequence(appID string, clusterID string) (int64, error)
	GetParentSequenceForSequence(appID string, clusterID string, sequence int64) (int64, error)
	GetPreviouslyDeployedSequence(appID string, clusterID string) (int64, error)
	MarkAsCurrentDownstreamVersion(appID string, sequence int64) error
	SetDownstreamVersionStatus(appID string, sequence int64, status types.DownstreamVersionStatus, statusInfo string) error
	GetDownstreamVersionStatus(appID string, sequence int64) (types.DownstreamVersionStatus, error)
	GetDownstreamVersionSource(appID string, sequence int64) (string, error)
	GetIgnoreRBACErrors(appID string, sequence int64) (bool, error)
	GetCurrentDownstreamVersion(appID string, clusterID string) (*downstreamtypes.DownstreamVersion, error)
	GetStatusForVersion(appID string, clusterID string, sequence int64) (types.DownstreamVersionStatus, error)
	// GetDownstreamVersions returns a sorted list of app releases without additional details. The sort order is determined by semver being enabled in the license.
	GetDownstreamVersions(appID string, clusterID string, downloadedOnly bool) (*downstreamtypes.DownstreamVersions, error)
	// Same as GetDownstreamVersions, but finds a cluster where app is deployed
	FindDownstreamVersions(appID string, downloadedOnly bool) (*downstreamtypes.DownstreamVersions, error)
	GetDownstreamVersionHistory(appID string, clusterID string, currentPage int, pageSize int, pinLatest bool, pinLatestDeployable bool) (*downstreamtypes.DownstreamVersionHistory, error)
	AddDownstreamVersionDetails(appID string, clusterID string, version *downstreamtypes.DownstreamVersion, checkIfDeployable bool) error
	AddDownstreamVersionsDetails(appID string, clusterID string, versions []*downstreamtypes.DownstreamVersion, checkIfDeployable bool) error
	// GetLatestDeployableDownstreamVersion returns the latest allowed version to upgrade to from the currently deployed version
	GetLatestDeployableDownstreamVersion(appID string, clusterID string) (latestDeployableVersion *downstreamtypes.DownstreamVersion, numOfSkippedVersions int, numOfRemainingVersions int, finalError error)
	IsAppVersionDeployable(appID string, sequence int64) (bool, string, error)
	GetDownstreamOutput(appID string, clusterID string, sequence int64) (*downstreamtypes.DownstreamOutput, error)
	IsDownstreamDeploySuccessful(appID string, clusterID string, sequence int64) (bool, error)
	UpdateDownstreamDeployStatus(appID string, clusterID string, sequence int64, isError bool, output downstreamtypes.DownstreamOutput) error
	DeleteDownstreamDeployStatus(appID string, clusterID string, sequence int64) error
}

type SnapshotStore interface {
	ListPendingScheduledSnapshots(appID string) ([]snapshottypes.ScheduledSnapshot, error)
	UpdateScheduledSnapshot(snapshotID string, backupName string) error
	DeletePendingScheduledSnapshots(appID string) error
	CreateScheduledSnapshot(snapshotID string, appID string, timestamp time.Time) error

	ListPendingScheduledInstanceSnapshots(clusterID string) ([]snapshottypes.ScheduledInstanceSnapshot, error)
	UpdateScheduledInstanceSnapshot(snapshotID string, backupName string) error
	DeletePendingScheduledInstanceSnapshots(clusterID string) error
	CreateScheduledInstanceSnapshot(snapshotID string, clusterID string, timestamp time.Time) error
}

type VersionStore interface {
	IsIdentityServiceSupportedForVersion(appID string, sequence int64) (bool, error)
	IsRollbackSupportedForVersion(appID string, sequence int64) (bool, error)
	IsSnapshotsSupportedForVersion(a *apptypes.App, sequence int64, renderer rendertypes.Renderer) (bool, error)
	GetTargetKotsVersionForVersion(appID string, sequence int64) (string, error)
	GetAppVersionArchive(appID string, sequence int64, dstPath string) error
	GetAppVersionBaseSequence(appID string, versionLabel string) (int64, error)
	GetAppVersionBaseArchive(appID string, versionLabel string) (string, int64, error)
	CreatePendingDownloadAppVersion(appID string, update upstreamtypes.Update, kotsApplication *kotsv1beta1.Application, license *kotsv1beta1.License) (int64, error)
	UpdateAppVersion(appID string, sequence int64, baseSequence *int64, filesInDir string, source string, skipPreflights bool) error
	CreateAppVersion(appID string, baseSequence *int64, filesInDir string, source string, isInstall bool, isAutomated bool, configFile string, skipPreflights bool) (int64, error)
	GetAppVersion(appID string, sequence int64) (*versiontypes.AppVersion, error)
	GetLatestAppSequence(appID string, downloadedOnly bool) (int64, error)
	UpdateNextAppVersionDiffSummary(appID string, baseSequence int64) error
	GetNextAppSequence(appID string) (int64, error)
	GetCurrentUpdateCursor(appID string, channelID string) (string, error)
	HasStrictPreflights(appID string, sequence int64) (bool, error)
	GetEmbeddedClusterConfigForVersion(appID string, sequence int64) (*embeddedclusterv1beta1.Config, error)
}

type LicenseStore interface {
	GetLatestLicenseForApp(appID string) (*kotsv1beta1.License, error)
	GetLicenseForAppVersion(appID string, sequence int64) (*kotsv1beta1.License, error)
	GetAllAppLicenses() ([]*kotsv1beta1.License, error)

	// originalLicenseData is the data received from the replicated API that was never marshalled locally so all fields are intact
	UpdateAppLicense(appID string, sequence int64, archiveDir string, newLicense *kotsv1beta1.License, originalLicenseData string, channelChanged bool, failOnVersionCreate bool, renderer rendertypes.Renderer, reportingInfo *reportingtypes.ReportingInfo) (int64, error)
	UpdateAppLicenseSyncNow(appID string) error
}

type UserStore interface {
	GetSharedPasswordBcrypt() ([]byte, error)
	GetPasswordUpdatedAt() (*time.Time, error)
	FlagInvalidPassword() error
	FlagSuccessfulLogin() error
}

type ClusterStore interface {
	ListClusters() ([]*downstreamtypes.Downstream, error)
	GetClusterIDFromSlug(slug string) (clusterID string, err error)
	GetClusterIDFromDeployToken(deployToken string) (clusterID string, err error)
	CreateNewCluster(userID string, isAllUsers bool, title string, token string) (clusterID string, err error)
	SetInstanceSnapshotTTL(clusterID string, snapshotTTL string) error
	SetInstanceSnapshotSchedule(clusterID string, snapshotSchedule string) error
}

type InstallationStore interface {
	GetPendingInstallationStatus() (*installationtypes.InstallStatus, error)
}

type KotsadmParamsStore interface {
	IsKotsadmIDGenerated() (bool, error)
	SetIsKotsadmIDGenerated() error
}

type EmbeddedStore interface {
	GetEmbeddedClusterAuthToken() (string, error)
	SetEmbeddedClusterAuthToken(token string) error
}

type BrandingStore interface {
	GetInitialBranding() ([]byte, error)
	CreateInitialBranding(brandingArchive []byte) (string, error)
	GetLatestBranding() ([]byte, error)
	GetLatestBrandingForApp(appID string) ([]byte, error)
}

type EmbeddedClusterStore interface {
	SetEmbeddedClusterInstallCommandRoles(roles []string) (string, error)
	GetEmbeddedClusterInstallCommandRoles(token string) ([]string, error)
}
