package store

import (
	"context"
	"time"

	airgaptypes "github.com/replicatedhq/kots/kotsadm/pkg/airgap/types"
	apptypes "github.com/replicatedhq/kots/kotsadm/pkg/app/types"
	appstatustypes "github.com/replicatedhq/kots/kotsadm/pkg/appstatus/types"
	downstreamtypes "github.com/replicatedhq/kots/kotsadm/pkg/downstream/types"
	gitopstypes "github.com/replicatedhq/kots/kotsadm/pkg/gitops/types"
	installationtypes "github.com/replicatedhq/kots/kotsadm/pkg/online/types"
	preflighttypes "github.com/replicatedhq/kots/kotsadm/pkg/preflight/types"
	registrytypes "github.com/replicatedhq/kots/kotsadm/pkg/registry/types"
	sessiontypes "github.com/replicatedhq/kots/kotsadm/pkg/session/types"
	snapshottypes "github.com/replicatedhq/kots/kotsadm/pkg/snapshot/types"
	supportbundletypes "github.com/replicatedhq/kots/kotsadm/pkg/supportbundle/types"
	usertypes "github.com/replicatedhq/kots/kotsadm/pkg/user/types"
	versiontypes "github.com/replicatedhq/kots/kotsadm/pkg/version/types"
	kotsv1beta1 "github.com/replicatedhq/kots/kotskinds/apis/kots/v1beta1"
	"github.com/replicatedhq/kots/pkg/kotsutil"
	troubleshootredact "github.com/replicatedhq/troubleshoot/pkg/redact"
)

type KOTSStore interface {
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
	VersionStore
	LicenseStore
	ClusterStore
	SnapshotStore
	InstallationStore

	Init() error // this may need options
	WaitForReady(ctx context.Context) error
	IsNotFound(err error) bool
}

type Migrations interface {
	RunMigrations()
}

type RegistryStore interface {
	GetRegistryDetailsForApp(appID string) (*registrytypes.RegistrySettings, error)
	UpdateRegistry(appID string, hostname string, username string, password string, namespace string) error
}

type SupportBundleStore interface {
	ListSupportBundles(appID string) ([]*supportbundletypes.SupportBundle, error)
	ListPendingSupportBundlesForApp(appID string) ([]*supportbundletypes.PendingSupportBundle, error)
	GetSupportBundleFromSlug(slug string) (*supportbundletypes.SupportBundle, error)
	GetSupportBundle(bundleID string) (*supportbundletypes.SupportBundle, error)
	CreatePendingSupportBundle(bundleID string, appID string, clusterID string) error
	CreateSupportBundle(bundleID string, appID string, archivePath string, marshalledTree []byte) (*supportbundletypes.SupportBundle, error)
	GetSupportBundleArchive(bundleID string) (archivePath string, err error)
	GetSupportBundleAnalysis(bundleID string) (*supportbundletypes.SupportBundleAnalysis, error)
	SetSupportBundleAnalysis(bundleID string, insights []byte) error
	GetRedactions(bundleID string) (troubleshootredact.RedactionList, error)
	SetRedactions(bundleID string, redacts troubleshootredact.RedactionList) error
	GetSupportBundleSpecForApp(id string) (spec string, err error)
}

type PreflightStore interface {
	SetPreflightResults(appID string, sequence int64, results []byte) error
	GetPreflightResults(appID string, sequence int64) (*preflighttypes.PreflightResult, error)
	GetLatestPreflightResultsForSequenceZero() (*preflighttypes.PreflightResult, error)
	ResetPreflightResults(appID string, sequence int64) error
	SetIgnorePreflightPermissionErrors(appID string, sequence int64) error
}

type PrometheusStore interface {
	GetPrometheusAddress() (address string, err error)
	SetPrometheusAddress(address string) error
}

type AirgapStore interface {
	GetPendingAirgapUploadApp() (*airgaptypes.PendingApp, error)
	GetAirgapInstallStatus() (*airgaptypes.InstallStatus, error)
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
	CreateSession(user *usertypes.User, expiresAt *time.Time, roles []string) (*sessiontypes.Session, error)
	DeleteSession(sessionID string) error
	GetSession(sessionID string) (*sessiontypes.Session, error)
}

type AppStatusStore interface {
	GetAppStatus(appID string) (*appstatustypes.AppStatus, error)
}

type AppStore interface {
	AddAppToAllDownstreams(appID string) error
	SetAppInstallState(appID string, state string) error
	ListInstalledApps() ([]*apptypes.App, error)
	GetAppIDFromSlug(slug string) (appID string, err error)
	GetApp(appID string) (*apptypes.App, error)
	GetAppFromSlug(slug string) (*apptypes.App, error)
	CreateApp(name string, upstreamURI string, licenseData string, isAirgapEnabled bool, skipImagePush bool) (*apptypes.App, error)
	ListDownstreamsForApp(appID string) ([]downstreamtypes.Downstream, error)
	ListAppsForDownstream(clusterID string) ([]*apptypes.App, error)
	GetDownstream(clusterID string) (*downstreamtypes.Downstream, error)
	IsGitOpsEnabledForApp(appID string) (bool, error)
	SetUpdateCheckerSpec(appID string, updateCheckerSpec string) error
	SetSnapshotTTL(appID string, snapshotTTL string) error
	SetSnapshotSchedule(appID string, snapshotSchedule string) error
	RemoveApp(appID string) error
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
	IsGitOpsSupportedForVersion(appID string, sequence int64) (bool, error)
	IsRollbackSupportedForVersion(appID string, sequence int64) (bool, error)
	IsSnapshotsSupportedForVersion(a *apptypes.App, sequence int64) (bool, error)
	GetAppVersionArchive(appID string, sequence int64, dstPath string) error
	CreateAppVersionArchive(appID string, sequence int64, archivePath string) error
	CreateAppVersion(appID string, currentSequence *int64, appName string, appIcon string, kotsKinds *kotsutil.KotsKinds, filesInDir string, gitops gitopstypes.DownstreamGitOps, source string) (int64, error)
	GetAppVersion(string, int64) (*versiontypes.AppVersion, error)
	GetAppVersionsAfter(string, int64) ([]*versiontypes.AppVersion, error)
}

type LicenseStore interface {
	GetInitialLicenseForApp(appID string) (*kotsv1beta1.License, error)
	GetLatestLicenseForApp(appID string) (*kotsv1beta1.License, error)
	GetLicenseForAppVersion(appID string, sequence int64) (*kotsv1beta1.License, error)
	GetAllAppLicenses() ([]*kotsv1beta1.License, error)
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
