package store

import (
	"context"
	"time"

	kotsv1beta1 "github.com/replicatedhq/kots/kotskinds/apis/kots/v1beta1"
	airgaptypes "github.com/replicatedhq/kots/pkg/airgap/types"
	appstatustypes "github.com/replicatedhq/kots/pkg/api/appstatus/types"
	downstreamtypes "github.com/replicatedhq/kots/pkg/api/downstream/types"
	versiontypes "github.com/replicatedhq/kots/pkg/api/version/types"
	apptypes "github.com/replicatedhq/kots/pkg/app/types"
	gitopstypes "github.com/replicatedhq/kots/pkg/gitops/types"
	snapshottypes "github.com/replicatedhq/kots/pkg/kotsadmsnapshot/types"
	installationtypes "github.com/replicatedhq/kots/pkg/online/types"
	preflighttypes "github.com/replicatedhq/kots/pkg/preflight/types"
	registrytypes "github.com/replicatedhq/kots/pkg/registry/types"
	rendertypes "github.com/replicatedhq/kots/pkg/render/types"
	sessiontypes "github.com/replicatedhq/kots/pkg/session/types"
	"github.com/replicatedhq/kots/pkg/store/types"
	supportbundletypes "github.com/replicatedhq/kots/pkg/supportbundle/types"
	usertypes "github.com/replicatedhq/kots/pkg/user/types"
	troubleshootredact "github.com/replicatedhq/troubleshoot/pkg/redact"
)

type Store interface {
	Migrations
	RegistryStore
	SupportBundleStore
	PreflightStore
	PrometheusStore
	AirgapStore
	TaskStore
	SessionStore
	AppStatusStore
	AppStore
	DownstreamStore
	VersionStore
	LicenseStore
	ClusterStore
	SnapshotStore
	InstallationStore
	KotsadmParamsStore

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

type TaskStore interface {
	SetTaskStatus(taskID string, message string, status string) error
	UpdateTaskStatusTimestamp(taskID string) error
	ClearTaskStatus(taskID string) error
	GetTaskStatus(taskID string) (status string, message string, err error)
}

type SessionStore interface {
	CreateSession(user *usertypes.User, issuedAt time.Time, expiresAt time.Time, roles []string) (*sessiontypes.Session, error)
	DeleteSession(sessionID string) error
	GetSession(sessionID string) (*sessiontypes.Session, error)
}

type AppStatusStore interface {
	GetAppStatus(appID string) (*appstatustypes.AppStatus, error)
	SetAppStatus(appID string, resourceStates []appstatustypes.ResourceState, updatedAt time.Time, sequence int64) error
}

type AppStore interface {
	AddAppToAllDownstreams(appID string) error
	SetAppInstallState(appID string, state string) error
	ListInstalledApps() ([]*apptypes.App, error)
	ListInstalledAppSlugs() ([]string, error)
	GetAppIDFromSlug(slug string) (appID string, err error)
	GetApp(appID string) (*apptypes.App, error)
	GetAppFromSlug(slug string) (*apptypes.App, error)
	CreateApp(name string, upstreamURI string, licenseData string, isAirgapEnabled bool, skipImagePush bool, registryIsReadOnly bool) (*apptypes.App, error)
	ListDownstreamsForApp(appID string) ([]downstreamtypes.Downstream, error)
	ListAppsForDownstream(clusterID string) ([]*apptypes.App, error)
	GetDownstream(clusterID string) (*downstreamtypes.Downstream, error)
	IsGitOpsEnabledForApp(appID string) (bool, error)
	SetUpdateCheckerSpec(appID string, updateCheckerSpec string) error
	SetSnapshotTTL(appID string, snapshotTTL string) error
	SetSnapshotSchedule(appID string, snapshotSchedule string) error
	RemoveApp(appID string) error
}

type DownstreamStore interface {
	GetCurrentSequence(appID string, clusterID string) (int64, error)
	GetCurrentParentSequence(appID string, clusterID string) (int64, error)
	GetParentSequenceForSequence(appID string, clusterID string, sequence int64) (int64, error)
	GetPreviouslyDeployedSequence(appID string, clusterID string) (int64, error)
	SetDownstreamVersionReady(appID string, sequence int64) error
	SetDownstreamVersionPendingPreflight(appID string, sequence int64) error
	UpdateDownstreamVersionStatus(appID string, sequence int64, status string, statusInfo string) error
	GetDownstreamVersionStatus(appID string, sequence int64) (types.DownstreamVersionStatus, error)
	GetIgnoreRBACErrors(appID string, sequence int64) (bool, error)
	GetCurrentVersion(appID string, clusterID string) (*downstreamtypes.DownstreamVersion, error)
	GetStatusForVersion(appID string, clusterID string, sequence int64) (types.DownstreamVersionStatus, error)
	GetPendingVersions(appID string, clusterID string) ([]downstreamtypes.DownstreamVersion, error)
	GetPastVersions(appID string, clusterID string) ([]downstreamtypes.DownstreamVersion, error)
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
	GetAppVersionArchive(appID string, sequence int64, dstPath string) error
	CreateAppVersionArchive(appID string, sequence int64, archivePath string) error
	CreateAppVersion(appID string, currentSequence *int64, filesInDir string, source string, skipPreflights bool, gitops gitopstypes.DownstreamGitOps) (int64, error)
	GetAppVersion(appID string, sequence int64) (*versiontypes.AppVersion, error)
	GetAppVersionsAfter(appID string, sequence int64) ([]*versiontypes.AppVersion, error)
	UpdateAppVersionInstallationSpec(appID string, sequence int64, spec kotsv1beta1.Installation) error
}

type LicenseStore interface {
	GetLatestLicenseForApp(appID string) (*kotsv1beta1.License, error)
	GetLicenseForAppVersion(appID string, sequence int64) (*kotsv1beta1.License, error)
	GetAllAppLicenses() ([]*kotsv1beta1.License, error)

	// originalLicenseData is the data received from the replicated API that was never marshalled locally so all fields are intact
	UpdateAppLicense(appID string, sequence int64, archiveDir string, newLicense *kotsv1beta1.License, originalLicenseData string, failOnVersionCreate bool, gitops gitopstypes.DownstreamGitOps, renderer rendertypes.Renderer) (int64, error)
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
