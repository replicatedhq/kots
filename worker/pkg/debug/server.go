package debug

import (
	"expvar"
	"fmt"
	"net/http"

	"github.com/go-kit/kit/log"
	"github.com/gorilla/mux"
)

func NewServer(logger log.Logger) http.Handler {
	router := mux.NewRouter()
	router.HandleFunc("/healthz", getHealthz(logger)).Methods("GET")
	return router
}

func getHealthz(logger log.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		fmt.Fprintf(w, "{\n")
		first := true
		expvar.Do(func(kv expvar.KeyValue) {
			if kv.Key == "cmdline" || kv.Key == "memstats" {
				return
			}
			if !first {
				fmt.Fprintf(w, ",\n")
			}
			first = false
			fmt.Fprintf(w, "  %q: %s", kv.Key, kv.Value)
		})
		fmt.Fprintf(w, "\n}\n")
	}
}
