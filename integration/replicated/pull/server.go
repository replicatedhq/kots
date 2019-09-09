package pull

import (
	"context"
	"net/http"
)

func StartMockServer(endpoint string, appSlug string, licenseID string, archive []byte) (chan bool, error) {
	stopCh := make(chan bool)

	srv := &http.Server{Addr: ":3000"}
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Write(archive)
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
