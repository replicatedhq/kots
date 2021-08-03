package cluster

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/replicatedhq/kots/pkg/logger"
	authenticationv1 "k8s.io/api/authentication/v1"
	authorizationv1 "k8s.io/api/authorization/v1"
)

func StartAuthnzServer() {
	log.Printf("starting k8s authnz server")

	r := mux.NewRouter()

	r.HandleFunc("/healthz", HealthzHandler)
	r.Path("/api/v1/cluster-authn").Methods("POST").HandlerFunc(ClusterAuthnHandler)
	r.Path("/api/v1/cluster-authz").Methods("POST").HandlerFunc(ClusterAuthzHandler)

	srv := &http.Server{
		Handler: r,
		Addr:    ":8880",
	}

	log.Fatal(srv.ListenAndServe())
}

type HealthzResponse struct {
	Version string         `json:"version"`
	GitSHA  string         `json:"gitSha"`
	Status  StatusResponse `json:"status"`
}
type StatusResponse struct {
}

func HealthzHandler(w http.ResponseWriter, r *http.Request) {
	statusCode := 200
	healthzResponse := HealthzResponse{}
	JSON(w, statusCode, healthzResponse)
}

// ClusterAuthn
func ClusterAuthnHandler(w http.ResponseWriter, r *http.Request) {
	logger.Debug("cluster-authn handler")

	tokenReview := authenticationv1.TokenReview{}
	if err := json.NewDecoder(r.Body).Decode(&tokenReview); err != nil {
		logger.Error(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	response := authenticationv1.TokenReview{
		Status: authenticationv1.TokenReviewStatus{
			Authenticated: true,
			User: authenticationv1.UserInfo{
				Username: "kots", // TODO
			},
		},
	}

	JSON(w, http.StatusOK, response)
	return
}

// ClusterAuthz
func ClusterAuthzHandler(w http.ResponseWriter, r *http.Request) {
	logger.Debug("cluster-authz handler")

	subjectAccessReview := authorizationv1.SubjectAccessReview{}
	if err := json.NewDecoder(r.Body).Decode(&subjectAccessReview); err != nil {
		logger.Error(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	// if this is a kubelet bootstrap token, only allow CSR request to bootstrap

	response := authorizationv1.SubjectAccessReview{
		Status: authorizationv1.SubjectAccessReviewStatus{
			Allowed: true, // TODO
		},
	}

	JSON(w, http.StatusOK, response)
	return
}

func JSON(w http.ResponseWriter, code int, payload interface{}) {
	response, err := json.Marshal(payload)
	if err != nil {
		logger.Error(err)
		w.WriteHeader(500)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	w.Write(response)
}
