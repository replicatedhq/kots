package handlers

import (
	"context"
	"encoding/base64"
	"net/http"
	"os"
	"strings"

	"github.com/pkg/errors"
	"github.com/replicatedhq/kots/kotsadm/pkg/session"
	"github.com/replicatedhq/kots/kotsadm/pkg/session/types"
	"github.com/replicatedhq/kots/kotsadm/pkg/store"
	kuberneteserrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

type authorization struct {
	Username string
	Password string
}

func parseClusterAuthorization(authHeader string) (authorization, error) {
	if !strings.HasPrefix(authHeader, "Basic ") { // does this need "Kots " too?
		return authorization{}, errors.New("only basic auth is supported")
	}

	authHeader = strings.TrimSpace(strings.TrimPrefix(authHeader, "Basic "))

	data, err := base64.StdEncoding.DecodeString(authHeader)
	if err != nil {
		return authorization{}, errors.Wrap(err, "failed ot base64 decode auth header")
	}

	parts := strings.Split(string(data), ":")
	if len(parts) != 2 {
		return authorization{}, errors.Errorf("expected 2 parts in auth header, found %d", len(parts))
	}

	return authorization{
		Username: parts[0],
		Password: parts[1],
	}, nil
}

func requireValidSession(kotsStore store.KOTSStore, w http.ResponseWriter, r *http.Request) (*types.Session, error) {
	if r.Method == "OPTIONS" {
		return nil, nil
	}

	auth := r.Header.Get("authorization")

	if auth == "" {
		err := errors.New("authorization header empty")
		response := ErrorResponse{Error: err.Error()}
		JSON(w, http.StatusUnauthorized, response)
		return nil, err
	}

	sess, err := session.Parse(kotsStore, auth)
	if err != nil {
		response := ErrorResponse{Error: "failed to parse authorization header"}
		JSON(w, http.StatusUnauthorized, response)
		return nil, errors.Wrap(err, "invalid session")
	}

	// we don't currently have roles, all valid tokens are valid sessions
	if sess == nil || sess.ID == "" {
		err := errors.New("no session in auth header")
		response := ErrorResponse{Error: err.Error()}
		JSON(w, http.StatusUnauthorized, response)
		return nil, err
	}

	return sess, nil
}

func requireValidKOTSToken(w http.ResponseWriter, r *http.Request) error {
	if r.Header.Get("Authorization") == "" {
		w.WriteHeader(http.StatusUnauthorized)
		return errors.New("authorization header empty")
	}

	config, err := rest.InClusterConfig()
	if err != nil {
		return errors.Wrap(err, "failed to get in cluster config")
	}

	client, err := kubernetes.NewForConfig(config)
	if err != nil {
		return errors.Wrap(err, "Failed to create kubernetes clientset")
	}

	secret, err := client.CoreV1().Secrets(os.Getenv("POD_NAMESPACE")).Get(context.TODO(), "kotsadm-authstring", metav1.GetOptions{})
	if kuberneteserrors.IsNotFound(err) {
		return errors.New("no authstring found in cluster")
	}

	if err != nil {
		return errors.Wrap(err, "failed to read auth string")
	}

	if r.Header.Get("Authorization") == string(secret.Data["kotsadm-authstring"]) {
		return nil
	}

	return errors.New("invalid auth")
}
