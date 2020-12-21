package handlers

import "net/http"

type KOTSHandler interface {
	Ping(w http.ResponseWriter, r *http.Request)

	UploadNewLicense(w http.ResponseWriter, r *http.Request)
	ExchangePlatformLicense(w http.ResponseWriter, r *http.Request)
	ResumeInstallOnline(w http.ResponseWriter, r *http.Request)
	GetOnlineInstallStatus(w http.ResponseWriter, r *http.Request)

	// Support Bundles
	GetSupportBundle(w http.ResponseWriter, r *http.Request) // TODO: appSlug
	ListSupportBundles(w http.ResponseWriter, r *http.Request)
	GetSupportBundleCommand(w http.ResponseWriter, r *http.Request)
	GetSupportBundleFiles(w http.ResponseWriter, r *http.Request)      // TODO: appSlug
	GetSupportBundleRedactions(w http.ResponseWriter, r *http.Request) // TODO: appSlug
	DownloadSupportBundle(w http.ResponseWriter, r *http.Request)      // TODO: appSlug
	CollectSupportBundle(w http.ResponseWriter, r *http.Request)

	// redactor routes
	UpdateRedact(w http.ResponseWriter, r *http.Request)
	GetRedact(w http.ResponseWriter, r *http.Request)
	ListRedactors(w http.ResponseWriter, r *http.Request)
	GetRedactMetadataAndYaml(w http.ResponseWriter, r *http.Request)
	SetRedactMetadataAndYaml(w http.ResponseWriter, r *http.Request)
	DeleteRedact(w http.ResponseWriter, r *http.Request)
	SetRedactEnabled(w http.ResponseWriter, r *http.Request)

	// Kotsadm Identity Service
	ConfigureIdentityService(w http.ResponseWriter, r *http.Request)
	GetIdentityServiceConfig(w http.ResponseWriter, r *http.Request)

	// App Identity Service
	ConfigureAppIdentityService(w http.ResponseWriter, r *http.Request)
	GetAppIdentityServiceConfig(w http.ResponseWriter, r *http.Request)

	// Apps
	ListApps(w http.ResponseWriter, r *http.Request)
	GetApp(w http.ResponseWriter, r *http.Request)
	GetAppStatus(w http.ResponseWriter, r *http.Request)
	GetAppVersionHistory(w http.ResponseWriter, r *http.Request)
	GetUpdateDownloadStatus(w http.ResponseWriter, r *http.Request) // NOTE: appSlug is unused

	// Airgap
	AirgapBundleProgress(w http.ResponseWriter, r *http.Request)
	AirgapBundleExists(w http.ResponseWriter, r *http.Request)
	CreateAppFromAirgap(w http.ResponseWriter, r *http.Request)
	UpdateAppFromAirgap(w http.ResponseWriter, r *http.Request)
	CheckAirgapBundleChunk(w http.ResponseWriter, r *http.Request)
	UploadAirgapBundleChunk(w http.ResponseWriter, r *http.Request)
	GetAirgapInstallStatus(w http.ResponseWriter, r *http.Request)
	ResetAirgapInstallStatus(w http.ResponseWriter, r *http.Request)

	// Implemented handlers
	IgnorePreflightRBACErrors(w http.ResponseWriter, r *http.Request)
	StartPreflightChecks(w http.ResponseWriter, r *http.Request)
	GetLatestPreflightResultsForSequenceZero(w http.ResponseWriter, r *http.Request)
	GetPreflightResult(w http.ResponseWriter, r *http.Request)
	GetPreflightCommand(w http.ResponseWriter, r *http.Request) // this is intentionally policy.AppRead

	DeployAppVersion(w http.ResponseWriter, r *http.Request)
	RedeployAppVersion(w http.ResponseWriter, r *http.Request)
	GetAppRenderedContents(w http.ResponseWriter, r *http.Request)
	GetAppContents(w http.ResponseWriter, r *http.Request)
	GetAppDashboard(w http.ResponseWriter, r *http.Request)
	GetDownstreamOutput(w http.ResponseWriter, r *http.Request)

	GetKotsadmRegistry(w http.ResponseWriter, r *http.Request)
	GetImageRewriteStatus(w http.ResponseWriter, r *http.Request)
	UpdateAppRegistry(w http.ResponseWriter, r *http.Request)
	GetAppRegistry(w http.ResponseWriter, r *http.Request)
	ValidateAppRegistry(w http.ResponseWriter, r *http.Request)

	UpdateAppConfig(w http.ResponseWriter, r *http.Request)
	CurrentAppConfig(w http.ResponseWriter, r *http.Request)
	LiveAppConfig(w http.ResponseWriter, r *http.Request)

	SyncLicense(w http.ResponseWriter, r *http.Request)
	GetLicense(w http.ResponseWriter, r *http.Request)

	AppUpdateCheck(w http.ResponseWriter, r *http.Request)
	UpdateCheckerSpec(w http.ResponseWriter, r *http.Request)
	RemoveApp(w http.ResponseWriter, r *http.Request)

	// App snapshot routes
	CreateApplicationBackup(w http.ResponseWriter, r *http.Request)
	GetRestoreStatus(w http.ResponseWriter, r *http.Request)
	CancelRestore(w http.ResponseWriter, r *http.Request)
	CreateApplicationRestore(w http.ResponseWriter, r *http.Request)
	GetRestoreDetails(w http.ResponseWriter, r *http.Request)
	ListBackups(w http.ResponseWriter, r *http.Request)
	GetSnapshotConfig(w http.ResponseWriter, r *http.Request)
	SaveSnapshotConfig(w http.ResponseWriter, r *http.Request)

	// Global snapshot routes
	ListInstanceBackups(w http.ResponseWriter, r *http.Request)
	CreateInstanceBackup(w http.ResponseWriter, r *http.Request)
	GetInstanceSnapshotConfig(w http.ResponseWriter, r *http.Request)
	SaveInstanceSnapshotConfig(w http.ResponseWriter, r *http.Request)
	GetGlobalSnapshotSettings(w http.ResponseWriter, r *http.Request)
	UpdateGlobalSnapshotSettings(w http.ResponseWriter, r *http.Request)
	GetBackup(w http.ResponseWriter, r *http.Request)
	DeleteBackup(w http.ResponseWriter, r *http.Request)
	RestoreApps(w http.ResponseWriter, r *http.Request)
	GetRestoreAppsStatus(w http.ResponseWriter, r *http.Request)
	DownloadSnapshotLogs(w http.ResponseWriter, r *http.Request)
	GetVeleroStatus(w http.ResponseWriter, r *http.Request)

	// KURL
	GenerateNodeJoinCommandWorker(w http.ResponseWriter, r *http.Request)
	GenerateNodeJoinCommandMaster(w http.ResponseWriter, r *http.Request)
	DrainNode(w http.ResponseWriter, r *http.Request)
	DeleteNode(w http.ResponseWriter, r *http.Request)
	GetKurlNodes(w http.ResponseWriter, r *http.Request)

	// Prometheus
	SetPrometheusAddress(w http.ResponseWriter, r *http.Request)

	// GitOps
	UpdateAppGitOps(w http.ResponseWriter, r *http.Request)
	DisableAppGitOps(w http.ResponseWriter, r *http.Request)
	InitGitOpsConnection(w http.ResponseWriter, r *http.Request)
	CreateGitOps(w http.ResponseWriter, r *http.Request)
	ResetGitOps(w http.ResponseWriter, r *http.Request)
	GetGitOpsRepo(w http.ResponseWriter, r *http.Request)
}
