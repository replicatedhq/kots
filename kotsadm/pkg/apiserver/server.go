package apiserver

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"path/filepath"
	"time"

	"github.com/gorilla/mux"
	"github.com/replicatedhq/kots/kotsadm/pkg/automation"
	"github.com/replicatedhq/kots/kotsadm/pkg/handlers"
	"github.com/replicatedhq/kots/kotsadm/pkg/informers"
	"github.com/replicatedhq/kots/kotsadm/pkg/policy"
	"github.com/replicatedhq/kots/kotsadm/pkg/snapshotscheduler"
	"github.com/replicatedhq/kots/kotsadm/pkg/socketservice"
	"github.com/replicatedhq/kots/kotsadm/pkg/store"
	"github.com/replicatedhq/kots/kotsadm/pkg/supportbundle"
	"github.com/replicatedhq/kots/kotsadm/pkg/updatechecker"
)

func Start() {
	log.Printf("kotsadm version %s\n", os.Getenv("VERSION"))

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Minute)
	if err := store.GetStore().WaitForReady(ctx); err != nil {
		panic(err)
	}
	cancel()

	if err := bootstrap(); err != nil {
		panic(err)
	}

	store.GetStore().RunMigrations()

	supportbundle.StartServer()

	if err := informers.Start(); err != nil {
		log.Println("Failed to start informers", err)
	}

	if err := updatechecker.Start(); err != nil {
		log.Println("Failed to start update checker", err)
	}

	if err := snapshotscheduler.Start(); err != nil {
		log.Println("Failed to start snapshot scheduler", err)
	}

	waitForAirgap, err := automation.NeedToWaitForAirgapApp()
	if err != nil {
		log.Println("Failed to check if airgap install is in progress", err)
	} else if !waitForAirgap {
		if err := automation.AutomateInstall(); err != nil {
			log.Println("Failed to run automated installs", err)
		}
	}

	r := mux.NewRouter()

	r.Use(handlers.CorsMiddleware)
	r.Methods("OPTIONS").HandlerFunc(handlers.CORS)

	/**********************************************************************
	* Unauthenticated routes
	**********************************************************************/

	r.HandleFunc("/healthz", handlers.Healthz)
	r.HandleFunc("/api/v1/login", handlers.Login)
	r.HandleFunc("/api/v1/logout", handlers.Logout) // this route uses its own auth
	r.Path("/api/v1/metadata").Methods("GET").HandlerFunc(handlers.Metadata)

	r.HandleFunc("/api/v1/oidc/login", handlers.OIDCLogin)
	r.HandleFunc("/api/v1/oidc/login/callback", handlers.OIDCLoginCallback)

	r.Path("/api/v1/troubleshoot/{appId}/{bundleId}").Methods("PUT").HandlerFunc(handlers.UploadSupportBundle)
	r.Path("/api/v1/troubleshoot/supportbundle/{bundleId}/redactions").Methods("PUT").HandlerFunc(handlers.SetSupportBundleRedactions)
	r.Path("/api/v1/preflight/app/{appSlug}/sequence/{sequence}").Methods("POST").HandlerFunc(handlers.PostPreflightStatus)

	// This the handler for license API and should be called by the application only.
	r.Path("/license/v1/license").Methods("GET").HandlerFunc(handlers.GetPlatformLicenseCompatibility)

	/**********************************************************************
	* Cluster auth routes (functions that the operator calls)
	**********************************************************************/

	r.Path("/api/v1/appstatus").Methods("PUT").HandlerFunc(handlers.SetAppStatus)
	r.Path("/api/v1/deploy/result").Methods("PUT").HandlerFunc(handlers.UpdateDeployResult)
	r.Path("/api/v1/undeploy/result").Methods("PUT").HandlerFunc(handlers.UpdateUndeployResult)
	r.Handle("/socket.io/", socketservice.Start())

	/**********************************************************************
	* KOTS token auth routes
	**********************************************************************/

	r.Path("/api/v1/kots/ports").Methods("GET").HandlerFunc(handlers.GetApplicationPorts)
	r.Path("/api/v1/upload").Methods("PUT").HandlerFunc(handlers.UploadExistingApp)
	r.Path("/api/v1/download").Methods("GET").HandlerFunc(handlers.DownloadApp)
	r.Path("/api/v1/airgap/install").Methods("POST").HandlerFunc(handlers.UploadInitialAirgapApp)

	/**********************************************************************
	* Session auth routes
	**********************************************************************/

	sessionAuthQuietRouter := r.PathPrefix("").Subrouter()
	sessionAuthQuietRouter.Use(handlers.RequireValidSessionQuietMiddleware)

	sessionAuthQuietRouter.Path("/api/v1/ping").Methods("GET").HandlerFunc(handlers.Ping)

	sessionAuthRouter := r.PathPrefix("").Subrouter()
	sessionAuthRouter.Use(handlers.RequireValidSessionMiddleware)

	// Support Bundles
	sessionAuthRouter.Path("/api/v1/troubleshoot/supportbundle/{bundleSlug}").Methods("GET").
		HandlerFunc(policy.AppSupportbundleRead.Enforce(handlers.GetSupportBundle)) // TODO: appSlug
	sessionAuthRouter.Path("/api/v1/troubleshoot/app/{appSlug}/supportbundles").Methods("GET").
		HandlerFunc(policy.AppSupportbundleRead.Enforce(handlers.ListSupportBundles))
	sessionAuthRouter.Path("/api/v1/troubleshoot/app/{appSlug}/supportbundlecommand").Methods("POST").
		HandlerFunc(policy.AppSupportbundleRead.Enforce(handlers.GetSupportBundleCommand))
	sessionAuthRouter.Path("/api/v1/troubleshoot/supportbundle/{bundleId}/files").Methods("GET").
		HandlerFunc(policy.AppSupportbundleRead.Enforce(handlers.GetSupportBundleFiles)) // TODO: appSlug
	sessionAuthRouter.Path("/api/v1/troubleshoot/supportbundle/{bundleId}/redactions").Methods("GET").
		HandlerFunc(policy.AppSupportbundleRead.Enforce(handlers.GetSupportBundleRedactions)) // TODO: appSlug
	sessionAuthRouter.Path("/api/v1/troubleshoot/supportbundle/{bundleId}/download").Methods("GET").
		HandlerFunc(policy.AppSupportbundleRead.Enforce(handlers.DownloadSupportBundle)) // TODO: appSlug
	sessionAuthRouter.Path("/api/v1/troubleshoot/supportbundle/app/{appId}/cluster/{clusterId}/collect").Methods("POST").
		HandlerFunc(policy.AppSupportbundleWrite.Enforce(handlers.CollectSupportBundle))

	// redactor routes
	sessionAuthRouter.Path("/api/v1/redact/set").Methods("PUT").
		HandlerFunc(policy.RedactorWrite.Enforce(handlers.UpdateRedact))
	sessionAuthRouter.Path("/api/v1/redact/get").Methods("GET").
		HandlerFunc(policy.RedactorRead.Enforce(handlers.GetRedact))
	sessionAuthRouter.Path("/api/v1/redacts").Methods("GET").
		HandlerFunc(policy.RedactorRead.Enforce(handlers.ListRedactors))
	sessionAuthRouter.Path("/api/v1/redact/spec/{slug}").Methods("GET").
		HandlerFunc(policy.RedactorRead.Enforce(handlers.GetRedactMetadataAndYaml))
	sessionAuthRouter.Path("/api/v1/redact/spec/{slug}").Methods("POST").
		HandlerFunc(policy.RedactorWrite.Enforce(handlers.SetRedactMetadataAndYaml))
	sessionAuthRouter.Path("/api/v1/redact/spec/{slug}").Methods("DELETE").
		HandlerFunc(policy.RedactorWrite.Enforce(handlers.DeleteRedact))
	sessionAuthRouter.Path("/api/v1/redact/enabled/{slug}").Methods("POST").
		HandlerFunc(policy.RedactorWrite.Enforce(handlers.SetRedactEnabled))

	// Apps
	sessionAuthRouter.Path("/api/v1/apps").Methods("GET").
		HandlerFunc(policy.AppList.Enforce(handlers.ListApps))
	sessionAuthRouter.Path("/api/v1/app/{appSlug}").Methods("GET").
		HandlerFunc(policy.AppRead.Enforce(handlers.GetApp))
	sessionAuthRouter.Path("/api/v1/app/{appSlug}/versions").Methods("GET").
		HandlerFunc(policy.AppDownstreamRead.Enforce(handlers.GetAppVersionHistory))
	sessionAuthRouter.Path("/api/v1/app/{appSlug}/task/updatedownload").Methods("GET").
		HandlerFunc(policy.AppRead.Enforce(handlers.GetUpdateDownloadStatus)) // NOTE: appSlug is unused

	// Airgap
	sessionAuthRouter.Path("/api/v1/app/{appSlug}/airgap/bundleprogress/{identifier}/{totalChunks}").Methods("GET").
		HandlerFunc(policy.AppDownstreamWrite.Enforce(handlers.AirgapBundleProgress))
	sessionAuthRouter.Path("/api/v1/app/{appSlug}/airgap/bundleexists/{identifier}/{totalChunks}").Methods("GET").
		HandlerFunc(policy.AppDownstreamWrite.Enforce(handlers.AirgapBundleExists))
	sessionAuthRouter.Path("/api/v1/app/{appSlug}/airgap/processbundle/{identifier}/{totalChunks}").Methods("POST").
		HandlerFunc(policy.AppDownstreamWrite.Enforce(handlers.CreateAppFromAirgap))
	sessionAuthRouter.Path("/api/v1/app/{appSlug}/airgap/processbundle/{identifier}/{totalChunks}").Methods("PUT").
		HandlerFunc(policy.AppDownstreamWrite.Enforce(handlers.UpdateAppFromAirgap))
	sessionAuthRouter.Path("/api/v1/app/{appSlug}/airgap/chunk").Methods("GET").
		HandlerFunc(policy.AppDownstreamWrite.Enforce(handlers.CheckAirgapBundleChunk))
	sessionAuthRouter.Path("/api/v1/app/{appSlug}/airgap/chunk").Methods("POST").
		HandlerFunc(policy.AppDownstreamWrite.Enforce(handlers.UploadAirgapBundleChunk))
	sessionAuthRouter.Path("/api/v1/app/{appSlug}/airgap/status").Methods("GET").
		HandlerFunc(policy.AppDownstreamWrite.Enforce(handlers.GetAirgapInstallStatus))
	sessionAuthRouter.Path("/api/v1/app/{appSlug}/airgap/reset").Methods("POST").
		HandlerFunc(policy.AppDownstreamWrite.Enforce(handlers.ResetAirgapInstallStatus))

	// Implemented handlers
	sessionAuthRouter.Path("/api/v1/app/{appSlug}/sequence/{sequence}/preflight/ignore-rbac").Methods("POST").
		HandlerFunc(policy.AppDownstreamPreflightWrite.Enforce(handlers.IgnorePreflightRBACErrors))
	sessionAuthRouter.Path("/api/v1/app/{appSlug}/sequence/{sequence}/preflight/run").Methods("POST").
		HandlerFunc(policy.AppDownstreamPreflightWrite.Enforce(handlers.StartPreflightChecks))
	sessionAuthRouter.Path("/api/v1/app/{appSlug}/preflight/result").Methods("GET").
		HandlerFunc(policy.AppDownstreamPreflightRead.Enforce(handlers.GetLatestPreflightResultsForSequenceZero))
	sessionAuthRouter.Path("/api/v1/app/{appSlug}/sequence/{sequence}/preflight/result").Methods("GET").
		HandlerFunc(policy.AppDownstreamPreflightRead.Enforce(handlers.GetPreflightResult))
	sessionAuthRouter.Path("/api/v1/app/{appSlug}/sequence/{sequence}/preflightcommand").Methods("POST").
		HandlerFunc(policy.AppRead.Enforce(handlers.GetPreflightCommand)) // this is intentionally policy.AppRead

	sessionAuthRouter.Path("/api/v1/app/{appSlug}/sequence/{sequence}/deploy").Methods("POST").
		HandlerFunc(policy.AppDownstreamWrite.Enforce(handlers.DeployAppVersion))
	sessionAuthRouter.Path("/api/v1/app/{appSlug}/sequence/{sequence}/redeploy").Methods("POST").
		HandlerFunc(policy.AppDownstreamWrite.Enforce(handlers.RedeployAppVersion))
	sessionAuthRouter.Path("/api/v1/app/{appSlug}/sequence/{sequence}/renderedcontents").Methods("GET").
		HandlerFunc(policy.AppDownstreamFiletreeRead.Enforce(handlers.GetAppRenderedContents))
	sessionAuthRouter.Path("/api/v1/app/{appSlug}/sequence/{sequence}/contents").Methods("GET").
		HandlerFunc(policy.AppDownstreamFiletreeRead.Enforce(handlers.GetAppContents))
	sessionAuthRouter.Path("/api/v1/app/{appSlug}/cluster/{clusterId}/dashboard").Methods("GET").
		HandlerFunc(policy.AppRead.Enforce(handlers.GetAppDashboard))
	sessionAuthRouter.Path("/api/v1/app/{appSlug}/cluster/{clusterId}/sequence/{sequence}/downstreamoutput").Methods("GET").
		HandlerFunc(policy.AppDownstreamLogsRead.Enforce(handlers.GetDownstreamOutput))

	// Installation
	sessionAuthRouter.Path("/api/v1/license").Methods("POST").
		HandlerFunc(policy.AppCreate.Enforce(handlers.UploadNewLicense))
	sessionAuthRouter.Path("/api/v1/license/platform").Methods("POST").
		HandlerFunc(policy.AppCreate.Enforce(handlers.ExchangePlatformLicense))
	sessionAuthRouter.Path("/api/v1/license/resume").Methods("PUT").
		HandlerFunc(policy.AppCreate.Enforce(handlers.ResumeInstallOnline))
	sessionAuthRouter.Path("/api/v1/app/online/status").Methods("GET").
		HandlerFunc(policy.AppCreate.Enforce(handlers.GetOnlineInstallStatus))

	sessionAuthRouter.Path("/api/v1/registry").Methods("GET").
		HandlerFunc(policy.RegistryRead.Enforce(handlers.GetKotsadmRegistry))
	sessionAuthRouter.Path("/api/v1/imagerewritestatus").Methods("GET").
		HandlerFunc(policy.RegistryRead.Enforce(handlers.GetImageRewriteStatus))

	sessionAuthRouter.Path("/api/v1/app/{appSlug}/registry").Methods("PUT").
		HandlerFunc(policy.AppRegistryWrite.Enforce(handlers.UpdateAppRegistry))
	sessionAuthRouter.Path("/api/v1/app/{appSlug}/registry").Methods("GET").
		HandlerFunc(policy.AppRegistryRead.Enforce(handlers.GetAppRegistry))
	sessionAuthRouter.Path("/api/v1/app/{appSlug}/imagerewritestatus").Methods("GET").
		HandlerFunc(policy.AppRegistryRead.Enforce(handlers.GetImageRewriteStatus))
	sessionAuthRouter.Path("/api/v1/app/{appSlug}/registry/validate").Methods("POST").
		HandlerFunc(policy.AppRegistryWrite.Enforce(handlers.ValidateAppRegistry))

	sessionAuthRouter.Path("/api/v1/app/{appSlug}/config").Methods("PUT").
		HandlerFunc(policy.AppDownstreamConfigWrite.Enforce(handlers.UpdateAppConfig))
	sessionAuthRouter.Path("/api/v1/app/{appSlug}/config/{sequence}").Methods("GET").
		HandlerFunc(policy.AppDownstreamConfigRead.Enforce(handlers.CurrentAppConfig))
	sessionAuthRouter.Path("/api/v1/app/{appSlug}/liveconfig").Methods("POST").
		HandlerFunc(policy.AppDownstreamConfigWrite.Enforce(handlers.LiveAppConfig))

	sessionAuthRouter.Path("/api/v1/app/{appSlug}/license").Methods("PUT").
		HandlerFunc(policy.AppLicenseWrite.Enforce(handlers.SyncLicense))
	sessionAuthRouter.Path("/api/v1/app/{appSlug}/license").Methods("GET").
		HandlerFunc(policy.AppLicenseRead.Enforce(handlers.GetLicense))

	sessionAuthRouter.Path("/api/v1/app/{appSlug}/updatecheck").Methods("POST").
		HandlerFunc(policy.AppDownstreamWrite.Enforce(handlers.AppUpdateCheck))
	sessionAuthRouter.Path("/api/v1/app/{appSlug}/updatecheckerspec").Methods("PUT").
		HandlerFunc(policy.AppDownstreamWrite.Enforce(handlers.UpdateCheckerSpec))
	sessionAuthRouter.Path("/api/v1/app/{appSlug}/remove").Methods("POST").
		HandlerFunc(policy.AppUpdate.Enforce(handlers.RemoveApp))

	// App snapshot routes
	sessionAuthRouter.Path("/api/v1/app/{appSlug}/snapshot/backup").Methods("POST").
		HandlerFunc(policy.AppBackupWrite.Enforce(handlers.CreateApplicationBackup))
	sessionAuthRouter.Path("/api/v1/app/{appSlug}/snapshot/restore/status").Methods("GET").
		HandlerFunc(policy.AppRestoreRead.Enforce(handlers.GetRestoreStatus))
	sessionAuthRouter.Path("/api/v1/app/{appSlug}/snapshot/restore").Methods("DELETE").
		HandlerFunc(policy.AppRestoreWrite.Enforce(handlers.CancelRestore))
	sessionAuthRouter.Path("/api/v1/app/{appSlug}/snapshot/restore/{snapshotName}").Methods("POST").
		HandlerFunc(policy.AppRestoreWrite.Enforce(handlers.CreateApplicationRestore))
	sessionAuthRouter.Path("/api/v1/app/{appSlug}/snapshot/restore/{restoreName}").Methods("GET").
		HandlerFunc(policy.AppRestoreRead.Enforce(handlers.GetRestoreDetails))
	sessionAuthRouter.Path("/api/v1/app/{appSlug}/snapshots").Methods("GET").
		HandlerFunc(policy.AppBackupRead.Enforce(handlers.ListBackups))
	sessionAuthRouter.Path("/api/v1/app/{appSlug}/snapshot/config").Methods("GET").
		HandlerFunc(policy.AppSnapshotsettingsRead.Enforce(handlers.GetSnapshotConfig))
	sessionAuthRouter.Path("/api/v1/app/{appSlug}/snapshot/config").Methods("PUT").
		HandlerFunc(policy.AppSnapshotsettingsWrite.Enforce(handlers.SaveSnapshotConfig))

	// Global snapshot routes
	sessionAuthRouter.Path("/api/v1/snapshots").Methods("GET").
		HandlerFunc(policy.BackupRead.Enforce(handlers.ListInstanceBackups))
	sessionAuthRouter.Path("/api/v1/snapshot/backup").Methods("POST").
		HandlerFunc(policy.BackupWrite.Enforce(handlers.CreateInstanceBackup))
	sessionAuthRouter.Path("/api/v1/snapshot/config").Methods("GET").
		HandlerFunc(policy.SnapshotsettingsRead.Enforce(handlers.GetInstanceSnapshotConfig))
	sessionAuthRouter.Path("/api/v1/snapshot/config").Methods("PUT").
		HandlerFunc(policy.SnapshotsettingsWrite.Enforce(handlers.SaveInstanceSnapshotConfig))
	sessionAuthRouter.Path("/api/v1/snapshots/settings").Methods("GET").
		HandlerFunc(policy.SnapshotsettingsRead.Enforce(handlers.GetGlobalSnapshotSettings))
	sessionAuthRouter.Path("/api/v1/snapshots/settings").Methods("PUT").
		HandlerFunc(policy.SnapshotsettingsWrite.Enforce(handlers.UpdateGlobalSnapshotSettings))
	sessionAuthRouter.Path("/api/v1/snapshot/{snapshotName}").Methods("GET").
		HandlerFunc(policy.BackupRead.Enforce(handlers.GetBackup))
	sessionAuthRouter.Path("/api/v1/snapshot/{snapshotName}/delete").Methods("POST").
		HandlerFunc(policy.BackupWrite.Enforce(handlers.DeleteBackup))
	sessionAuthRouter.Path("/api/v1/snapshot/{snapshotName}/restore-apps").Methods("POST").
		HandlerFunc(policy.RestoreWrite.Enforce(handlers.RestoreApps))
	sessionAuthRouter.Path("/api/v1/snapshot/{snapshotName}/apps-restore-status").Methods("GET").
		HandlerFunc(policy.RestoreWrite.Enforce(handlers.GetRestoreAppsStatus))
	sessionAuthRouter.Path("/api/v1/snapshot/{backup}/logs").Methods("GET").
		HandlerFunc(policy.BackupRead.Enforce(handlers.DownloadSnapshotLogs))
	sessionAuthRouter.Path("/api/v1/velero").Methods("GET").
		HandlerFunc(policy.BackupRead.Enforce(handlers.GetVeleroStatus))

	// KURL
	sessionAuthRouter.HandleFunc("/api/v1/kurl", handlers.NotImplemented) // I'm not sure why this is here
	sessionAuthRouter.Path("/api/v1/kurl/generate-node-join-command-worker").Methods("POST").
		HandlerFunc(policy.ClusterWrite.Enforce(handlers.GenerateNodeJoinCommandWorker))
	sessionAuthRouter.Path("/api/v1/kurl/generate-node-join-command-master").Methods("POST").
		HandlerFunc(policy.ClusterWrite.Enforce(handlers.GenerateNodeJoinCommandMaster))
	sessionAuthRouter.Path("/api/v1/kurl/nodes/{nodeName}/drain").Methods("POST").
		HandlerFunc(policy.ClusterWrite.Enforce(handlers.DrainNode))
	sessionAuthRouter.Path("/api/v1/kurl/nodes/{nodeName}").Methods("DELETE").
		HandlerFunc(policy.ClusterWrite.Enforce(handlers.DeleteNode))
	sessionAuthRouter.Path("/api/v1/kurl/nodes").Methods("GET").
		HandlerFunc(policy.ClusterRead.Enforce(handlers.GetKurlNodes))

	// Prometheus
	sessionAuthRouter.Path("/api/v1/prometheus").Methods("POST").
		HandlerFunc(policy.PrometheussettingsWrite.Enforce(handlers.SetPrometheusAddress))

	// GitOps
	sessionAuthRouter.Path("/api/v1/gitops/app/{appId}/cluster/{clusterId}/update").Methods("PUT").
		HandlerFunc(policy.AppGitopsWrite.Enforce(handlers.UpdateAppGitOps))
	sessionAuthRouter.Path("/api/v1/gitops/app/{appId}/cluster/{clusterId}/disable").Methods("POST").
		HandlerFunc(policy.AppGitopsWrite.Enforce(handlers.DisableAppGitOps))
	sessionAuthRouter.Path("/api/v1/gitops/app/{appId}/cluster/{clusterId}/initconnection").Methods("POST").
		HandlerFunc(policy.AppGitopsWrite.Enforce(handlers.InitGitOpsConnection))
	sessionAuthRouter.Path("/api/v1/gitops/create").Methods("POST").
		HandlerFunc(policy.GitopsWrite.Enforce(handlers.CreateGitOps))
	sessionAuthRouter.Path("/api/v1/gitops/reset").Methods("POST").
		HandlerFunc(policy.GitopsWrite.Enforce(handlers.ResetGitOps))
	sessionAuthRouter.Path("/api/v1/gitops/get").Methods("GET").
		HandlerFunc(policy.GitopsRead.Enforce(handlers.GetGitOpsRepo))

	/**********************************************************************
	* Static routes
	**********************************************************************/

	// to avoid confusion, we don't serve this in the dev env...
	if os.Getenv("DISABLE_SPA_SERVING") != "1" {
		spa := handlers.SPAHandler{StaticPath: filepath.Join("web", "dist"), IndexPath: "index.html"}
		r.PathPrefix("/").Handler(spa)
	} else if os.Getenv("ENABLE_WEB_PROXY") == "1" { // for dev env
		u, err := url.Parse("http://kotsadm-web:30000")
		if err != nil {
			panic(err)
		}
		upstream := httputil.NewSingleHostReverseProxy(u)
		webProxy := func(upstream *httputil.ReverseProxy) func(http.ResponseWriter, *http.Request) {
			return func(w http.ResponseWriter, r *http.Request) {
				r.Header.Set("X-Forwarded-Host", r.Header.Get("Host"))
				upstream.ServeHTTP(w, r)
			}
		}(upstream)
		r.PathPrefix("/").HandlerFunc(webProxy)
	}

	srv := &http.Server{
		Handler: r,
		Addr:    ":3000",
	}

	fmt.Printf("Starting kotsadm API on port %d...\n", 3000)

	log.Fatal(srv.ListenAndServe())
}
