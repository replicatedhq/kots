package upload

import (
	"context"
	"errors"
	"net/http"
	"time"
)

func StartMockServer(method string) (chan bool, error) {
	stopCh := make(chan bool)

	srv := &http.Server{Addr: ":3001"}
	http.HandleFunc("/api/v1/upload", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != method {
			w.WriteHeader(404)
			return
		}
		w.Write([]byte(`{"slug": "sluggy"}`))
	})
	http.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{}`))
	})

	go func() {
		srv.ListenAndServe()
	}()

	go func() {
		<-stopCh
		srv.Shutdown(context.TODO())
	}()

	// for the the http server to be ready
	quickClient := &http.Client{
		Timeout: time.Millisecond * 100,
	}
	start := time.Now()
	for {
		response, err := quickClient.Get("http://localhost:3001/healthz")
		if err == nil && response.StatusCode == http.StatusOK {
			break
		}
		if time.Now().Sub(start) > time.Second*5 {
			return nil, errors.New("mock server failed to start in the allocated time")
		}

		time.Sleep(time.Millisecond * 10)
	}

	return stopCh, nil
}
