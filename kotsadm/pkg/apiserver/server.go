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
	"github.com/replicatedhq/kots/kotsadm/pkg/snapshotscheduler"
	"github.com/replicatedhq/kots/kotsadm/pkg/socketservice"
	"github.com/replicatedhq/kots/kotsadm/pkg/store"
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

	if err := informers.Start(); err != nil {
		log.Println("Failed to start informers", err)
	}

	if err := updatechecker.Start(); err != nil {
		log.Println("Failed to start update checker", err)
	}

	if err := snapshotscheduler.Start(); err != nil {
		log.Println("Failed to start snapshot scheduler", err)
	}

	if err := automation.AutomateInstall(); err != nil {
		log.Println("Failed to run automated installs", err)
	}

	u, err := url.Parse("http://kotsadm-api-node:3000")
	if err != nil {
		panic(err)
	}
	upstream := httputil.NewSingleHostReverseProxy(u)

	r := mux.NewRouter()

	r.Use(handlers.CorsMiddleware)
	r.Methods("OPTIONS").HandlerFunc(handlers.CORS)

	// proxy all graphql requests
	r.Path("/graphql").Methods("POST").HandlerFunc(handlers.NodeProxy(upstream))

	/**********************************************************************
	* Unauthenticated routes
	**********************************************************************/

	r.HandleFunc("/healthz", handlers.Healthz)
	r.HandleFunc("/api/v1/login", handlers.Login)
	r.HandleFunc("/api/v1/logout", handlers.Logout) // this route uses its own auth
	r.Path("/api/v1/metadata").Methods("GET").HandlerFunc(handlers.Metadata)

	r.Path("/api/v1/troubleshoot").Methods("GET").HandlerFunc(handlers.GetDefaultTroubleshoot)
	r.Path("/api/v1/troubleshoot/{appSlug}").Methods("GET").HandlerFunc(handlers.GetTroubleshoot)
	r.Path("/api/v1/troubleshoot/{appId}/{bundleId}").Methods("PUT").HandlerFunc(handlers.UploadSupportBundle)
	r.Path("/api/v1/troubleshoot/supportbundle/{bundleId}/redactions").Methods("PUT").HandlerFunc(handlers.SetSupportBundleRedactions)
	r.Path("/api/v1/preflight/app/{appSlug}/sequence/{sequence}").Methods("GET").HandlerFunc(handlers.GetPreflightStatus)
	r.Path("/api/v1/preflight/app/{appSlug}/sequence/{sequence}").Methods("POST").HandlerFunc(handlers.PostPreflightStatus)

	/**********************************************************************
	* Cluster auth routes (functions that the operator calls)
	**********************************************************************/

	r.Path("/api/v1/appstatus").Methods("PUT").HandlerFunc(handlers.SetAppStatus)
	r.Path("/api/v1/deploy/result").Methods("PUT").HandlerFunc(handlers.UpdateDeployResult)
	r.Path("/api/v1/undeploy/result").Methods("PUT").HandlerFunc(handlers.UpdateUndeployResult)
	r.Handle("/socket.io/", socketservice.Start().Server)

	/**********************************************************************
	* KOTS token auth routes
	**********************************************************************/

	r.Path("/api/v1/kots/ports").Methods("GET").HandlerFunc(handlers.GetApplicationPorts)
	r.Path("/api/v1/upload").Methods("PUT").HandlerFunc(handlers.UploadExistingApp)
	r.Path("/api/v1/download").Methods("GET").HandlerFunc(handlers.DownloadApp)

	/**********************************************************************
	* Session auth routes
	**********************************************************************/

	sessionAuthQuietRouter := r.PathPrefix("").Subrouter()
	sessionAuthQuietRouter.Use(handlers.RequireValidSessionQuietMiddleware)

	sessionAuthQuietRouter.Path("/api/v1/ping").Methods("GET").HandlerFunc(handlers.Ping)

	sessionAuthRouter := r.PathPrefix("").Subrouter()
	sessionAuthRouter.Use(handlers.RequireValidSessionMiddleware)

	// Support Bundles
	sessionAuthRouter.Path("/api/v1/troubleshoot/supportbundle/{bundleSlug}").Methods("GET").HandlerFunc(handlers.GetSupportBundle)
	sessionAuthRouter.Path("/api/v1/troubleshoot/app/{appSlug}/supportbundles").Methods("GET").HandlerFunc(handlers.ListSupportBundles)
	sessionAuthRouter.Path("/api/v1/troubleshoot/app/{appSlug}/supportbundlecommand").Methods("GET").HandlerFunc(handlers.GetSupportBundleCommand)
	sessionAuthRouter.Path("/api/v1/troubleshoot/supportbundle/{bundleId}/files").Methods("GET").HandlerFunc(handlers.GetSupportBundleFiles)
	sessionAuthRouter.Path("/api/v1/troubleshoot/supportbundle/{bundleId}/redactions").Methods("GET").HandlerFunc(handlers.GetSupportBundleRedactions)
	sessionAuthRouter.Path("/api/v1/troubleshoot/supportbundle/{bundleId}/download").Methods("GET").HandlerFunc(handlers.DownloadSupportBundle)
	sessionAuthRouter.Path("/api/v1/troubleshoot/supportbundle/app/{appId}/cluster/{clusterId}/collect").Methods("POST").HandlerFunc(handlers.CollectSupportBundle)
	sessionAuthRouter.Path("/api/v1/troubleshoot/analyzebundle/{bundleId}").Methods("POST").HandlerFunc(handlers.NodeProxy(upstream))

	// redactor routes
	sessionAuthRouter.Path("/api/v1/redact/set").Methods("PUT").HandlerFunc(handlers.UpdateRedact)
	sessionAuthRouter.Path("/api/v1/redact/get").Methods("GET").HandlerFunc(handlers.GetRedact)
	sessionAuthRouter.Path("/api/v1/redacts").Methods("GET").HandlerFunc(handlers.ListRedactors)
	sessionAuthRouter.Path("/api/v1/redact/spec/{slug}").Methods("GET").HandlerFunc(handlers.GetRedactMetadataAndYaml)
	sessionAuthRouter.Path("/api/v1/redact/spec/{slug}").Methods("POST").HandlerFunc(handlers.SetRedactMetadataAndYaml)
	sessionAuthRouter.Path("/api/v1/redact/spec/{slug}").Methods("DELETE").HandlerFunc(handlers.DeleteRedact)
	sessionAuthRouter.Path("/api/v1/redact/enabled/{slug}").Methods("POST").HandlerFunc(handlers.SetRedactEnabled)

	// This the handler for license API and should be called by the application only.
	sessionAuthRouter.Path("/license/v1/license").Methods("GET").HandlerFunc(handlers.GetPlatformLicenseCompatibility)

	// Apps
	sessionAuthRouter.Path("/api/v1/apps").Methods("GET").HandlerFunc(handlers.ListApps)
	sessionAuthRouter.Path("/api/v1/apps/app/{appSlug}").Methods("GET").HandlerFunc(handlers.GetApp)
	sessionAuthRouter.Path("/api/v1/app/{appSlug}/versions").Methods("GET").HandlerFunc(handlers.GetAppVersionHistory)

	// Airgap
	sessionAuthRouter.Path("/api/v1/app/airgap/bundleexists/{identifier}").Methods("GET").HandlerFunc(handlers.AirgapBundleExists)
	sessionAuthRouter.Path("/api/v1/app/airgap/processbundle/{identifier}").Methods("POST", "PUT").HandlerFunc(handlers.ProcessAirgapBundle)
	sessionAuthRouter.Path("/api/v1/app/airgap/chunk").Methods("GET").HandlerFunc(handlers.CheckAirgapBundleChunk)
	sessionAuthRouter.Path("/api/v1/app/airgap/chunk").Methods("POST").HandlerFunc(handlers.UploadAirgapBundleChunk)
	sessionAuthRouter.Path("/api/v1/app/airgap/status").Methods("GET").HandlerFunc(handlers.GetAirgapInstallStatus)
	sessionAuthRouter.Path("/api/v1/kots/airgap/reset/{appSlug}").Methods("POST").HandlerFunc(handlers.ResetAirgapInstallStatus)

	// Implemented handlers
	sessionAuthRouter.Path("/api/v1/license/platform").Methods("POST").HandlerFunc(handlers.ExchangePlatformLicense)
	sessionAuthRouter.Path("/api/v1/app/{appSlug}/sequence/{sequence}/preflight/ignore-rbac").Methods("POST").HandlerFunc(handlers.IgnorePreflightRBACErrors)
	sessionAuthRouter.Path("/api/v1/app/{appSlug}/sequence/{sequence}/preflight/run").Methods("POST").HandlerFunc(handlers.StartPreflightChecks)
	sessionAuthRouter.Path("/api/v1/app/{appSlug}/sequence/{sequence}/preflight/result").Methods("GET").HandlerFunc(handlers.GetPreflightResult)
	sessionAuthRouter.Path("/api/v1/app/{appSlug}/sequence/{sequence}/preflightcommand").Methods("GET").HandlerFunc(handlers.GetPreflightCommand)
	sessionAuthRouter.Path("/api/v1/preflight/result").Methods("GET").HandlerFunc(handlers.GetLatestPreflightResult)
	sessionAuthRouter.Path("/api/v1/app/{appSlug}/sequence/{sequence}/deploy").Methods("POST").HandlerFunc(handlers.DeployAppVersion)
	sessionAuthRouter.Path("/api/v1/app/{appSlug}/sequence/{sequence}/renderedcontents").Methods("GET").HandlerFunc(handlers.GetAppRenderedContents)
	sessionAuthRouter.Path("/api/v1/app/{appSlug}/sequence/{sequence}/contents").Methods("GET").HandlerFunc(handlers.GetAppContents)
	sessionAuthRouter.Path("/api/v1/app/{appSlug}/cluster/{clusterId}/dashboard").Methods("GET").HandlerFunc(handlers.GetAppDashboard)
	sessionAuthRouter.Path("/api/v1/app/{appSlug}/cluster/{clusterId}/sequence/{sequence}/downstreamoutput").Methods("GET").HandlerFunc(handlers.GetDownstreamOutput)

	// Installation
	sessionAuthRouter.Path("/api/v1/license").Methods("POST").HandlerFunc(handlers.UploadNewLicense)
	sessionAuthRouter.Path("/api/v1/license/resume").Methods("PUT").HandlerFunc(handlers.ResumeInstallOnline)

	sessionAuthRouter.Path("/api/v1/registry").Methods("GET").HandlerFunc(handlers.GetKotsadmRegistry)
	sessionAuthRouter.Path("/api/v1/imagerewritestatus").Methods("GET").HandlerFunc(handlers.GetImageRewriteStatus)

	sessionAuthRouter.Path("/api/v1/app/online/status").Methods("GET").HandlerFunc(handlers.GetOnlineInstallStatus)
	sessionAuthRouter.Path("/api/v1/app/{appSlug}/registry").Methods("PUT").HandlerFunc(handlers.UpdateAppRegistry)
	sessionAuthRouter.Path("/api/v1/app/{appSlug}/registry").Methods("GET").HandlerFunc(handlers.GetAppRegistry)
	sessionAuthRouter.Path("/api/v1/app/{appSlug}/registry/validate").Methods("POST").HandlerFunc(handlers.ValidateAppRegistry)
	sessionAuthRouter.Path("/api/v1/app/{appSlug}/config").Methods("PUT").HandlerFunc(handlers.UpdateAppConfig)
	sessionAuthRouter.Path("/api/v1/app/{appSlug}/config/{sequence}").Methods("GET").HandlerFunc(handlers.CurrentAppConfig)
	sessionAuthRouter.Path("/api/v1/app/{appSlug}/liveconfig").Methods("POST").HandlerFunc(handlers.LiveAppConfig)
	sessionAuthRouter.Path("/api/v1/app/{appSlug}/license").Methods("PUT").HandlerFunc(handlers.SyncLicense)
	sessionAuthRouter.Path("/api/v1/app/{appSlug}/license").Methods("GET").HandlerFunc(handlers.GetLicense)
	sessionAuthRouter.Path("/api/v1/app/{appSlug}/updatecheck").Methods("POST").HandlerFunc(handlers.AppUpdateCheck)
	sessionAuthRouter.Path("/api/v1/app/{appSlug}/updatecheckerspec").Methods("PUT").HandlerFunc(handlers.UpdateCheckerSpec)

	// kotsadm snapshots
	sessionAuthRouter.Path("/api/v1/snapshots").Methods("GET").HandlerFunc(handlers.ListKotsadmBackups)
	sessionAuthRouter.Path("/api/v1/snapshot/{snapshotName}").Methods("GET").HandlerFunc(handlers.GetKotsadmBackup)
	sessionAuthRouter.Path("/api/v1/snapshot/{snapshotName}/delete").Methods("POST").HandlerFunc(handlers.DeleteKotsadmBackup)
	sessionAuthRouter.Path("/api/v1/velero").Methods("GET").HandlerFunc(handlers.GetVeleroStatus)

	// App snapshot routes
	sessionAuthRouter.Path("/api/v1/app/{appSlug}/snapshot/backup").Methods("POST").HandlerFunc(handlers.CreateBackup)
	sessionAuthRouter.Path("/api/v1/app/{appSlug}/snapshot/restore/status").Methods("GET").HandlerFunc(handlers.GetRestoreStatus)
	sessionAuthRouter.Path("/api/v1/app/{appSlug}/snapshot/restore").Methods("DELETE").HandlerFunc(handlers.CancelRestore)
	sessionAuthRouter.Path("/api/v1/app/{appSlug}/snapshot/restore/{restoreName}").Methods("GET").HandlerFunc(handlers.GetKotsadmRestore)
	sessionAuthRouter.Path("/api/v1/app/{appSlug}/snapshots").Methods("GET").HandlerFunc(handlers.ListBackups)
	sessionAuthRouter.Path("/api/v1/app/{appSlug}/snapshot/config").Methods("GET").HandlerFunc(handlers.GetSnapshotConfig)
	sessionAuthRouter.Path("/api/v1/app/{appSlug}/snapshot/config").Methods("PUT").HandlerFunc(handlers.SaveSnapshotConfig)

	// Global snapshot routes
	sessionAuthRouter.Path("/api/v1/snapshots/settings").Methods("GET").HandlerFunc(handlers.GetGlobalSnapshotSettings)
	sessionAuthRouter.Path("/api/v1/snapshots/settings").Methods("PUT").HandlerFunc(handlers.UpdateGlobalSnapshotSettings)
	sessionAuthRouter.Path("/api/v1/snapshot/{snapshotName}/restore").Methods("POST").HandlerFunc(handlers.CreateRestore)

	// Find a home snapshot routes
	sessionAuthRouter.Path("/api/v1/snapshot/{backup}/logs").Methods("GET").HandlerFunc(handlers.DownloadSnapshotLogs)

	// KURL
	sessionAuthRouter.HandleFunc("/api/v1/kurl", handlers.NotImplemented)
	sessionAuthRouter.Path("/api/v1/kurl/generate-node-join-command-worker").Methods("POST").HandlerFunc(handlers.GenerateNodeJoinCommandWorker)
	sessionAuthRouter.Path("/api/v1/kurl/generate-node-join-command-master").Methods("POST").HandlerFunc(handlers.GenerateNodeJoinCommandMaster)
	sessionAuthRouter.Path("/api/v1/kurl/nodes/{nodeName}/drain").Methods("POST").HandlerFunc(handlers.DrainNode)
	sessionAuthRouter.Path("/api/v1/kurl/nodes/{nodeName}").Methods("DELETE").HandlerFunc(handlers.DeleteNode)
	sessionAuthRouter.Path("/api/v1/kurl/nodes").Methods("GET").HandlerFunc(handlers.GetKurlNodes)

	// Prometheus
	sessionAuthRouter.Path("/api/v1/prometheus").Methods("POST").HandlerFunc(handlers.SetPrometheusAddress)

	// GitOps
	sessionAuthRouter.Path("/api/v1/gitops/app/{appId}/cluster/{clusterId}/update").Methods("PUT").HandlerFunc(handlers.UpdateAppGitOps)
	sessionAuthRouter.Path("/api/v1/gitops/app/{appId}/cluster/{clusterId}/disable").Methods("POST").HandlerFunc(handlers.DisableAppGitOps)
	sessionAuthRouter.Path("/api/v1/gitops/app/{appId}/cluster/{clusterId}/initconnection").Methods("POST").HandlerFunc(handlers.InitGitOpsConnection)
	sessionAuthRouter.Path("/api/v1/gitops/create").Methods("POST").HandlerFunc(handlers.CreateGitOps)
	sessionAuthRouter.Path("/api/v1/gitops/reset").Methods("POST").HandlerFunc(handlers.ResetGitOps)
	sessionAuthRouter.Path("/api/v1/gitops/get").Methods("GET").HandlerFunc(handlers.GetGitOpsRepo)

	// task status
	sessionAuthRouter.Path("/api/v1/task/updatedownload").Methods("GET").HandlerFunc(handlers.GetUpdateDownloadStatus)

	/**********************************************************************
	* Static routes
	**********************************************************************/

	// to avoid confusion, we don't serve this in the dev env...
	if os.Getenv("DISABLE_SPA_SERVING") != "1" {
		spa := handlers.SPAHandler{StaticPath: filepath.Join("web", "dist"), IndexPath: "index.html"}
		r.PathPrefix("/").Handler(spa)
	}

	srv := &http.Server{
		Handler: r,
		Addr:    ":3000",
	}

	fmt.Printf("Starting kotsadm API on port %d...\n", 3000)

	log.Fatal(srv.ListenAndServe())
}
