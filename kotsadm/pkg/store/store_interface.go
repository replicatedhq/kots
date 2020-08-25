package store

import (
	"time"

	airgaptypes "github.com/replicatedhq/kots/kotsadm/pkg/airgap/types"
	apptypes "github.com/replicatedhq/kots/kotsadm/pkg/app/types"
	appstatustypes "github.com/replicatedhq/kots/kotsadm/pkg/appstatus/types"
	downstreamtypes "github.com/replicatedhq/kots/kotsadm/pkg/downstream/types"
	preflighttypes "github.com/replicatedhq/kots/kotsadm/pkg/preflight/types"
	registrytypes "github.com/replicatedhq/kots/kotsadm/pkg/registry/types"
	sessiontypes "github.com/replicatedhq/kots/kotsadm/pkg/session/types"
	supportbundletypes "github.com/replicatedhq/kots/kotsadm/pkg/supportbundle/types"
	usertypes "github.com/replicatedhq/kots/kotsadm/pkg/user/types"
	kotsv1beta1 "github.com/replicatedhq/kots/kotskinds/apis/kots/v1beta1"
	troubleshootredact "github.com/replicatedhq/troubleshoot/pkg/redact"
)

type KOTSStore interface {
	RegistryStore
	SupportBundleStore
	PreflightStore
	PrometheusStore
	AirgapStore
	TaskStore
	SessionStore
	AppStatusStore
	AppStore
	LicenseStore
	ClusterStore
	SnapshotStore

	IsNotFound(err error) bool
}

type RegistryStore interface {
	GetRegistryDetailsForApp(string) (*registrytypes.RegistrySettings, error)
	UpdateRegistry(string, string, string, string, string) error
}

type SupportBundleStore interface {
	ListSupportBundles(string) ([]*supportbundletypes.SupportBundle, error)
	GetSupportBundleFromSlug(string) (*supportbundletypes.SupportBundle, error)
	GetSupportBundle(id string) (*supportbundletypes.SupportBundle, error)
	CreatePendingSupportBundle(string, string, string) error
	CreateSupportBundle(string, string, string, []byte) (*supportbundletypes.SupportBundle, error)
	GetSupportBundleArchive(string) (string, error)
	GetSupportBundleAnalysis(string) (*supportbundletypes.SupportBundleAnalysis, error)
	SetSupportBundleAnalysis(string, []byte) error
	GetRedactions(string) (troubleshootredact.RedactionList, error)
	SetRedactions(string, troubleshootredact.RedactionList) error
	GetSupportBundleSpecForApp(string) (string, error)
}

type PreflightStore interface {
	SetPreflightResults(string, int64, []byte) error
	GetPreflightResults(string, int64) (*preflighttypes.PreflightResult, error)
	GetLatestPreflightResults() (*preflighttypes.PreflightResult, error)
	ResetPreflightResults(string, int64) error
	SetIgnorePreflightPermissionErrors(string, int64) error
}

type PrometheusStore interface {
	GetPrometheusAddress() (string, error)
	SetPrometheusAddress(string) error
}

type AirgapStore interface {
	GetPendingAirgapUploadApp() (*airgaptypes.PendingApp, error)
	GetAirgapInstallStatus() (*airgaptypes.InstallStatus, error)
	ResetAirgapInstallInProgress(string) error
	SetAppIsAirgap(string) error
}

type TaskStore interface {
	SetTaskStatus(string, string, string) error
	UpdateTaskStatusTimestamp(string) error
	ClearTaskStatus(string) error
	GetTaskStatus(string) (string, string, error)
}

type SessionStore interface {
	CreateSession(*usertypes.User) (*sessiontypes.Session, error)
	DeleteSession(string) error
	GetSession(id string) (*sessiontypes.Session, error)
}

type AppStatusStore interface {
	GetAppStatus(string) (*appstatustypes.AppStatus, error)
}

type AppStore interface {
	AddAppToAllDownstreams(string) error
	SetAppInstallState(string, string) error
	ListInstalledApps() ([]*apptypes.App, error)
	GetAppIDFromSlug(string) (string, error)
	GetApp(string) (*apptypes.App, error)
	GetAppFromSlug(string) (*apptypes.App, error)
	CreateApp(string, string, string, bool) (*apptypes.App, error)
	ListDownstreamsForApp(string) ([]downstreamtypes.Downstream, error)
	ListAppsForDownstream(string) ([]*apptypes.App, error)
	GetDownstream(string) (*downstreamtypes.Downstream, error)
	IsGitOpsEnabledForApp(string) (bool, error)
	SetUpdateCheckerSpec(string, string) error
	SetSnapshotTTL(string, string) error
	SetSnapshotSchedule(string, string) error
}

type SnapshotStore interface {
	DeletePendingScheduledSnapshots(string) error
	CreateScheduledSnapshot(string, string, time.Time) error
}

type LicenseStore interface {
	GetLicenseForApp(string) (*kotsv1beta1.License, error)
}

type ClusterStore interface {
	ListClusters() (map[string]string, error)
}
