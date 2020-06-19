package handlers

import (
	"context"
	"net/http"
	"os"

	"github.com/pkg/errors"
	"github.com/replicatedhq/kots/kotsadm/pkg/logger"
	"github.com/replicatedhq/kots/kotsadm/pkg/session"
	kuberneteserrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

func requireValidSession(w http.ResponseWriter, r *http.Request) error {
	if r.Header.Get("Authorization") == "" {
		w.WriteHeader(http.StatusUnauthorized)
		return errors.New("authorization header empty")
	}

	sess, err := session.Parse(r.Header.Get("Authorization"))
	if err != nil {
		logger.Error(err)
		w.WriteHeader(http.StatusInternalServerError)
		return errors.Wrap(err, "invalid session")
	}

	// we don't currently have roles, all valid tokens are valid sessions
	if sess == nil || sess.ID == "" {
		w.WriteHeader(http.StatusUnauthorized)
		return errors.New("empty session")
	}

	return nil
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
