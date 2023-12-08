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
	"github.com/replicatedhq/kots/pkg/embeddedcluster"
	"github.com/replicatedhq/kots/pkg/handlers"
	"github.com/replicatedhq/kots/pkg/helm"
	identitymigrate "github.com/replicatedhq/kots/pkg/identity/migrate"
	"github.com/replicatedhq/kots/pkg/informers"
	"github.com/replicatedhq/kots/pkg/k8sutil"
	"github.com/replicatedhq/kots/pkg/operator"
	"github.com/replicatedhq/kots/pkg/operator/client"
	"github.com/replicatedhq/kots/pkg/persistence"
	"github.com/replicatedhq/kots/pkg/policy"
	"github.com/replicatedhq/kots/pkg/rbac"
	"github.com/replicatedhq/kots/pkg/reporting"
	"github.com/replicatedhq/kots/pkg/session"
	"github.com/replicatedhq/kots/pkg/snapshotscheduler"
	"github.com/replicatedhq/kots/pkg/store"
	"github.com/replicatedhq/kots/pkg/supportbundle"
	"github.com/replicatedhq/kots/pkg/updatechecker"
	"github.com/replicatedhq/kots/pkg/util"
	"golang.org/x/crypto/bcrypt"
)

type APIServerParams struct {
	Version                string
	RqliteURI              string
	AutocreateClusterToken string
	SharedPassword         string
}

func Start(params *APIServerParams) {
	log.Printf("kotsadm version %s\n", params.Version)

	if !util.IsHelmManaged() {
		// set some persistence variables
		persistence.InitDB(params.RqliteURI)

		ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
		if err := store.GetStore().WaitForReady(ctx); err != nil {
			log.Println("error waiting for ready")
			panic(err)
		}
		cancel()
	}

	// check if we need to migrate from postgres before doing anything else
	if !util.IsHelmManaged() {
		if err := persistence.MigrateFromPostgresToRqlite(); err != nil {
			log.Println("error migrating from postgres to rqlite")
			panic(err)
		}
	}

	if err := bootstrap(BootstrapParams{
		AutoCreateClusterToken: params.AutocreateClusterToken,
	}); err != nil {
		log.Println("error bootstrapping")
		panic(err)
	}

	if !util.IsHelmManaged() {
		store.GetStore().RunMigrations()
		if err := identitymigrate.RunMigrations(context.TODO(), util.PodNamespace); err != nil {
			log.Println("Failed to run identity migrations: ", err)
		}
	}

	if err := binaries.InitKubectl(); err != nil {
		log.Println("error initializing kubectl binaries package")
		panic(err)
	}

	if err := binaries.InitKustomize(); err != nil {
		log.Println("error initializing kustomize binaries package")
		panic(err)
	}

	if !util.IsHelmManaged() {
		client := &client.Client{
			TargetNamespace:       util.AppNamespace(),
			ExistingHookInformers: map[string]bool{},
			HookStopChans:         []chan struct{}{},
		}
		store := store.GetStore()
		k8sClientset, err := k8sutil.GetClientset()
		if err != nil {
			log.Println("error getting k8s clientset")
			panic(err)
		}
		op := operator.Init(client, store, params.AutocreateClusterToken, k8sClientset)
		if err := op.Start(); err != nil {
			log.Println("error starting the operator")
			panic(err)
		}
		defer op.Shutdown()

		if err := embeddedcluster.InitClusterState(context.TODO(), k8sClientset, store); err != nil {
			log.Println("Failed to initialize cluster state:", err)
		}
	}

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

	if err := reporting.Init(); err != nil {
		log.Println("failed to initialize reporting:", err)
	}

	supportbundle.StartServer()

	if err := informers.Start(); err != nil {
		log.Println("Failed to start informers:", err)
	}

	if !util.IsHelmManaged() {
		if err := updatechecker.Start(); err != nil {
			log.Println("Failed to start update checker:", err)
		}
		if err := snapshotscheduler.Start(); err != nil {
			log.Println("Failed to start snapshot scheduler:", err)
		}
	}

	if err := session.StartSessionPurgeCronJob(); err != nil {
		log.Println("Failed to start session purge cron job:", err)
	}

	waitForAirgap, err := automation.NeedToWaitForAirgapApp()
	if err != nil {
		log.Println("Failed to check if airgap install is in progress:", err)
	} else if !waitForAirgap {
		opts := automation.AutomateInstallOptions{}
		if err := automation.AutomateInstall(opts); err != nil {
			log.Println("Failed to run automated installs:", err)
		}
	}

	if util.IsHelmManaged() {
		if err := helm.Init(context.TODO()); err != nil {
			log.Println("Failed to initialize helm data: ", err)
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

	kotsStore := store.GetStore()

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

	// to avoid confusion, we don't serve this in the dev env...
	if os.Getenv("DISABLE_SPA_SERVING") != "1" {
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
