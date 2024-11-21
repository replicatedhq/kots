package handlers

import (
	"context"
	"encoding/base64"
	"net/http"
	"strings"
	"time"

	"github.com/pkg/errors"
	"github.com/replicatedhq/kots/pkg/handlers/types"
	"github.com/replicatedhq/kots/pkg/k8sutil"
	"github.com/replicatedhq/kots/pkg/logger"
	"github.com/replicatedhq/kots/pkg/session"
	sessiontypes "github.com/replicatedhq/kots/pkg/session/types"
	"github.com/replicatedhq/kots/pkg/store"
	"github.com/replicatedhq/kots/pkg/util"
	kuberneteserrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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

	parts := strings.SplitN(string(data), ":", 2)
	if len(parts) != 2 {
		return authorization{}, errors.Errorf("expected 2 parts in auth header, found %d", len(parts))
	}

	return authorization{
		Username: parts[0],
		Password: parts[1],
	}, nil
}

func requireValidSession(kotsStore store.Store, w http.ResponseWriter, r *http.Request) (*sessiontypes.Session, error) {
	if r.Method == "OPTIONS" {
		return nil, nil
	}

	auth := r.Header.Get("authorization")
	var signedTokenCookie *http.Cookie

	if auth == "" {
		signedTokenCookie, err := r.Cookie("signed-token")

		if err == http.ErrNoCookie && auth == "" {
			err := errors.New("missing authorization token")
			response := types.ErrorResponse{Error: util.StrPointer(err.Error())}
			JSON(w, http.StatusUnauthorized, response)
			return nil, err
		}
		auth = signedTokenCookie.Value
	}

	sess, err := session.Parse(kotsStore, auth)
	if err != nil {
		response := types.ErrorResponse{Error: util.StrPointer("failed to parse authorization header")}
		JSON(w, http.StatusUnauthorized, response)
		return nil, errors.Wrap(err, "invalid session")
	}

	// we don't currently have roles, all valid tokens are valid sessions
	if sess == nil || sess.ID == "" {
		err := errors.New("no session in auth header")
		response := types.ErrorResponse{Error: util.StrPointer(err.Error())}
		JSON(w, http.StatusUnauthorized, response)
		return nil, err
	}

	if time.Now().After(sess.ExpiresAt) {
		if err := kotsStore.DeleteSession(sess.ID); err != nil {
			logger.Error(errors.Wrapf(err, "session expired. failed to delete expired session %s", sess.ID))
		}
		err := errors.New("session expired")
		response := types.ErrorResponse{Error: util.StrPointer(err.Error())}
		JSON(w, http.StatusUnauthorized, response)
		return nil, err
	}

	passwordUpdatedAt, err := kotsStore.GetPasswordUpdatedAt()
	if err != nil {
		response := types.ErrorResponse{Error: util.StrPointer("failed to validate session with current password")}
		JSON(w, http.StatusInternalServerError, response)
		return nil, err
	}
	if passwordUpdatedAt != nil && passwordUpdatedAt.After(sess.IssuedAt) {
		if err := kotsStore.DeleteSession(sess.ID); err != nil {
			logger.Error(errors.Wrapf(err, "password was updated after session created. failed to delete invalid session %s", sess.ID))
		}
		err := errors.New("password changed, please login again")
		response := types.ErrorResponse{Error: util.StrPointer(err.Error())}
		JSON(w, http.StatusUnauthorized, response)
		return nil, err
	}

	// give the user the full session timeout if they have been active at least an hour
	if time.Now().Add(SessionTimeout - time.Hour).After(sess.ExpiresAt) {
		sess.ExpiresAt = time.Now().Add(SessionTimeout)
		if err := kotsStore.UpdateSessionExpiresAt(sess.ID, sess.ExpiresAt); err != nil {
			logger.Error(errors.Wrapf(err, "failed to update session expiry %s", sess.ID))
		}
		if signedTokenCookie != nil {
			origin := r.Header.Get("Origin")
			expiration := sess.ExpiresAt
			tokenCookie, err := session.GetSessionCookie(auth, expiration, origin)
			if err != nil {
				logger.Error(errors.Wrapf(err, "failed to update session cookie expiry %s", sess.ID))
			} else {
				http.SetCookie(w, tokenCookie)
			}
		}
	}

	return sess, nil
}

func requireValidKOTSToken(w http.ResponseWriter, r *http.Request) error {
	if r.Header.Get("Authorization") == "" {
		w.WriteHeader(http.StatusUnauthorized)
		return errors.New("authorization header empty")
	}

	clientset, err := k8sutil.GetClientset()
	if err != nil {
		return errors.Wrap(err, "failed to get clientset")
	}

	secret, err := clientset.CoreV1().Secrets(util.PodNamespace).Get(context.TODO(), "kotsadm-authstring", metav1.GetOptions{})
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
