package upgradeservice

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"os/signal"
	"path/filepath"
	"time"

	"github.com/gorilla/mux"
	"github.com/pkg/errors"
	"github.com/replicatedhq/kots/pkg/buildversion"
	"github.com/replicatedhq/kots/pkg/upgradeservice/handlers"
	upgradepreflight "github.com/replicatedhq/kots/pkg/upgradeservice/preflight"
	"github.com/replicatedhq/kots/pkg/upgradeservice/types"
)

func Serve(params types.UpgradeServiceParams) error {
	fmt.Printf("Starting KOTS Upgrade Service version %s on port %s\n", buildversion.Version(), params.Port)

	// cleanup on shutdown
	defer cleanup(params)

	if err := bootstrap(params); err != nil {
		return errors.Wrap(err, "failed to bootstrap")
	}

	if err := upgradepreflight.Init(); err != nil {
		return errors.Wrap(err, "failed to init preflight")
	}

	r := mux.NewRouter()
	r.Use(handlers.ParamsMiddleware(params))

	handler := &handlers.Handler{}
	handlers.RegisterAPIRoutes(r, handler)

	/**********************************************************************
	* Static routes
	**********************************************************************/

	if os.Getenv("DISABLE_SPA_SERVING") != "1" { // we don't serve this in the dev env
		spa := handlers.SPAHandler{}
		r.PathPrefix("/upgrade-service/app/{appSlug}").Handler(spa)
	} else if os.Getenv("ENABLE_WEB_PROXY") == "1" { // for dev env
		u, err := url.Parse("http://kotsadm-web:8080")
		if err != nil {
			return errors.Wrap(err, "failed to parse kotsadm-web url")
		}
		upstream := httputil.NewSingleHostReverseProxy(u)
		webProxy := func(upstream *httputil.ReverseProxy) func(http.ResponseWriter, *http.Request) {
			return func(w http.ResponseWriter, r *http.Request) {
				r.Header.Set("X-Forwarded-Host", r.Header.Get("Host"))
				upstream.ServeHTTP(w, r)
			}
		}(upstream)
		r.PathPrefix("/upgrade-service/app/{appSlug}").HandlerFunc(webProxy)
	}

	srv := &http.Server{
		Handler: r,
		Addr:    fmt.Sprintf(":%s", params.Port),
	}

	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("listen: %s\n", err)
		}
	}()

	// wait for interrupt signal to gracefully shut down the server and cleanup
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	<-c

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		return errors.Wrap(err, "failed to shutdown server")
	}

	return nil
}

func cleanup(params types.UpgradeServiceParams) {
	os.RemoveAll(params.AppArchive)
	os.RemoveAll(params.UpdateKOTSBin)
	os.RemoveAll(filepath.Dir(upgradepreflight.PreflightDataFile))
}
