package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
	kotsscheme "github.com/replicatedhq/kots/kotskinds/client/kotsclientset/scheme"
	"github.com/replicatedhq/kots/pkg/logger"
	"github.com/replicatedhq/kots/pkg/policy"
	"github.com/replicatedhq/kots/pkg/store"
	troubleshootscheme "github.com/replicatedhq/troubleshoot/pkg/client/troubleshootclientset/scheme"
	yaml "github.com/replicatedhq/yaml/v3"
	veleroscheme "github.com/vmware-tanzu/velero/pkg/generated/clientset/versioned/scheme"
	"k8s.io/client-go/kubernetes/scheme"
)

var _ KOTSHandler = (*Handler)(nil)

type Handler struct {
}

func init() {
	kotsscheme.AddToScheme(scheme.Scheme)
	troubleshootscheme.AddToScheme(scheme.Scheme)
	veleroscheme.AddToScheme(scheme.Scheme)
}

func RegisterSessionAuthRoutes(r *mux.Router, kotsStore store.Store, handler KOTSHandler, middleware *policy.Middleware) {
	r.Use(LoggingMiddleware, RequireValidSessionMiddleware(kotsStore))

	// Installation
	r.Name("UploadNewLicense").Path("/api/v1/license").Methods("POST").
		HandlerFunc(middleware.EnforceAccess(policy.AppCreate, handler.UploadNewLicense))
	r.Name("ExchangePlatformLicense").Path("/api/v1/license/platform").Methods("POST").
		HandlerFunc(middleware.EnforceAccess(policy.AppCreate, handler.ExchangePlatformLicense))
	r.Name("ResumeInstallOnline").Path("/api/v1/license/resume").Methods("PUT").
		HandlerFunc(middleware.EnforceAccess(policy.AppCreate, handler.ResumeInstallOnline))
	r.Name("GetOnlineInstallStatus").Path("/api/v1/app/online/status").Methods("GET").
		HandlerFunc(middleware.EnforceAccess(policy.AppCreate, handler.GetOnlineInstallStatus))
	r.Name("CanInstallAppVersion").Path("/api/v1/app/{appSlug}/can-install").Methods("POST").
		HandlerFunc(middleware.EnforceAccess(policy.AppCreate, handler.CanInstallAppVersion))
	r.Name("GetAutomatedInstallStatus").Path("/api/v1/app/{appSlug}/automated/status").Methods("GET").
		HandlerFunc(middleware.EnforceAccess(policy.AppCreate, handler.GetAutomatedInstallStatus))

	// Support Bundles
	r.Name("GetSupportBundle").Path("/api/v1/troubleshoot/supportbundle/{bundleSlug}").Methods("GET").
		HandlerFunc(middleware.EnforceAccess(policy.AppSupportbundleRead, handler.GetSupportBundle)) // TODO: appSlug
	r.Name("ListSupportBundles").Path("/api/v1/troubleshoot/app/{appSlug}/supportbundles").Methods("GET").
		HandlerFunc(middleware.EnforceAccess(policy.AppSupportbundleRead, handler.ListSupportBundles))
	r.Name("GetSupportBundleCommand").Path("/api/v1/troubleshoot/app/{appSlug}/supportbundlecommand").Methods("POST").
		HandlerFunc(middleware.EnforceAccess(policy.AppSupportbundleRead, handler.GetSupportBundleCommand))
	r.Name("GetSupportBundleFiles").Path("/api/v1/troubleshoot/supportbundle/{bundleId}/files").Methods("GET").
		HandlerFunc(middleware.EnforceAccess(policy.AppSupportbundleRead, handler.GetSupportBundleFiles)) // TODO: appSlug
	r.Name("GetSupportBundleRedactions").Path("/api/v1/troubleshoot/supportbundle/{bundleId}/redactions").Methods("GET").
		HandlerFunc(middleware.EnforceAccess(policy.AppSupportbundleRead, handler.GetSupportBundleRedactions)) // TODO: appSlug
	r.Name("DownloadSupportBundle").Path("/api/v1/troubleshoot/supportbundle/{bundleId}/download").Methods("GET").
		HandlerFunc(middleware.EnforceAccess(policy.AppSupportbundleRead, handler.DownloadSupportBundle)) // TODO: appSlug
	r.Name("CollectSupportBundle").Path("/api/v1/troubleshoot/supportbundle/app/{appId}/cluster/{clusterId}/collect").Methods("POST").
		HandlerFunc(middleware.EnforceAccess(policy.AppSupportbundleWrite, handler.CollectSupportBundle))
	r.Name("CollectHelmSupportBundle").Path("/api/v1/troubleshoot/supportbundle/app/{appSlug}/collect").Methods("POST").
		HandlerFunc(middleware.EnforceAccess(policy.AppSupportbundleWrite, handler.CollectHelmSupportBundle))
	r.Name("ShareSupportBundle").Path("/api/v1/troubleshoot/app/{appSlug}/supportbundle/{bundleId}/share").Methods("POST").
		HandlerFunc(middleware.EnforceAccess(policy.AppSupportbundleWrite, handler.ShareSupportBundle))
	r.Name("GetPodDetailsFromSupportBundle").Path("/api/v1/troubleshoot/app/{appSlug}/supportbundle/{bundleId}/pod").Methods("GET").
		HandlerFunc(middleware.EnforceAccess(policy.AppSupportbundleRead, handler.GetPodDetailsFromSupportBundle))

	// redactor routes
	r.Name("UpdateRedact").Path("/api/v1/redact/set").Methods("PUT").
		HandlerFunc(middleware.EnforceAccess(policy.RedactorWrite, handler.UpdateRedact))
	r.Name("GetRedact").Path("/api/v1/redact/get").Methods("GET").
		HandlerFunc(middleware.EnforceAccess(policy.RedactorRead, handler.GetRedact))
	r.Name("ListRedactors").Path("/api/v1/redacts").Methods("GET").
		HandlerFunc(middleware.EnforceAccess(policy.RedactorRead, handler.ListRedactors))
	r.Name("GetRedactMetadataAndYaml").Path("/api/v1/redact/spec/{slug}").Methods("GET").
		HandlerFunc(middleware.EnforceAccess(policy.RedactorRead, handler.GetRedactMetadataAndYaml))
	r.Name("SetRedactMetadataAndYaml").Path("/api/v1/redact/spec/{slug}").Methods("POST").
		HandlerFunc(middleware.EnforceAccess(policy.RedactorWrite, handler.SetRedactMetadataAndYaml))
	r.Name("DeleteRedact").Path("/api/v1/redact/spec/{slug}").Methods("DELETE").
		HandlerFunc(middleware.EnforceAccess(policy.RedactorWrite, handler.DeleteRedact))
	r.Name("SetRedactEnabled").Path("/api/v1/redact/enabled/{slug}").Methods("POST").
		HandlerFunc(middleware.EnforceAccess(policy.RedactorWrite, handler.SetRedactEnabled))

	// Kotsadm Identity Service
	r.Name("ConfigureIdentityService").Path("/api/v1/identity/config").Methods("POST").
		HandlerFunc(middleware.EnforceAccess(policy.IdentityServiceWrite, handler.ConfigureIdentityService))
	r.Name("GetIdentityServiceConfig").Path("/api/v1/identity/config").Methods("GET").
		HandlerFunc(middleware.EnforceAccess(policy.IdentityServiceRead, handler.GetIdentityServiceConfig))

	// App Identity Service
	r.Name("ConfigureAppIdentityService").Path("/api/v1/app/{appSlug}/identity/config").Methods("POST").
		HandlerFunc(middleware.EnforceAccess(policy.AppIdentityServiceWrite, handler.ConfigureAppIdentityService))
	r.Name("GetAppIdentityServiceConfig").Path("/api/v1/app/{appSlug}/identity/config").Methods("GET").
		HandlerFunc(middleware.EnforceAccess(policy.AppIdentityServiceRead, handler.GetAppIdentityServiceConfig))

	// Apps
	r.Name("GetPendingApp").Path("/api/v1/pendingapp").Methods("GET").
		HandlerFunc(middleware.EnforceAccess(policy.AppList, handler.GetPendingApp))
	r.Name("ListApps").Path("/api/v1/apps").Methods("GET").
		HandlerFunc(middleware.EnforceAccess(policy.AppList, handler.ListApps))
	r.Name("GetApp").Path("/api/v1/app/{appSlug}").Methods("GET").
		HandlerFunc(middleware.EnforceAccess(policy.AppRead, handler.GetApp))
	r.Name("GetAppStatus").Path("/api/v1/app/{appSlug}/status").Methods("GET").
		HandlerFunc(middleware.EnforceAccess(policy.AppStatusRead, handler.GetAppStatus))
	r.Name("GetAppVersionHistory").Path("/api/v1/app/{appSlug}/versions").Methods("GET").
		HandlerFunc(middleware.EnforceAccess(policy.AppDownstreamRead, handler.GetAppVersionHistory))
	r.Name("GetLatestDeployableVersion").Path("/api/v1/app/{appSlug}/next-app-version").Methods("GET").
		HandlerFunc(middleware.EnforceAccess(policy.AppRead, handler.GetLatestDeployableVersion))
	r.Name("GetUpdateDownloadStatus").Path("/api/v1/app/{appSlug}/task/updatedownload").Methods("GET").
		HandlerFunc(middleware.EnforceAccess(policy.AppRead, handler.GetUpdateDownloadStatus)) // NOTE: appSlug is unused

	// Airgap
	r.Name("AirgapBundleProgress").Path("/api/v1/app/{appSlug}/airgap/bundleprogress/{identifier}/{totalChunks}").Methods("GET").
		HandlerFunc(middleware.EnforceAccess(policy.AppDownstreamWrite, handler.AirgapBundleProgress))
	r.Name("AirgapBundleExists").Path("/api/v1/app/{appSlug}/airgap/bundleexists/{identifier}/{totalChunks}").Methods("GET").
		HandlerFunc(middleware.EnforceAccess(policy.AppDownstreamWrite, handler.AirgapBundleExists))
	r.Name("CreateAppFromAirgap").Path("/api/v1/app/{appSlug}/airgap/processbundle/{identifier}/{totalChunks}").Methods("POST").
		HandlerFunc(middleware.EnforceAccess(policy.AppDownstreamWrite, handler.CreateAppFromAirgap))
	r.Name("UpdateAppFromAirgap").Path("/api/v1/app/{appSlug}/airgap/processbundle/{identifier}/{totalChunks}").Methods("PUT").
		HandlerFunc(middleware.EnforceAccess(policy.AppDownstreamWrite, handler.UpdateAppFromAirgap))
	r.Name("CheckAirgapBundleChunk").Path("/api/v1/app/{appSlug}/airgap/chunk").Methods("GET").
		HandlerFunc(middleware.EnforceAccess(policy.AppDownstreamWrite, handler.CheckAirgapBundleChunk))
	r.Name("UploadAirgapBundleChunk").Path("/api/v1/app/{appSlug}/airgap/chunk").Methods("POST").
		HandlerFunc(middleware.EnforceAccess(policy.AppDownstreamWrite, handler.UploadAirgapBundleChunk))
	r.Name("GetAirgapInstallStatus").Path("/api/v1/app/{appSlug}/airgap/status").Methods("GET").
		HandlerFunc(middleware.EnforceAccess(policy.AppDownstreamWrite, handler.GetAirgapInstallStatus))
	r.Name("ResetAirgapInstallStatus").Path("/api/v1/app/{appSlug}/airgap/reset").Methods("POST").
		HandlerFunc(middleware.EnforceAccess(policy.AppDownstreamWrite, handler.ResetAirgapInstallStatus))
	r.Name("GetAirgapUploadConfig").Path("/api/v1/app/{appSlug}/airgap/config").Methods("GET").
		HandlerFunc(middleware.EnforceAccess(policy.AppDownstreamWrite, handler.GetAirgapUploadConfig))

	// Implemented handlers
	r.Name("IgnorePreflightRBACErrors").Path("/api/v1/app/{appSlug}/sequence/{sequence}/preflight/ignore-rbac").Methods("POST").
		HandlerFunc(middleware.EnforceAccess(policy.AppDownstreamPreflightWrite, handler.IgnorePreflightRBACErrors))
	r.Name("StartPreflightChecks").Path("/api/v1/app/{appSlug}/sequence/{sequence}/preflight/run").Methods("POST").
		HandlerFunc(middleware.EnforceAccess(policy.AppDownstreamPreflightWrite, handler.StartPreflightChecks))
	r.Name("GetLatestPreflightResultsForSequenceZero").Path("/api/v1/app/{appSlug}/preflight/result").Methods("GET").
		HandlerFunc(middleware.EnforceAccess(policy.AppDownstreamPreflightRead, handler.GetLatestPreflightResultsForSequenceZero))
	r.Name("GetPreflightResult").Path("/api/v1/app/{appSlug}/sequence/{sequence}/preflight/result").Methods("GET").
		HandlerFunc(middleware.EnforceAccess(policy.AppDownstreamPreflightRead, handler.GetPreflightResult))
	r.Name("GetPreflightCommand").Path("/api/v1/app/{appSlug}/sequence/{sequence}/preflightcommand").Methods("POST").
		HandlerFunc(middleware.EnforceAccess(policy.AppRead, handler.GetPreflightCommand)) // this is intentional
	r.Name("PreflightsReports").Path("/api/v1/app/{appSlug}/preflight/report").Methods("POST").
		HandlerFunc(middleware.EnforceAccess(policy.AppDownstreamPreflightWrite, handler.PreflightsReports))

	r.Name("UpdateAdminConsole").Path("/api/v1/app/{appSlug}/sequence/{sequence}/update-console").Methods("POST").
		HandlerFunc(middleware.EnforceAccess(policy.ClusterWrite, handler.UpdateAdminConsole))
	r.Name("GetAdminConsoleUpdateStatus").Path("/api/v1/app/{appSlug}/task/update-admin-console").Methods("GET").
		HandlerFunc(middleware.EnforceAccess(policy.ClusterWrite, handler.GetAdminConsoleUpdateStatus))
	r.Name("DownloadAppVersion").Path("/api/v1/app/{appSlug}/sequence/{sequence}/download").Methods("POST").
		HandlerFunc(middleware.EnforceAccess(policy.AppDownstreamWrite, handler.DownloadAppVersion))
	r.Name("GetAppVersionDownloadStatus").Path("/api/v1/app/{appSlug}/sequence/{sequence}/task/updatedownload").Methods("GET").
		HandlerFunc(middleware.EnforceAccess(policy.AppRead, handler.GetAppVersionDownloadStatus)) // NOTE: appSlug is unused
	r.Name("DeployAppVersion").Path("/api/v1/app/{appSlug}/sequence/{sequence}/deploy").Methods("POST").
		HandlerFunc(middleware.EnforceAccess(policy.AppDownstreamWrite, handler.DeployAppVersion))
	r.Name("RedeployAppVersion").Path("/api/v1/app/{appSlug}/sequence/{sequence}/redeploy").Methods("POST").
		HandlerFunc(middleware.EnforceAccess(policy.AppDownstreamWrite, handler.RedeployAppVersion))
	r.Name("GetAppRenderedContents").Path("/api/v1/app/{appSlug}/sequence/{sequence}/renderedcontents").Methods("GET").
		HandlerFunc(middleware.EnforceAccess(policy.AppDownstreamFiletreeRead, handler.GetAppRenderedContents))
	r.Name("GetAppContents").Path("/api/v1/app/{appSlug}/sequence/{sequence}/contents").Methods("GET").
		HandlerFunc(middleware.EnforceAccess(policy.AppDownstreamFiletreeRead, handler.GetAppContents))
	r.Name("GetAppDashboard").Path("/api/v1/app/{appSlug}/cluster/{clusterId}/dashboard").Methods("GET").
		HandlerFunc(middleware.EnforceAccess(policy.AppRead, handler.GetAppDashboard))
	r.Name("GetDownstreamOutput").Path("/api/v1/app/{appSlug}/cluster/{clusterId}/sequence/{sequence}/downstreamoutput").Methods("GET").
		HandlerFunc(middleware.EnforceAccess(policy.AppDownstreamLogsRead, handler.GetDownstreamOutput))

	r.Name("GetKotsadmRegistry").Path("/api/v1/registry").Methods("GET").
		HandlerFunc(middleware.EnforceAccess(policy.RegistryRead, handler.GetKotsadmRegistry))
	r.Name("GetImageRewriteStatusOld").Path("/api/v1/imagerewritestatus").Methods("GET").
		HandlerFunc(middleware.EnforceAccess(policy.RegistryRead, handler.GetImageRewriteStatus))
	r.Name("GarbageCollectImages").Path("/api/v1/garbage-collect-images").Methods("POST").
		HandlerFunc(middleware.EnforceAccess(policy.AppCreate, handler.GarbageCollectImages))
	r.Name("DockerHubSecretUpdated").Path("/api/v1/docker/secret-updated").Methods("POST").
		HandlerFunc(middleware.EnforceAccess(policy.AppCreate, handler.DockerHubSecretUpdated))

	r.Name("UpdateAppRegistry").Path("/api/v1/app/{appSlug}/registry").Methods("PUT").
		HandlerFunc(middleware.EnforceAccess(policy.AppRegistryWrite, handler.UpdateAppRegistry))
	r.Name("GetAppRegistry").Path("/api/v1/app/{appSlug}/registry").Methods("GET").
		HandlerFunc(middleware.EnforceAccess(policy.AppRegistryRead, handler.GetAppRegistry))
	r.Name("GetImageRewriteStatus").Path("/api/v1/app/{appSlug}/imagerewritestatus").Methods("GET").
		HandlerFunc(middleware.EnforceAccess(policy.AppRegistryRead, handler.GetImageRewriteStatus))
	r.Name("ValidateAppRegistry").Path("/api/v1/app/{appSlug}/registry/validate").Methods("POST").
		HandlerFunc(middleware.EnforceAccess(policy.AppRegistryWrite, handler.ValidateAppRegistry))

	r.Name("UpdateAppConfig").Path("/api/v1/app/{appSlug}/config").Methods("PUT").
		HandlerFunc(middleware.EnforceAccess(policy.AppDownstreamConfigWrite, handler.UpdateAppConfig))
	r.Name("CurrentAppConfig").Path("/api/v1/app/{appSlug}/config/{sequence}").Methods("GET").
		HandlerFunc(middleware.EnforceAccess(policy.AppDownstreamConfigRead, handler.CurrentAppConfig))
	r.Name("LiveAppConfig").Path("/api/v1/app/{appSlug}/liveconfig").Methods("POST").
		HandlerFunc(middleware.EnforceAccess(policy.AppDownstreamConfigWrite, handler.LiveAppConfig))
	r.Name("SetAppConfigValues").Path("/api/v1/app/{appSlug}/config/values").Methods("POST").
		HandlerFunc(middleware.EnforceAccess(policy.AppDownstreamConfigWrite, handler.SetAppConfigValues))
	r.Name("DownloadFileFromConfig").Path("/api/v1/app/{appSlug}/config/{sequence}/{filename}/download").Methods("GET").
		HandlerFunc(middleware.EnforceAccess(policy.AppDownstreamConfigRead, handler.DownloadFileFromConfig))

	r.Name("SyncLicense").Path("/api/v1/app/{appSlug}/license").Methods("PUT").
		HandlerFunc(middleware.EnforceAccess(policy.AppLicenseWrite, handler.SyncLicense))
	r.Name("ChangeLicense").Path("/api/v1/app/{appSlug}/change-license").Methods("PUT").
		HandlerFunc(middleware.EnforceAccess(policy.AppLicenseWrite, handler.ChangeLicense))
	r.Name("GetLicense").Path("/api/v1/app/{appSlug}/license").Methods("GET").
		HandlerFunc(middleware.EnforceAccess(policy.AppLicenseRead, handler.GetLicense))

	r.Name("AppUpdateCheck").Path("/api/v1/app/{appSlug}/updatecheck").Methods("POST").
		HandlerFunc(middleware.EnforceAccess(policy.AppDownstreamWrite, handler.AppUpdateCheck))
	r.Name("ConfigureAutomaticUpdates").Path("/api/v1/app/{appSlug}/automaticupdates").Methods("PUT").
		HandlerFunc(middleware.EnforceAccess(policy.AppDownstreamWrite, handler.ConfigureAutomaticUpdates))
	r.Name("RemoveApp").Path("/api/v1/app/{appSlug}/remove").Methods("POST").
		HandlerFunc(middleware.EnforceAccess(policy.AppUpdate, handler.RemoveApp))

	// App snapshot routes
	r.Name("CreateApplicationBackup").Path("/api/v1/app/{appSlug}/snapshot/backup").Methods("POST").
		HandlerFunc(middleware.EnforceAccess(policy.AppBackupWrite, handler.CreateApplicationBackup))
	r.Name("GetRestoreStatus").Path("/api/v1/app/{appSlug}/snapshot/restore/status").Methods("GET").
		HandlerFunc(middleware.EnforceAccess(policy.AppRestoreRead, handler.GetRestoreStatus))
	r.Name("CancelRestore").Path("/api/v1/app/{appSlug}/snapshot/restore").Methods("DELETE").
		HandlerFunc(middleware.EnforceAccess(policy.AppRestoreWrite, handler.CancelRestore))
	r.Name("CreateApplicationRestore").Path("/api/v1/app/{appSlug}/snapshot/restore/{snapshotName}").Methods("POST").
		HandlerFunc(middleware.EnforceAccess(policy.AppRestoreWrite, handler.CreateApplicationRestore))
	r.Name("GetRestoreDetails").Path("/api/v1/app/{appSlug}/snapshot/restore/{restoreName}").Methods("GET").
		HandlerFunc(middleware.EnforceAccess(policy.AppRestoreRead, handler.GetRestoreDetails))
	r.Name("ListBackups").Path("/api/v1/app/{appSlug}/snapshots").Methods("GET").
		HandlerFunc(middleware.EnforceAccess(policy.AppBackupRead, handler.ListBackups))
	r.Name("GetSnapshotConfig").Path("/api/v1/app/{appSlug}/snapshot/config").Methods("GET").
		HandlerFunc(middleware.EnforceAccess(policy.AppSnapshotsettingsRead, handler.GetSnapshotConfig))
	r.Name("SaveSnapshotConfig").Path("/api/v1/app/{appSlug}/snapshot/config").Methods("PUT").
		HandlerFunc(middleware.EnforceAccess(policy.AppSnapshotsettingsWrite, handler.SaveSnapshotConfig))

	// Global snapshot routes
	r.Name("ListInstanceBackups").Path("/api/v1/snapshots").Methods("GET").
		HandlerFunc(middleware.EnforceAccess(policy.BackupRead, handler.ListInstanceBackups))
	r.Name("CreateInstanceBackup").Path("/api/v1/snapshot/backup").Methods("POST").
		HandlerFunc(middleware.EnforceAccess(policy.BackupWrite, handler.CreateInstanceBackup))
	r.Name("GetInstanceSnapshotConfig").Path("/api/v1/snapshot/config").Methods("GET").
		HandlerFunc(middleware.EnforceAccess(policy.SnapshotsettingsRead, handler.GetInstanceSnapshotConfig))
	r.Name("SaveInstanceSnapshotConfig").Path("/api/v1/snapshot/config").Methods("PUT").
		HandlerFunc(middleware.EnforceAccess(policy.SnapshotsettingsWrite, handler.SaveInstanceSnapshotConfig))
	r.Name("GetGlobalSnapshotSettings").Path("/api/v1/snapshots/settings").Methods("GET").
		HandlerFunc(middleware.EnforceAccess(policy.SnapshotsettingsRead, handler.GetGlobalSnapshotSettings))
	r.Name("UpdateGlobalSnapshotSettings").Path("/api/v1/snapshots/settings").Methods("PUT").
		HandlerFunc(middleware.EnforceAccess(policy.SnapshotsettingsWrite, handler.UpdateGlobalSnapshotSettings))
	r.Name("ConfigureFileSystemSnapshotProvider").Path("/api/v1/snapshots/filesystem").Methods("PUT").
		HandlerFunc(middleware.EnforceAccess(policy.SnapshotsettingsWrite, handler.ConfigureFileSystemSnapshotProvider))
	r.Name("GetBackup").Path("/api/v1/snapshot/{snapshotName}").Methods("GET").
		HandlerFunc(middleware.EnforceAccess(policy.BackupRead, handler.GetBackup))
	r.Name("DeleteBackup").Path("/api/v1/snapshot/{snapshotName}/delete").Methods("POST").
		HandlerFunc(middleware.EnforceAccess(policy.BackupWrite, handler.DeleteBackup))
	r.Name("RestoreApps").Path("/api/v1/snapshot/{snapshotName}/restore-apps").Methods("POST").
		HandlerFunc(middleware.EnforceAccess(policy.RestoreWrite, handler.RestoreApps))
	r.Name("GetRestoreAppsStatus").Path("/api/v1/snapshot/{snapshotName}/apps-restore-status").Methods("POST").
		HandlerFunc(middleware.EnforceAccess(policy.RestoreWrite, handler.GetRestoreAppsStatus))
	r.Name("DownloadSnapshotLogs").Path("/api/v1/snapshot/{backup}/logs").Methods("GET").
		HandlerFunc(middleware.EnforceAccess(policy.BackupRead, handler.DownloadSnapshotLogs))
	r.Name("GetVeleroStatus").Path("/api/v1/velero").Methods("GET").
		HandlerFunc(middleware.EnforceAccess(policy.BackupRead, handler.GetVeleroStatus))

	// KURL
	r.Name("Kurl").Path("/api/v1/kurl").HandlerFunc(NotImplemented) // I'm not sure why this is here
	r.Name("GenerateNodeJoinCommandWorker").Path("/api/v1/kurl/generate-node-join-command-worker").Methods("POST").
		HandlerFunc(middleware.EnforceAccess(policy.ClusterWrite, handler.GenerateNodeJoinCommandWorker))
	r.Name("GenerateNodeJoinCommandMaster").Path("/api/v1/kurl/generate-node-join-command-master").Methods("POST").
		HandlerFunc(middleware.EnforceAccess(policy.ClusterWrite, handler.GenerateNodeJoinCommandMaster))
	r.Name("GenerateNodeJoinCommandSecondary").Path("/api/v1/kurl/generate-node-join-command-secondary").Methods("POST").
		HandlerFunc(middleware.EnforceAccess(policy.ClusterWrite, handler.GenerateNodeJoinCommandSecondary))
	r.Name("GenerateNodeJoinCommandPrimary").Path("/api/v1/kurl/generate-node-join-command-primary").Methods("POST").
		HandlerFunc(middleware.EnforceAccess(policy.ClusterWrite, handler.GenerateNodeJoinCommandPrimary))
	r.Name("DrainNode").Path("/api/v1/kurl/nodes/{nodeName}/drain").Methods("POST").
		HandlerFunc(middleware.EnforceAccess(policy.ClusterWrite, handler.DrainNode))
	r.Name("DeleteNode").Path("/api/v1/kurl/nodes/{nodeName}").Methods("DELETE").
		HandlerFunc(middleware.EnforceAccess(policy.ClusterWrite, handler.DeleteNode))
	r.Name("GetKurlNodes").Path("/api/v1/kurl/nodes").Methods("GET").
		HandlerFunc(middleware.EnforceAccess(policy.ClusterRead, handler.GetKurlNodes))

	// Prometheus
	r.Name("SetPrometheusAddress").Path("/api/v1/prometheus").Methods("POST").
		HandlerFunc(middleware.EnforceAccess(policy.PrometheussettingsWrite, handler.SetPrometheusAddress))

	// GitOps
	r.Name("UpdateAppGitOps").Path("/api/v1/gitops/app/{appId}/cluster/{clusterId}/update").Methods("PUT").
		HandlerFunc(middleware.EnforceAccess(policy.AppGitopsWrite, handler.UpdateAppGitOps))
	r.Name("DisableAppGitOps").Path("/api/v1/gitops/app/{appId}/cluster/{clusterId}/disable").Methods("POST").
		HandlerFunc(middleware.EnforceAccess(policy.AppGitopsWrite, handler.DisableAppGitOps))
	r.Name("InitGitOpsConnection").Path("/api/v1/gitops/app/{appId}/cluster/{clusterId}/initconnection").Methods("POST").
		HandlerFunc(middleware.EnforceAccess(policy.AppGitopsWrite, handler.InitGitOpsConnection))
	r.Name("CreateGitOps").Path("/api/v1/gitops/create").Methods("POST").
		HandlerFunc(middleware.EnforceAccess(policy.GitopsWrite, handler.CreateGitOps))
	r.Name("ResetGitOps").Path("/api/v1/gitops/reset").Methods("POST").
		HandlerFunc(middleware.EnforceAccess(policy.GitopsWrite, handler.ResetGitOps))
	r.Name("GetGitOpsRepo").Path("/api/v1/gitops/get").Methods("GET").
		HandlerFunc(middleware.EnforceAccess(policy.GitopsRead, handler.GetGitOpsRepo))

	// Password change
	r.Name("ChangePassword").Path("/api/v1/password/change").Methods("PUT").
		HandlerFunc(middleware.EnforceAccess(policy.PasswordChange, handler.ChangePassword))

	// Helm
	r.Name("IsHelmManaged").Path("/api/v1/is-helm-managed").Methods("GET").
		HandlerFunc(middleware.EnforceAccess(policy.IsHelmManaged, handler.IsHelmManaged))
	r.Name("GetAppValuesFile").Path("/api/v1/app/{appSlug}/values").Methods("GET").
		HandlerFunc(middleware.EnforceAccess(policy.GetAppValuesFile, handler.GetAppValuesFile))
}

