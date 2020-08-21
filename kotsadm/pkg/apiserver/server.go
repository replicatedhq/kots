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
	"github.com/replicatedhq/kots/kotsadm/pkg/socketservice"
	"github.com/replicatedhq/kots/kotsadm/pkg/updatechecker"
)

func Start() {
	log.Printf("kotsadm version %s\n", os.Getenv("VERSION"))

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Minute)
	if err := waitForDependencies(ctx); err != nil {
		panic(err)
	}
	cancel()

	if err := informers.Start(); err != nil {
		log.Println("Failed to start informers", err)
	}

	if err := updatechecker.Start(); err != nil {
		log.Println("Failed to start update checker", err)
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
	r.Use(mux.CORSMethodMiddleware(r))

	r.HandleFunc("/healthz", handlers.Healthz)

	// proxy all graphql requests
	r.Path("/graphql").Methods("OPTIONS").HandlerFunc(handlers.CORS)
	r.Path("/graphql").Methods("POST").HandlerFunc(handlers.NodeProxy(upstream))

	// Api ping
	r.HandleFunc("/api/v1/ping", handlers.Ping)

	// Functions that the operator calls
	r.Path("/api/v1/appstatus").Methods("PUT").HandlerFunc(handlers.NodeProxy(upstream))
	r.Path("/api/v1/deploy/result").Methods("PUT").HandlerFunc(handlers.NodeProxy(upstream))

	// Functions that are not called by the browser
	r.Path("/api/v1/undeploy/result").Methods("PUT").HandlerFunc(handlers.NodeProxy(upstream))
	r.Path("/api/v1/preflight/app/{appSlug}/sequence/{sequence}").Methods("GET").HandlerFunc(handlers.GetPreflightStatus)
	r.Path("/api/v1/preflight/app/{appSlug}/sequence/{sequence}").Methods("POST").HandlerFunc(handlers.PostPreflightStatus)

	// Support Bundles
	r.Path("/api/v1/troubleshoot").Methods("OPTIONS", "GET").HandlerFunc(handlers.GetDefaultTroubleshoot)
	r.Path("/api/v1/troubleshoot/{appSlug}").Methods("OPTIONS", "GET").HandlerFunc(handlers.GetTroubleshoot)
	r.Path("/api/v1/troubleshoot/supportbundle/{bundleSlug}").Methods("OPTIONS", "GET").HandlerFunc(handlers.GetSupportBundle)
	r.Path("/api/v1/troubleshoot/{appId}/{bundleId}").Methods("OPTIONS", "PUT").HandlerFunc(handlers.UploadSupportBundle)
	r.Path("/api/v1/troubleshoot/app/{appSlug}/supportbundles").Methods("OPTIONS", "GET").HandlerFunc(handlers.ListSupportBundles)
	r.Path("/api/v1/troubleshoot/app/{appSlug}/supportbundlecommand").Methods("OPTIONS", "GET").HandlerFunc(handlers.GetSupportBundleCommand)
	r.Path("/api/v1/troubleshoot/supportbundle/{bundleId}/files").Methods("OPTIONS", "GET").HandlerFunc(handlers.GetSupportBundleFiles)
	r.Path("/api/v1/troubleshoot/supportbundle/{bundleId}/redactions").Methods("OPTIONS", "GET").HandlerFunc(handlers.GetSupportBundleRedactions)
	r.Path("/api/v1/troubleshoot/supportbundle/{bundleId}/redactions").Methods("PUT").HandlerFunc(handlers.SetSupportBundleRedactions)
	r.Path("/api/v1/troubleshoot/supportbundle/{bundleId}/download").Methods("OPTIONS", "GET").HandlerFunc(handlers.DownloadSupportBundle)
	r.Path("/api/v1/troubleshoot/supportbundle/app/{appId}/cluster/{clusterId}/collect").Methods("OPTIONS", "POST").HandlerFunc(handlers.CollectSupportBundle)
	r.Path("/api/v1/troubleshoot/analyzebundle/{bundleId}").Methods("POST").HandlerFunc(handlers.NodeProxy(upstream))

	// redactor routes
	r.Path("/api/v1/redact/set").Methods("OPTIONS", "PUT").HandlerFunc(handlers.UpdateRedact)
	r.Path("/api/v1/redact/get").Methods("OPTIONS", "GET").HandlerFunc(handlers.GetRedact)
	r.Path("/api/v1/redacts").Methods("OPTIONS", "GET").HandlerFunc(handlers.ListRedactors)
	r.Path("/api/v1/redact/spec/{slug}").Methods("OPTIONS", "GET").HandlerFunc(handlers.GetRedactMetadataAndYaml)
	r.Path("/api/v1/redact/spec/{slug}").Methods("POST").HandlerFunc(handlers.SetRedactMetadataAndYaml)
	r.Path("/api/v1/redact/spec/{slug}").Methods("DELETE").HandlerFunc(handlers.DeleteRedact)
	r.Path("/api/v1/redact/enabled/{slug}").Methods("OPTIONS", "POST").HandlerFunc(handlers.SetRedactEnabled)

	r.PathPrefix("/api/v1/kots/").Methods("OPTIONS").HandlerFunc(handlers.CORS)
	r.PathPrefix("/api/v1/kots/").Methods("HEAD", "GET", "POST", "PUT", "DELETE").HandlerFunc(handlers.NodeProxy(upstream))

	// proxy for license/titled api
	r.Path("/license/v1/license").Methods("GET").HandlerFunc(handlers.NodeProxy(upstream))

	// Apps
	r.Path("/api/v1/apps").Methods("OPTIONS", "GET").HandlerFunc(handlers.ListApps)
	r.Path("/api/v1/apps/app/{appSlug}").Methods("OPTIONS", "GET").HandlerFunc(handlers.GetApp)

	// Airgap
	r.Path("/api/v1/app/airgap").Methods("OPTIONS", "POST", "PUT").HandlerFunc(handlers.UploadAirgapBundle)
	r.Path("/api/v1/app/airgap/status").Methods("OPTIONS", "GET").HandlerFunc(handlers.GetAirgapInstallStatus)

	// Implemented handlers
	r.Path("/api/v1/license/platform").Methods("OPTIONS", "POST").HandlerFunc(handlers.ExchangePlatformLicense)
	r.Path("/api/v1/app/{appSlug}/sequence/{sequence}/preflight/ignore-rbac").Methods("OPTIONS", "POST").HandlerFunc(handlers.IgnorePreflightRBACErrors)
	r.Path("/api/v1/app/{appSlug}/sequence/{sequence}/preflight/run").Methods("OPTIONS", "POST").HandlerFunc(handlers.StartPreflightChecks)
	r.Path("/api/v1/app/{appSlug}/sequence/{sequence}/preflight/result").Methods("OPTIONS", "GET").HandlerFunc(handlers.GetPreflightResult)
	r.Path("/api/v1/app/{appSlug}/sequence/{sequence}/preflightcommand").Methods("OPTIONS", "GET").HandlerFunc(handlers.GetPreflightCommand)
	r.Path("/api/v1/preflight/result").Methods("OPTIONS", "GET").HandlerFunc(handlers.GetLatestPreflightResult)
	r.Path("/api/v1/upload").Methods("PUT").HandlerFunc(handlers.UploadExistingApp)
	r.Path("/api/v1/download").Methods("GET").HandlerFunc(handlers.DownloadApp)
	r.Path("/api/v1/app/{appSlug}/sequence/{sequence}/deploy").Methods("OPTIONS", "POST").HandlerFunc(handlers.DeployAppVersion)
	r.Path("/api/v1/app/{appSlug}/sequence/{sequence}/renderedcontents").Methods("OPTIONS", "GET").HandlerFunc(handlers.GetAppRenderedContents)
	r.Path("/api/v1/app/{appSlug}/sequence/{sequence}/contents").Methods("OPTIONS", "GET").HandlerFunc(handlers.GetAppContents)
	r.Path("/api/v1/app/{appSlug}/cluster/{clusterId}/dashboard").Methods("OPTIONS", "GET").HandlerFunc(handlers.GetAppDashboard)
	r.Path("/api/v1/app/{appSlug}/cluster/{clusterId}/sequence/{sequence}/downstreamoutput").Methods("OPTIONS", "GET").HandlerFunc(handlers.GetDownstreamOutput)

	r.HandleFunc("/api/v1/login", handlers.Login)
	r.HandleFunc("/api/v1/logout", handlers.Logout)

	// Installation
	r.Path("/api/v1/license").Methods("OPTIONS", "POST").HandlerFunc(handlers.UploadNewLicense)
	r.Path("/api/v1/license/resume").Methods("OPTIONS", "PUT").HandlerFunc(handlers.ResumeInstallOnline)

	r.Path("/api/v1/registry").Methods("OPTIONS", "GET").HandlerFunc(handlers.GetKotsadmRegistry)
	r.Path("/api/v1/imagerewritestatus").Methods("OPTIONS", "GET").HandlerFunc(handlers.GetImageRewriteStatus)

	r.Path("/api/v1/metadata").Methods("OPTIONS", "GET").HandlerFunc(handlers.Metadata)
	r.Path("/api/v1/app/online/status").Methods("OPTIONS", "GET").HandlerFunc(handlers.GetOnlineInstallStatus)
	r.Path("/api/v1/app/{appSlug}/registry").Methods("OPTIONS", "PUT").HandlerFunc(handlers.UpdateAppRegistry)
	r.Path("/api/v1/app/{appSlug}/registry").Methods("OPTIONS", "GET").HandlerFunc(handlers.GetAppRegistry)
	r.Path("/api/v1/app/{appSlug}/registry/validate").Methods("OPTIONS", "POST").HandlerFunc(handlers.ValidateAppRegistry)
	r.Path("/api/v1/app/{appSlug}/config").Methods("OPTIONS", "PUT").HandlerFunc(handlers.UpdateAppConfig)
	r.Path("/api/v1/app/{appSlug}/config/{sequence}").Methods("OPTIONS", "GET").HandlerFunc(handlers.CurrentAppConfig)
	r.Path("/api/v1/app/{appSlug}/liveconfig").Methods("OPTIONS", "POST").HandlerFunc(handlers.LiveAppConfig)
	r.Path("/api/v1/app/{appSlug}/license").Methods("OPTIONS", "PUT").HandlerFunc(handlers.SyncLicense)
	r.Path("/api/v1/app/{appSlug}/license").Methods("OPTIONS", "GET").HandlerFunc(handlers.GetLicense)
	r.Path("/api/v1/app/{appSlug}/updatecheck").Methods("OPTIONS", "POST").HandlerFunc(handlers.AppUpdateCheck)
	r.Path("/api/v1/app/{appSlug}/updatecheckerspec").Methods("OPTIONS", "PUT").HandlerFunc(handlers.UpdateCheckerSpec)

	// kotsadm snapshots
	r.Path("/api/v1/snapshots").Methods("OPTIONS", "GET").HandlerFunc(handlers.ListKotsadmBackups)
	r.Path("/api/v1/snapshot/{snapshotName}").Methods("OPTIONS", "GET").HandlerFunc(handlers.GetKotsadmBackup)
	r.Path("/api/v1/velero").Methods("OPTIONS", "GET").HandlerFunc(handlers.GetVeleroStatus)

	// App snapshot routes
	r.Path("/api/v1/app/{appSlug}/snapshot/backup").Methods("OPTIONS", "POST").HandlerFunc(handlers.CreateBackup)
	r.Path("/api/v1/app/{appSlug}/snapshot/restore/status").Methods("OPTIONS", "GET").HandlerFunc(handlers.GetRestoreStatus)
	r.Path("/api/v1/app/{appSlug}/snapshots").Methods("OPTIONS", "GET").HandlerFunc(handlers.ListBackups)
	r.Path("/api/v1/app/{appSlug}/snapshot/config").Methods("OPTIONS", "GET").HandlerFunc(handlers.GetSnapshotConfig)

	// Global snapshot routes
	r.Path("/api/v1/snapshots/settings").Methods("OPTIONS", "GET").HandlerFunc(handlers.GetGlobalSnapshotSettings)
	r.Path("/api/v1/snapshots/settings").Methods("OPTIONS", "PUT").HandlerFunc(handlers.UpdateGlobalSnapshotSettings)
	r.Path("/api/v1/snapshot/{snapshotName}/restore").Methods("OPTIONS", "POST").HandlerFunc(handlers.CreateRestore)

	// Find a home snapshot routes
	r.Path("/api/v1/snapshot/{backup}/logs").Methods("OPTIONS", "GET").HandlerFunc(handlers.DownloadSnapshotLogs)

	// KURL
	r.HandleFunc("/api/v1/kurl", handlers.NotImplemented)
	r.Path("/api/v1/kurl/generate-node-join-command-worker").Methods("OPTIONS", "POST").HandlerFunc(handlers.GenerateNodeJoinCommandWorker)
	r.Path("/api/v1/kurl/generate-node-join-command-master").Methods("OPTIONS", "POST").HandlerFunc(handlers.GenerateNodeJoinCommandMaster)
	r.Path("/api/v1/kurl/nodes/{nodeName}/drain").Methods("OPTIONS", "POST").HandlerFunc(handlers.DrainNode)
	r.Path("/api/v1/kurl/nodes/{nodeName}").Methods("OPTIONS", "DELETE").HandlerFunc(handlers.DeleteNode)

	// Prometheus
	r.Path("/api/v1/prometheus").Methods("OPTIONS", "POST").HandlerFunc(handlers.SetPrometheusAddress)

	// GitOps
	r.Path("/api/v1/gitops/app/{appId}/cluster/{clusterId}/update").Methods("OPTIONS", "PUT").HandlerFunc(handlers.UpdateAppGitOps)
	r.Path("/api/v1/gitops/app/{appId}/cluster/{clusterId}/disable").Methods("OPTIONS", "POST").HandlerFunc(handlers.DisableAppGitOps)
	r.Path("/api/v1/gitops/app/{appId}/cluster/{clusterId}/initconnection").Methods("OPTIONS", "POST").HandlerFunc(handlers.InitGitOpsConnection)
	r.Path("/api/v1/gitops/reset").Methods("OPTIONS", "POST").HandlerFunc(handlers.ResetGitOps)

	// to avoid confusion, we don't serve this in the dev env...
	if os.Getenv("DISABLE_SPA_SERVING") != "1" {
		spa := handlers.SPAHandler{StaticPath: filepath.Join("web", "dist"), IndexPath: "index.html"}
		r.PathPrefix("/").Handler(spa)
	}

	r.Handle("/socket.io/", socketservice.Start().Server)

	srv := &http.Server{
		Handler: r,
		Addr:    ":3000",
	}

	fmt.Printf("Starting kotsadm API on port %d...\n", 3000)

	log.Fatal(srv.ListenAndServe())
}
