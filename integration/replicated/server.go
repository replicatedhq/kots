package replicated

import (
	"fmt"
	"net/http"
)

func StartMockServer(archive, license []byte) (*http.Server, error) {
	mux := http.NewServeMux()
	srv := &http.Server{
		Addr:    ":3000",
		Handler: mux,
	}
	mux.HandleFunc("/release/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Printf("handling URL %s with release tarball\n", r.URL)
		w.Write(archive)
	})

	mux.HandleFunc("/license/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Printf("handling URL %s with license file\n", r.URL)
		w.Write(license)
	})
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Printf("WARNING! unable to handle url %s", r.URL)
		w.WriteHeader(501)
	})

	go func() {
		srv.ListenAndServe()
	}()

	return srv, nil
}
