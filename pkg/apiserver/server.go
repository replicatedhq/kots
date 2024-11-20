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
	"github.com/replicatedhq/kots/pkg/automation"
	"github.com/replicatedhq/kots/pkg/binaries"
	"github.com/replicatedhq/kots/pkg/handlers"
	identitymigrate "github.com/replicatedhq/kots/pkg/identity/migrate"
	"github.com/replicatedhq/kots/pkg/informers"
	"github.com/replicatedhq/kots/pkg/k8sutil"
	"github.com/replicatedhq/kots/pkg/logger"
	"github.com/replicatedhq/kots/pkg/operator"
	operatorclient "github.com/replicatedhq/kots/pkg/operator/client"
	"github.com/replicatedhq/kots/pkg/persistence"
	"github.com/replicatedhq/kots/pkg/policy"
	"github.com/replicatedhq/kots/pkg/rbac"
	"github.com/replicatedhq/kots/pkg/reporting"
	"github.com/replicatedhq/kots/pkg/session"
	"github.com/replicatedhq/kots/pkg/snapshotscheduler"
	"github.com/replicatedhq/kots/pkg/store"
	"github.com/replicatedhq/kots/pkg/supportbundle"
	"github.com/replicatedhq/kots/pkg/update"
	"github.com/replicatedhq/kots/pkg/updatechecker"
	"github.com/replicatedhq/kots/pkg/upgradeservice"
	"github.com/replicatedhq/kots/pkg/util"
	"golang.org/x/crypto/bcrypt"
)

type APIServerParams struct {
	Version                string
	AutocreateClusterToken string
	SharedPassword         string
}

func Start(params *APIServerParams) {
	log.Printf("kotsadm version %s\n", params.Version)

	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	if err := store.GetStore().WaitForReady(ctx); err != nil {
		log.Println("error waiting for ready")
		panic(err)
	}
	cancel()
	logger.Debug("store is ready")

	// check if we need to migrate from postgres before doing anything else
	if err := persistence.MigrateFromPostgresToRqlite(); err != nil {
		log.Println("error migrating from postgres to rqlite")
		panic(err)
	}
	logger.Debug("postgres is migrated to rqlite or the migration was not required")

	if err := bootstrap(BootstrapParams{
		AutoCreateClusterToken: params.AutocreateClusterToken,
	}); err != nil {
		log.Println("error bootstrapping")
		panic(err)
	}
	logger.Debug("bootstrap completed")

	store.GetStore().RunMigrations()
	logger.Debug("store migrations completed")

	if err := identitymigrate.RunMigrations(context.TODO(), util.PodNamespace); err != nil {
		log.Println("Failed to run identity migrations: ", err)
	}
	logger.Debug("identity migrations completed")

	if err := binaries.InitKubectl(); err != nil {
		log.Println("error initializing kubectl binaries package")
		panic(err)
	}
	logger.Debug("kubectl binaries are initialized")

	if err := binaries.InitKustomize(); err != nil {
		log.Println("error initializing kustomize binaries package")
		panic(err)
	}
	logger.Debug("kustomize binaries are initialized")

	kotsStore := store.GetStore()
	logger.Debug("kotsstore is initialized")

	operatorClient := &operatorclient.Client{
		TargetNamespace:       util.AppNamespace(),
		ExistingHookInformers: map[string]bool{},
		HookStopChans:         []chan struct{}{},
	}
	k8sClientset, err := k8sutil.GetClientset()
	if err != nil {
		log.Println("error getting k8s clientset")
		panic(err)
	}
	logger.Debug("k8s clientset is initialized")

	op := operator.Init(operatorClient, kotsStore, params.AutocreateClusterToken, k8sClientset)
	if err := op.Start(); err != nil {
		log.Println("error starting the operator")
		panic(err)
	}
	defer op.Shutdown()
	logger.Debug("operator is initialized")

	if params.SharedPassword != "" {
		// TODO: this won't override the password in the database
		// it's only possible to set this in the kots run workflow
		bcryptPassword, err := bcrypt.GenerateFromPassword([]byte(params.SharedPassword), 10)
		if err != nil {
			panic(err)
		}
		os.Setenv("SHARED_PASSWORD_BCRYPT", string(bcryptPassword))
	}

	if err := k8sutil.InitHelmCapabilities(); err != nil {
		panic(err)
	}
	logger.Debug("helm capabilities are initialized")

	if err := update.InitAvailableUpdatesDir(); err != nil {
		panic(err)
	}
	logger.Debug("available updates dir is initialized")

	if err := reporting.Init(); err != nil {
		log.Println("failed to initialize reporting:", err)
	}
	logger.Debug("reporting is initialized")

	supportbundle.StartServer()
	logger.Debug("support bundle server is started")

	if err := informers.Start(); err != nil {
		log.Println("Failed to start informers:", err)
	}
	logger.Debug("informers are started")

	if err := updatechecker.Start(); err != nil {
		log.Println("Failed to start update checker:", err)
	}
	logger.Debug("update checker is started")

	if err := snapshotscheduler.Start(); err != nil {
		log.Println("Failed to start snapshot scheduler:", err)
	}
	logger.Debug("snapshot scheduler is started")

	if err := session.StartSessionPurgeCronJob(); err != nil {
		log.Println("Failed to start session purge cron job:", err)
	}
	logger.Debug("session purge cron job is started")

	waitForAirgap, err := automation.NeedToWaitForAirgapApp()
	if err != nil {
		log.Println("Failed to check if airgap install is in progress:", err)
	} else if !waitForAirgap {
		opts := automation.AutomateInstallOptions{}
		if err := automation.AutomateInstall(opts); err != nil {
			log.Println("Failed to run automated installs:", err)
		}
	}
	logger.Debug("automated airgap installs are completed")

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

	handlers.RegisterUnauthenticatedRoutes(handler, kotsStore, debugRouter, loggingRouter)

	/**********************************************************************
	* KOTS token auth routes
	**********************************************************************/

	handlers.RegisterTokenAuthRoutes(handler, debugRouter, loggingRouter)

	/**********************************************************************
	* Session auth routes
	**********************************************************************/

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

	// Serve the upgrade UI from the upgrade service
	// CAUTION: modifying this route WILL break backwards compatibility
	r.PathPrefix("/upgrade-service/app/{appSlug}").Methods("GET").HandlerFunc(upgradeservice.Proxy)

	if os.Getenv("DISABLE_SPA_SERVING") != "1" { // we don't serve this in the dev env
		spa := handlers.SPAHandler{}
		r.PathPrefix("/").Handler(spa)
	} else if os.Getenv("ENABLE_WEB_PROXY") == "1" { // for dev env
		u, err := url.Parse("http://kotsadm-web:8080")
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