func JSON(w http.ResponseWriter, code int, payload interface{}) {
	response, err := json.Marshal(payload)
	if err != nil {
		logger.Error(err)
		w.WriteHeader(500)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	w.Write(response)
}

func RegisterTokenAuthRoutes(handler *Handler, debugRouter *mux.Router, loggingRouter *mux.Router) {
	debugRouter.Path("/api/v1/kots/ports").Methods("GET").HandlerFunc(handler.GetApplicationPorts)
	loggingRouter.Path("/api/v1/upload").Methods("PUT").HandlerFunc(handler.UploadExistingApp)
	loggingRouter.Path("/api/v1/download").Methods("GET").HandlerFunc(handler.DownloadApp)
	loggingRouter.Path("/api/v1/airgap/install").Methods("POST").HandlerFunc(handler.UploadInitialAirgapApp)
}

func RegisterUnauthenticatedRoutes(handler *Handler, debugRouter *mux.Router, loggingRouter *mux.Router) {
	debugRouter.HandleFunc("/healthz", handler.Healthz)
	loggingRouter.HandleFunc("/api/v1/login", handler.Login)
	loggingRouter.HandleFunc("/api/v1/login/info", handler.GetLoginInfo)
	loggingRouter.HandleFunc("/api/v1/logout", handler.Logout) // this route uses its own auth
	loggingRouter.Path("/api/v1/metadata").Methods("GET").HandlerFunc(GetMetadataHandler(GetMetaDataConfig))

	loggingRouter.HandleFunc("/api/v1/oidc/login", handler.OIDCLogin)
	loggingRouter.HandleFunc("/api/v1/oidc/login/callback", handler.OIDCLoginCallback)

	loggingRouter.Path("/api/v1/troubleshoot/{appId}/{bundleId}").Methods("PUT").HandlerFunc(handler.UploadSupportBundle)
	loggingRouter.Path("/api/v1/troubleshoot/supportbundle/{bundleId}/redactions").Methods("PUT").HandlerFunc(handler.SetSupportBundleRedactions)
	loggingRouter.Path("/api/v1/preflight/app/{appSlug}/sequence/{sequence}").Methods("POST").HandlerFunc(handler.PostPreflightStatus)

	// This the handler for license API and should be called by the application only.
	loggingRouter.Path("/license/v1/license").Methods("GET").HandlerFunc(handler.GetPlatformLicenseCompatibility)
}

func StreamJSON(c *websocket.Conn, payload interface{}) {
	response, err := json.Marshal(payload)
	if err != nil {
		logger.Error(err)
		return
	}

	err = c.WriteMessage(websocket.TextMessage, response)
	if err != nil {
		logger.Error(err)
		return
	}
}

func YAML(w http.ResponseWriter, code int, payload interface{}) {
	response, err := yaml.Marshal(payload)
	if err != nil {
		logger.Error(err)
		w.WriteHeader(500)
		return
	}

	w.Header().Set("Content-Type", "text/yaml")
	w.WriteHeader(code)
	w.Write(response)
}
