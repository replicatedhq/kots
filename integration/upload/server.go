package upload

import (
	"context"
	"net/http"
)

func StartMockServer(endpoint string, method string, expectedUpdateCursor string, expectedVersionLabel string, expectedLicense string, archive []byte) (chan bool, error) {
	stopCh := make(chan bool)

	srv := &http.Server{Addr: ":3000"}
	http.HandleFunc("/api/v1/kots", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != method {
			w.WriteHeader(404)
			return
		}
		w.Write([]byte(""))
	})

	go func() {
		srv.ListenAndServe()
	}()

	go func() {
		<-stopCh
		srv.Shutdown(context.TODO())
	}()

	return stopCh, nil
}
