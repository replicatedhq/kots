package supportbundle

import (
	"fmt"
	"net/http"
	"runtime/pprof"

	"github.com/gorilla/mux"
	"github.com/pkg/errors"
	"github.com/replicatedhq/kots/kotsadm/pkg/logger"
)

func StartServer() {
	go func() {
		port := 3030
		r := mux.NewRouter()

		r.HandleFunc("/goroutines", getGoRoutines)

		srv := &http.Server{
			Handler: r,
			Addr:    fmt.Sprintf(":%d", port),
		}

		fmt.Printf("Starting suppotbundle server on port %d...\n", port)

		err := srv.ListenAndServe()
		logger.Error(errors.Wrap(err, "failed to run support bundle server"))
	}()
}

func getGoRoutines(w http.ResponseWriter, r *http.Request) {
	profile := pprof.Lookup("goroutine")
	if profile == nil {
		logger.Errorf("failed to get goroutine info")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Disposition", "attachment; filename=goroutines.txt")
	w.Header().Set("Content-Type", "text/plain")

	w.WriteHeader(http.StatusOK)

	if err := profile.WriteTo(w, 2); err != nil {
		logger.Error(errors.Wrap(err, "failed to get goroutine info"))
		return
	}
}
