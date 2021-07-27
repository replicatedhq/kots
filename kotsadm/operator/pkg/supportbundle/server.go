package supportbundle

import (
	"fmt"
	"net/http"
	"runtime/pprof"

	"github.com/google/martian/log"
	"github.com/gorilla/mux"
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

		log.Infof("Starting support bundle server on port %d...", port)

		err := srv.ListenAndServe()
		log.Errorf("failed to run support bundle server: %v", err)
	}()
}

func getGoRoutines(w http.ResponseWriter, r *http.Request) {
	profile := pprof.Lookup("goroutine")
	if profile == nil {
		log.Errorf("failed to get goroutine info")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Disposition", "attachment; filename=goroutines.txt")
	w.Header().Set("Content-Type", "text/plain")

	w.WriteHeader(http.StatusOK)

	if err := profile.WriteTo(w, 2); err != nil {
		log.Errorf("failed to get goroutine info: %v", err)
		return
	}
}
