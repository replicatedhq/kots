package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
	"github.com/replicatedhq/kots/kotsadm/pkg/logger"
	"github.com/replicatedhq/kots/kotsadm/pkg/policy"
	"github.com/replicatedhq/kots/kotsadm/pkg/store"
	kotsscheme "github.com/replicatedhq/kots/kotskinds/client/kotsclientset/scheme"
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

func RegisterSessionAuthRoutes(r *mux.Router, kotsStore store.KOTSStore, handler KOTSHandler, middleware *policy.Middleware) {
	r.Use(RequireValidSessionMiddleware(kotsStore))

	r.Name("GetSupportBundle").Path("/api/v1/troubleshoot/supportbundle/{bundleSlug}").Methods("GET").
		HandlerFunc(middleware.EnforceAccess(policy.AppSupportbundleRead, handler.GetSupportBundle)) // TODO: appSlug
	r.Name("ConfigureAppIdentityService").Path("/api/v1/app/{appSlug}/identity/config").Methods("POST").
		HandlerFunc(middleware.EnforceAccess(policy.AppIdentityServiceWrite, handler.ConfigureAppIdentityService))
}

func _RegisterSessionAuthRoutes(r *mux.Router, kotsStore store.KOTSStore, handler KOTSHandler, middleware *policy.Middleware) {
	r.Use(RequireValidSessionMiddleware(kotsStore))

	// Installation
	r.Name("UploadNewLicense").Path("/api/v1/license").Methods("POST").
		HandlerFunc(middleware.EnforceAccess(policy.AppCreate, handler.UploadNewLicense))
	r.Path("/api/v1/license/platform").Methods("POST").
		HandlerFunc(middleware.EnforceAccess(policy.AppCreate, handler.ExchangePlatformLicense))
	r.Path("/api/v1/license/resume").Methods("PUT").
		HandlerFunc(middleware.EnforceAccess(policy.AppCreate, handler.ResumeInstallOnline))
	r.Path("/api/v1/app/online/status").Methods("GET").
		HandlerFunc(middleware.EnforceAccess(policy.AppCreate, handler.GetOnlineInstallStatus))

	// Support Bundles
	r.Name("GetSupportBundle").Path("/api/v1/troubleshoot/supportbundle/{bundleSlug}").Methods("GET").
		HandlerFunc(middleware.EnforceAccess(policy.AppSupportbundleRead, handler.GetSupportBundle)) // TODO: appSlug
	r.Path("/api/v1/troubleshoot/app/{appSlug}/supportbundles").Methods("GET").
		HandlerFunc(middleware.EnforceAccess(policy.AppSupportbundleRead, handler.ListSupportBundles))
	r.Path("/api/v1/troubleshoot/app/{appSlug}/supportbundlecommand").Methods("POST").
		HandlerFunc(middleware.EnforceAccess(policy.AppSupportbundleRead, handler.GetSupportBundleCommand))
	r.Path("/api/v1/troubleshoot/supportbundle/{bundleId}/files").Methods("GET").
		HandlerFunc(middleware.EnforceAccess(policy.AppSupportbundleRead, handler.GetSupportBundleFiles)) // TODO: appSlug
	r.Path("/api/v1/troubleshoot/supportbundle/{bundleId}/redactions").Methods("GET").
		HandlerFunc(middleware.EnforceAccess(policy.AppSupportbundleRead, handler.GetSupportBundleRedactions)) // TODO: appSlug
	r.Path("/api/v1/troubleshoot/supportbundle/{bundleId}/download").Methods("GET").
		HandlerFunc(middleware.EnforceAccess(policy.AppSupportbundleRead, handler.DownloadSupportBundle)) // TODO: appSlug
	r.Path("/api/v1/troubleshoot/supportbundle/app/{appId}/cluster/{clusterId}/collect").Methods("POST").
		HandlerFunc(middleware.EnforceAccess(policy.AppSupportbundleWrite, handler.CollectSupportBundle))

	// redactor routes
	r.Path("/api/v1/redact/set").Methods("PUT").
		HandlerFunc(middleware.EnforceAccess(policy.RedactorWrite, handler.UpdateRedact))
	r.Path("/api/v1/redact/get").Methods("GET").
		HandlerFunc(middleware.EnforceAccess(policy.RedactorRead, handler.GetRedact))
	r.Path("/api/v1/redacts").Methods("GET").
		HandlerFunc(middleware.EnforceAccess(policy.RedactorRead, handler.ListRedactors))
	r.Path("/api/v1/redact/spec/{slug}").Methods("GET").
		HandlerFunc(middleware.EnforceAccess(policy.RedactorRead, handler.GetRedactMetadataAndYaml))
	r.Path("/api/v1/redact/spec/{slug}").Methods("POST").
		HandlerFunc(middleware.EnforceAccess(policy.RedactorWrite, handler.SetRedactMetadataAndYaml))
	r.Path("/api/v1/redact/spec/{slug}").Methods("DELETE").
		HandlerFunc(middleware.EnforceAccess(policy.RedactorWrite, handler.DeleteRedact))
	r.Path("/api/v1/redact/enabled/{slug}").Methods("POST").
		HandlerFunc(middleware.EnforceAccess(policy.RedactorWrite, handler.SetRedactEnabled))

	// Kotsadm Identity Service
	r.Path("/api/v1/identity/config").Methods("POST").
		HandlerFunc(middleware.EnforceAccess(policy.IdentityServiceWrite, handler.ConfigureIdentityService))
	r.Path("/api/v1/identity/config").Methods("GET").
		HandlerFunc(middleware.EnforceAccess(policy.IdentityServiceRead, handler.GetIdentityServiceConfig))

	// App Identity Service
	r.Path("/api/v1/app/{appSlug}/identity/config").Methods("POST").
		HandlerFunc(middleware.EnforceAccess(policy.AppIdentityServiceWrite, handler.ConfigureAppIdentityService))
	r.Path("/api/v1/app/{appSlug}/identity/config").Methods("GET").
		HandlerFunc(middleware.EnforceAccess(policy.AppIdentityServiceRead, handler.GetAppIdentityServiceConfig))

	// Apps
	r.Path("/api/v1/apps").Methods("GET").
		HandlerFunc(middleware.EnforceAccess(policy.AppList, handler.ListApps))
	r.Path("/api/v1/app/{appSlug}").Methods("GET").
		HandlerFunc(middleware.EnforceAccess(policy.AppRead, handler.GetApp))
	r.Path("/api/v1/app/{appSlug}/status").Methods("GET").
		HandlerFunc(middleware.EnforceAccess(policy.AppStatusRead, handler.GetAppStatus))
	r.Path("/api/v1/app/{appSlug}/versions").Methods("GET").
		HandlerFunc(middleware.EnforceAccess(policy.AppDownstreamRead, handler.GetAppVersionHistory))
	r.Path("/api/v1/app/{appSlug}/task/updatedownload").Methods("GET").
		HandlerFunc(middleware.EnforceAccess(policy.AppRead, handler.GetUpdateDownloadStatus)) // NOTE: appSlug is unused

	// Airgap
	r.Path("/api/v1/app/{appSlug}/airgap/bundleprogress/{identifier}/{totalChunks}").Methods("GET").
		HandlerFunc(middleware.EnforceAccess(policy.AppDownstreamWrite, handler.AirgapBundleProgress))
	r.Path("/api/v1/app/{appSlug}/airgap/bundleexists/{identifier}/{totalChunks}").Methods("GET").
		HandlerFunc(middleware.EnforceAccess(policy.AppDownstreamWrite, handler.AirgapBundleExists))
	r.Path("/api/v1/app/{appSlug}/airgap/processbundle/{identifier}/{totalChunks}").Methods("POST").
		HandlerFunc(middleware.EnforceAccess(policy.AppDownstreamWrite, handler.CreateAppFromAirgap))
	r.Path("/api/v1/app/{appSlug}/airgap/processbundle/{identifier}/{totalChunks}").Methods("PUT").
		HandlerFunc(middleware.EnforceAccess(policy.AppDownstreamWrite, handler.UpdateAppFromAirgap))
	r.Path("/api/v1/app/{appSlug}/airgap/chunk").Methods("GET").
		HandlerFunc(middleware.EnforceAccess(policy.AppDownstreamWrite, handler.CheckAirgapBundleChunk))
	r.Path("/api/v1/app/{appSlug}/airgap/chunk").Methods("POST").
		HandlerFunc(middleware.EnforceAccess(policy.AppDownstreamWrite, handler.UploadAirgapBundleChunk))
	r.Path("/api/v1/app/{appSlug}/airgap/status").Methods("GET").
		HandlerFunc(middleware.EnforceAccess(policy.AppDownstreamWrite, handler.GetAirgapInstallStatus))
	r.Path("/api/v1/app/{appSlug}/airgap/reset").Methods("POST").
		HandlerFunc(middleware.EnforceAccess(policy.AppDownstreamWrite, handler.ResetAirgapInstallStatus))

	// Implemented handlers
	r.Path("/api/v1/app/{appSlug}/sequence/{sequence}/preflight/ignore-rbac").Methods("POST").
		HandlerFunc(middleware.EnforceAccess(policy.AppDownstreamPreflightWrite, handler.IgnorePreflightRBACErrors))
	r.Path("/api/v1/app/{appSlug}/sequence/{sequence}/preflight/run").Methods("POST").
		HandlerFunc(middleware.EnforceAccess(policy.AppDownstreamPreflightWrite, handler.StartPreflightChecks))
	r.Path("/api/v1/app/{appSlug}/preflight/result").Methods("GET").
		HandlerFunc(middleware.EnforceAccess(policy.AppDownstreamPreflightRead, handler.GetLatestPreflightResultsForSequenceZero))
	r.Path("/api/v1/app/{appSlug}/sequence/{sequence}/preflight/result").Methods("GET").
		HandlerFunc(middleware.EnforceAccess(policy.AppDownstreamPreflightRead, handler.GetPreflightResult))
	r.Path("/api/v1/app/{appSlug}/sequence/{sequence}/preflightcommand").Methods("POST").
		HandlerFunc(middleware.EnforceAccess(policy.AppRead, handler.GetPreflightCommand)) // this is intentionally policy.AppRead

	r.Path("/api/v1/app/{appSlug}/sequence/{sequence}/deploy").Methods("POST").
		HandlerFunc(middleware.EnforceAccess(policy.AppDownstreamWrite, handler.DeployAppVersion))
	r.Path("/api/v1/app/{appSlug}/sequence/{sequence}/redeploy").Methods("POST").
		HandlerFunc(middleware.EnforceAccess(policy.AppDownstreamWrite, handler.RedeployAppVersion))
	r.Path("/api/v1/app/{appSlug}/sequence/{sequence}/renderedcontents").Methods("GET").
		HandlerFunc(middleware.EnforceAccess(policy.AppDownstreamFiletreeRead, handler.GetAppRenderedContents))
	r.Path("/api/v1/app/{appSlug}/sequence/{sequence}/contents").Methods("GET").
		HandlerFunc(middleware.EnforceAccess(policy.AppDownstreamFiletreeRead, handler.GetAppContents))
	r.Path("/api/v1/app/{appSlug}/cluster/{clusterId}/dashboard").Methods("GET").
		HandlerFunc(middleware.EnforceAccess(policy.AppRead, handler.GetAppDashboard))
	r.Path("/api/v1/app/{appSlug}/cluster/{clusterId}/sequence/{sequence}/downstreamoutput").Methods("GET").
		HandlerFunc(middleware.EnforceAccess(policy.AppDownstreamLogsRead, handler.GetDownstreamOutput))

	r.Path("/api/v1/registry").Methods("GET").
		HandlerFunc(middleware.EnforceAccess(policy.RegistryRead, handler.GetKotsadmRegistry))
	r.Path("/api/v1/imagerewritestatus").Methods("GET").
		HandlerFunc(middleware.EnforceAccess(policy.RegistryRead, handler.GetImageRewriteStatus))

	r.Path("/api/v1/app/{appSlug}/registry").Methods("PUT").
		HandlerFunc(middleware.EnforceAccess(policy.AppRegistryWrite, handler.UpdateAppRegistry))
	r.Path("/api/v1/app/{appSlug}/registry").Methods("GET").
		HandlerFunc(middleware.EnforceAccess(policy.AppRegistryRead, handler.GetAppRegistry))
	r.Path("/api/v1/app/{appSlug}/imagerewritestatus").Methods("GET").
		HandlerFunc(middleware.EnforceAccess(policy.AppRegistryRead, handler.GetImageRewriteStatus))
	r.Path("/api/v1/app/{appSlug}/registry/validate").Methods("POST").
		HandlerFunc(middleware.EnforceAccess(policy.AppRegistryWrite, handler.ValidateAppRegistry))

	r.Path("/api/v1/app/{appSlug}/config").Methods("PUT").
		HandlerFunc(middleware.EnforceAccess(policy.AppDownstreamConfigWrite, handler.UpdateAppConfig))
	r.Path("/api/v1/app/{appSlug}/config/{sequence}").Methods("GET").
		HandlerFunc(middleware.EnforceAccess(policy.AppDownstreamConfigRead, handler.CurrentAppConfig))
	r.Path("/api/v1/app/{appSlug}/liveconfig").Methods("POST").
		HandlerFunc(middleware.EnforceAccess(policy.AppDownstreamConfigWrite, handler.LiveAppConfig))

	r.Path("/api/v1/app/{appSlug}/license").Methods("PUT").
		HandlerFunc(middleware.EnforceAccess(policy.AppLicenseWrite, handler.SyncLicense))
	r.Path("/api/v1/app/{appSlug}/license").Methods("GET").
		HandlerFunc(middleware.EnforceAccess(policy.AppLicenseRead, handler.GetLicense))

	r.Path("/api/v1/app/{appSlug}/updatecheck").Methods("POST").
		HandlerFunc(middleware.EnforceAccess(policy.AppDownstreamWrite, handler.AppUpdateCheck))
	r.Path("/api/v1/app/{appSlug}/updatecheckerspec").Methods("PUT").
		HandlerFunc(middleware.EnforceAccess(policy.AppDownstreamWrite, handler.UpdateCheckerSpec))
	r.Path("/api/v1/app/{appSlug}/remove").Methods("POST").
		HandlerFunc(middleware.EnforceAccess(policy.AppUpdate, handler.RemoveApp))

	// App snapshot routes
	r.Path("/api/v1/app/{appSlug}/snapshot/backup").Methods("POST").
		HandlerFunc(middleware.EnforceAccess(policy.AppBackupWrite, handler.CreateApplicationBackup))
	r.Path("/api/v1/app/{appSlug}/snapshot/restore/status").Methods("GET").
		HandlerFunc(middleware.EnforceAccess(policy.AppRestoreRead, handler.GetRestoreStatus))
	r.Path("/api/v1/app/{appSlug}/snapshot/restore").Methods("DELETE").
		HandlerFunc(middleware.EnforceAccess(policy.AppRestoreWrite, handler.CancelRestore))
	r.Path("/api/v1/app/{appSlug}/snapshot/restore/{snapshotName}").Methods("POST").
		HandlerFunc(middleware.EnforceAccess(policy.AppRestoreWrite, handler.CreateApplicationRestore))
	r.Path("/api/v1/app/{appSlug}/snapshot/restore/{restoreName}").Methods("GET").
		HandlerFunc(middleware.EnforceAccess(policy.AppRestoreRead, handler.GetRestoreDetails))
	r.Path("/api/v1/app/{appSlug}/snapshots").Methods("GET").
		HandlerFunc(middleware.EnforceAccess(policy.AppBackupRead, handler.ListBackups))
	r.Path("/api/v1/app/{appSlug}/snapshot/config").Methods("GET").
		HandlerFunc(middleware.EnforceAccess(policy.AppSnapshotsettingsRead, handler.GetSnapshotConfig))
	r.Path("/api/v1/app/{appSlug}/snapshot/config").Methods("PUT").
		HandlerFunc(middleware.EnforceAccess(policy.AppSnapshotsettingsWrite, handler.SaveSnapshotConfig))

	// Global snapshot routes
	r.Path("/api/v1/snapshots").Methods("GET").
		HandlerFunc(middleware.EnforceAccess(policy.BackupRead, handler.ListInstanceBackups))
	r.Path("/api/v1/snapshot/backup").Methods("POST").
		HandlerFunc(middleware.EnforceAccess(policy.BackupWrite, handler.CreateInstanceBackup))
	r.Path("/api/v1/snapshot/config").Methods("GET").
		HandlerFunc(middleware.EnforceAccess(policy.SnapshotsettingsRead, handler.GetInstanceSnapshotConfig))
	r.Path("/api/v1/snapshot/config").Methods("PUT").
		HandlerFunc(middleware.EnforceAccess(policy.SnapshotsettingsWrite, handler.SaveInstanceSnapshotConfig))
	r.Path("/api/v1/snapshots/settings").Methods("GET").
		HandlerFunc(middleware.EnforceAccess(policy.SnapshotsettingsRead, handler.GetGlobalSnapshotSettings))
	r.Path("/api/v1/snapshots/settings").Methods("PUT").
		HandlerFunc(middleware.EnforceAccess(policy.SnapshotsettingsWrite, handler.UpdateGlobalSnapshotSettings))
	r.Path("/api/v1/snapshot/{snapshotName}").Methods("GET").
		HandlerFunc(middleware.EnforceAccess(policy.BackupRead, handler.GetBackup))
	r.Path("/api/v1/snapshot/{snapshotName}/delete").Methods("POST").
		HandlerFunc(middleware.EnforceAccess(policy.BackupWrite, handler.DeleteBackup))
	r.Path("/api/v1/snapshot/{snapshotName}/restore-apps").Methods("POST").
		HandlerFunc(middleware.EnforceAccess(policy.RestoreWrite, handler.RestoreApps))
	r.Path("/api/v1/snapshot/{snapshotName}/apps-restore-status").Methods("GET").
		HandlerFunc(middleware.EnforceAccess(policy.RestoreWrite, handler.GetRestoreAppsStatus))
	r.Path("/api/v1/snapshot/{backup}/logs").Methods("GET").
		HandlerFunc(middleware.EnforceAccess(policy.BackupRead, handler.DownloadSnapshotLogs))
	r.Path("/api/v1/velero").Methods("GET").
		HandlerFunc(middleware.EnforceAccess(policy.BackupRead, handler.GetVeleroStatus))

	// KURL
	r.HandleFunc("/api/v1/kurl", NotImplemented) // I'm not sure why this is here
	r.Path("/api/v1/kurl/generate-node-join-command-worker").Methods("POST").
		HandlerFunc(middleware.EnforceAccess(policy.ClusterWrite, handler.GenerateNodeJoinCommandWorker))
	r.Path("/api/v1/kurl/generate-node-join-command-master").Methods("POST").
		HandlerFunc(middleware.EnforceAccess(policy.ClusterWrite, handler.GenerateNodeJoinCommandMaster))
	r.Path("/api/v1/kurl/nodes/{nodeName}/drain").Methods("POST").
		HandlerFunc(middleware.EnforceAccess(policy.ClusterWrite, handler.DrainNode))
	r.Path("/api/v1/kurl/nodes/{nodeName}").Methods("DELETE").
		HandlerFunc(middleware.EnforceAccess(policy.ClusterWrite, handler.DeleteNode))
	r.Path("/api/v1/kurl/nodes").Methods("GET").
		HandlerFunc(middleware.EnforceAccess(policy.ClusterRead, handler.GetKurlNodes))

	// Prometheus
	r.Path("/api/v1/prometheus").Methods("POST").
		HandlerFunc(middleware.EnforceAccess(policy.PrometheussettingsWrite, handler.SetPrometheusAddress))

	// GitOps
	r.Path("/api/v1/gitops/app/{appId}/cluster/{clusterId}/update").Methods("PUT").
		HandlerFunc(middleware.EnforceAccess(policy.AppGitopsWrite, handler.UpdateAppGitOps))
	r.Path("/api/v1/gitops/app/{appId}/cluster/{clusterId}/disable").Methods("POST").
		HandlerFunc(middleware.EnforceAccess(policy.AppGitopsWrite, handler.DisableAppGitOps))
	r.Path("/api/v1/gitops/app/{appId}/cluster/{clusterId}/initconnection").Methods("POST").
		HandlerFunc(middleware.EnforceAccess(policy.AppGitopsWrite, handler.InitGitOpsConnection))
	r.Path("/api/v1/gitops/create").Methods("POST").
		HandlerFunc(middleware.EnforceAccess(policy.GitopsWrite, handler.CreateGitOps))
	r.Path("/api/v1/gitops/reset").Methods("POST").
		HandlerFunc(middleware.EnforceAccess(policy.GitopsWrite, handler.ResetGitOps))
	r.Path("/api/v1/gitops/get").Methods("GET").
		HandlerFunc(middleware.EnforceAccess(policy.GitopsRead, handler.GetGitOpsRepo))
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
