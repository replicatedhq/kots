package upgrader

import (
	"fmt"
	"log"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/pkg/errors"
	"github.com/replicatedhq/kots/pkg/buildversion"
	"github.com/replicatedhq/kots/pkg/handlers"
)

type ServerParams struct {
	Port string
}

func Serve(params ServerParams) error {
	log.Printf("KOTS version %s\n", buildversion.Version())

	r := mux.NewRouter()

	r.Use(handlers.CorsMiddleware)
	r.Methods("OPTIONS").HandlerFunc(handlers.CORS)

	debugRouter := r.NewRoute().Subrouter()
	debugRouter.Use(handlers.DebugLoggingMiddleware)

	loggingRouter := r.NewRoute().Subrouter()
	loggingRouter.Use(handlers.LoggingMiddleware)

	handler := &handlers.Handler{}

	// TODO NOW: auth by authSlug token the cli typically uses?

	/**********************************************************************
	* KOTS token auth routes
	**********************************************************************/

	handlers.RegisterTokenAuthRoutes(handler, debugRouter, loggingRouter)

	// Prevent API requests that don't match anything in this router from returning UI content
	r.PathPrefix("/api").Handler(handlers.StatusNotFoundHandler{})

	/**********************************************************************
	* Static routes
	**********************************************************************/

	spa := handlers.SPAHandler{}
	r.PathPrefix("/").Handler(spa)

	srv := &http.Server{
		Handler: r,
		Addr:    fmt.Sprintf(":%s", params.Port),
	}

	fmt.Printf("Starting upgrader on port %s...\n", params.Port)

	if err := srv.ListenAndServe(); err != nil {
		return errors.Wrap(err, "failed to listen and serve")
	}

	return nil
}
