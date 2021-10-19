package apiserver

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"time"

	"github.com/gorilla/mux"
	"github.com/pkg/errors"
	"github.com/replicatedhq/kots/pkg/automation"
	"github.com/replicatedhq/kots/pkg/binaries"
	"github.com/replicatedhq/kots/pkg/handlers"
	"github.com/replicatedhq/kots/pkg/informers"
	"github.com/replicatedhq/kots/pkg/k8sutil"
	"github.com/replicatedhq/kots/pkg/logger"
	"github.com/replicatedhq/kots/pkg/operator"
	"github.com/replicatedhq/kots/pkg/persistence"
	"github.com/replicatedhq/kots/pkg/policy"
	"github.com/replicatedhq/kots/pkg/rbac"
	"github.com/replicatedhq/kots/pkg/snapshotscheduler"
	"github.com/replicatedhq/kots/pkg/store"
	"github.com/replicatedhq/kots/pkg/supportbundle"
	"github.com/replicatedhq/kots/pkg/updatechecker"
	"github.com/segmentio/ksuid"
	"golang.org/x/crypto/bcrypt"
)

type APIServerParams struct {
	Version                string
	PostgresURI            string
	SQLiteURI              string
	AutocreateClusterToken string
	EnableIdentity         bool
	SharedPassword         string
	KubeconfigPath         string
	KotsDataDir            string
}

