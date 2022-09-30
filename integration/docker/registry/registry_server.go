package replicated

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/gorilla/mux"
)

type MockServerOptions struct {
	Manifests map[string]string
}

func StartMockServer(opts MockServerOptions) (*http.Server, error) {
	r := mux.NewRouter()
	srv := &http.Server{
		Addr:    ":3002",
		Handler: r,
	}

	r.Path("/").Methods("GET").HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	r.Path("/v2/{imageName}/manifests/{reference}").Methods("GET").HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		imageName := mux.Vars(r)["imageName"]
		reference := mux.Vars(r)["reference"]
		key := ""
		if strings.HasPrefix(reference, "sha256:") {
			key = fmt.Sprintf("%s@%s", imageName, reference)
		} else {
			key = fmt.Sprintf("%s:%s", imageName, reference)
		}
		w.Write([]byte(opts.Manifests[key]))
	})

	go func() {
		srv.ListenAndServe()
	}()

	return srv, nil
}