func Start(params *APIServerParams) {
	log.Printf("kotsadm version %s\n", params.Version)

	if params.KubeconfigPath != "" {
		// it's only possible to set this in the kots run workflow
		os.Setenv("KUBECONFIG", params.KubeconfigPath)
	}
	if params.KotsDataDir != "" {
		// it's only possible to set this in the kots run workflow
		os.Setenv("KOTS_DATA_DIR", params.KotsDataDir)
	}

	// set some persistence variables
	persistence.PostgresURI = params.PostgresURI
	persistence.SQLiteURI = params.SQLiteURI

	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	if err := store.GetStore().WaitForReady(ctx); err != nil {
		log.Println("error waiting for ready")
		panic(err)
	}
	cancel()

	if err := bootstrap(BootstrapParams{
		AutoCreateClusterToken: params.AutocreateClusterToken,
	}); err != nil {
		log.Println("error bootstrapping")
		panic(err)
	}

	store.GetStore().RunMigrations()

	if err := binaries.InitKubectl(); err != nil {
		log.Println("error initializing kubectl binaries package")
		panic(err)
	}

	if err := operator.Start(params.AutocreateClusterToken); err != nil {
		log.Println("error starting the operator")
		panic(err)
	}
	defer operator.Shutdown()

	if params.SharedPassword != "" {
		// TODO: this won't override the password in the database
		// it's only possible to set this in the kots run workflow
		bcryptPassword, err := bcrypt.GenerateFromPassword([]byte(params.SharedPassword), 10)
		if err != nil {
			panic(err)
		}
		os.Setenv("SHARED_PASSWORD_BCRYPT", string(bcryptPassword))
	}

	if params.EnableIdentity {
		err := bootstrapIdentity()
		if err != nil {
			log.Println("error bootstrapping identity")
			panic(err)
		}
	}

	if err := k8sutil.InitHelmCapabilities(); err != nil {
		panic(err)
	}

	if err := generateKotsadmID(); err != nil {
		logger.Infof("failed to generate kotsadm id:", err)
	}

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

	debugRouter := r.NewRoute().Subrouter()
	debugRouter.Use(handlers.DebugLoggingMiddleware)

	loggingRouter := r.NewRoute().Subrouter()
	loggingRouter.Use(handlers.LoggingMiddleware)

	handler := &handlers.Handler{}

	/**********************************************************************
	* Unauthenticated routes
	**********************************************************************/

	debugRouter.HandleFunc("/healthz", handler.Healthz)
	loggingRouter.HandleFunc("/api/v1/login", handler.Login)
	loggingRouter.HandleFunc("/api/v1/login/info", handler.GetLoginInfo)
	loggingRouter.HandleFunc("/api/v1/logout", handler.Logout) // this route uses its own auth
	loggingRouter.Path("/api/v1/metadata").Methods("GET").HandlerFunc(handlers.GetMetadataHandler(handlers.GetMetaDataConfig))

	loggingRouter.HandleFunc("/api/v1/oidc/login", handler.OIDCLogin)
	loggingRouter.HandleFunc("/api/v1/oidc/login/callback", handler.OIDCLoginCallback)

	loggingRouter.Path("/api/v1/troubleshoot/{appId}/{bundleId}").Methods("PUT").HandlerFunc(handler.UploadSupportBundle)
	loggingRouter.Path("/api/v1/troubleshoot/supportbundle/{bundleId}/redactions").Methods("PUT").HandlerFunc(handler.SetSupportBundleRedactions)

	// This the handler for license API and should be called by the application only.
	loggingRouter.Path("/license/v1/license").Methods("GET").HandlerFunc(handler.GetPlatformLicenseCompatibility)

	/**********************************************************************
	* KOTS token auth routes
	**********************************************************************/

	debugRouter.Path("/api/v1/kots/ports").Methods("GET").HandlerFunc(handler.GetApplicationPorts)
	loggingRouter.Path("/api/v1/upload").Methods("PUT").HandlerFunc(handler.UploadExistingApp)
	loggingRouter.Path("/api/v1/download").Methods("GET").HandlerFunc(handler.DownloadApp)
	loggingRouter.Path("/api/v1/airgap/install").Methods("POST").HandlerFunc(handler.UploadInitialAirgapApp)

	/**********************************************************************
	* Session auth routes
	**********************************************************************/

	kotsStore := store.GetStore()
	policyMiddleware := policy.NewMiddleware(kotsStore, rbac.DefaultRoles())

	sessionAuthQuietRouter := r.PathPrefix("").Subrouter()
	sessionAuthQuietRouter.Use(handlers.RequireValidSessionQuietMiddleware(kotsStore))

	sessionAuthQuietRouter.Path("/api/v1/ping").Methods("GET").HandlerFunc(handler.Ping)

	handlers.RegisterSessionAuthRoutes(r.PathPrefix("").Subrouter(), kotsStore, handler, policyMiddleware)

	// Prevent API requests that don't match anything in this router from returning UI content
	r.PathPrefix("/api").Handler(handlers.StatusNotFoundHandler{})

	/**********************************************************************
	* Static routes
	**********************************************************************/

	// to avoid confusion, we don't serve this in the dev env...
	if os.Getenv("DISABLE_SPA_SERVING") != "1" {
		spa := handlers.SPAHandler{}
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

	fmt.Printf("Starting Admin Console API on port %d...\n", 3000)

	log.Fatal(srv.ListenAndServe())
}

func generateKotsadmID() error {
	// Retrieve the ClusterID from store
	clusters, err := store.GetStore().ListClusters()
	if err != nil {
		return errors.Wrap(err, "failed to list clusters")
	}
	if len(clusters) == 0 {
		return nil
	}
	clusterID := clusters[0].ClusterID

	isKotsadmIDGenerated, err := store.GetStore().IsKotsadmIDGenerated()
	if err != nil {
		return errors.Wrap(err, "failed to generate id")
	}
	cmpExists, err := k8sutil.IsKotsadmIDConfigMapPresent()
	if err != nil {
		return errors.Wrap(err, "failed to check configmap")
	}

	if isKotsadmIDGenerated && !cmpExists {
		kotsadmID := ksuid.New().String()
		err = k8sutil.CreateKotsadmIDConfigMap(kotsadmID)
	} else if !isKotsadmIDGenerated && !cmpExists {
		err = k8sutil.CreateKotsadmIDConfigMap(clusterID)
	} else if !isKotsadmIDGenerated && cmpExists {
		err = k8sutil.UpdateKotsadmIDConfigMap(clusterID)
	} else {
		// id exists and so as configmap, noop
	}
	if err == nil {
		err = store.GetStore().SetIsKotsadmIDGenerated()
	}

	return err
}
