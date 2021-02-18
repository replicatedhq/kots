package ocistore

import (
	"context"
	"encoding/json"
	"os"
	"time"

	"github.com/pkg/errors"
	"github.com/replicatedhq/kots/pkg/logger"
	sessiontypes "github.com/replicatedhq/kots/pkg/session/types"
	usertypes "github.com/replicatedhq/kots/pkg/user/types"
	"github.com/segmentio/ksuid"
	corev1 "k8s.io/api/core/v1"
	kuberneteserrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

/* SessionStore
   The session_store will uses a single Kubernetes secret to store all sessions.
   The keys in the secret.data are the session id, and the values are the JSON marshalled session (userId, expireAt, etc)
   No data is actually written to the OCI registry in this store
*/

const (
	SessionSecretName = "kotsadm-sessions"
)

func (s OCIStore) CreateSession(forUser *usertypes.User, issuedAt time.Time, expiresAt time.Time, roles []string) (*sessiontypes.Session, error) {
	logger.Debug("creating session")

	randomID, err := ksuid.NewRandom()
	if err != nil {
		return nil, errors.Wrap(err, "failed to generate random session id")
	}

	id := randomID.String()

	sessionSecret, err := s.getSessionSecret()
	if err != nil {
		return nil, errors.Wrap(err, "failed to get session secret")
	}

	session := sessiontypes.Session{
		ID:        id,
		IssuedAt:  issuedAt,
		ExpiresAt: expiresAt,
		Roles:     roles,
		HasRBAC:   true,
	}

	b, err := json.Marshal(session)
	if err != nil {
		return nil, errors.Wrap(err, "failed to encoded session")
	}

	if sessionSecret.Data == nil {
		sessionSecret.Data = map[string][]byte{}
	}

	sessionSecret.Data[id] = b

	if err := s.updateSessionSecret(sessionSecret); err != nil {
		return nil, errors.Wrap(err, "failed to update session")
	}

	return s.GetSession(id)
}

func (s OCIStore) GetSession(id string) (*sessiontypes.Session, error) {
	secret, err := s.getSessionSecret()
	if err != nil {
		return nil, errors.Wrap(err, "failed to get session secret")
	}

	data, ok := secret.Data[id]
	if !ok {
		return nil, nil
	}

	session := sessiontypes.Session{}
	if err := json.Unmarshal(data, &session); err != nil {
		return nil, errors.Wrap(err, "failed to unmarshal session")
	}

	// sessions created before this change will not have IssuedAt
	if session.IssuedAt.IsZero() {
		session.IssuedAt = session.ExpiresAt.AddDate(0, 0, -14)
	}

	return &session, nil
}

func (s OCIStore) DeleteSession(id string) error {
	secret, err := s.getSessionSecret()
	if err != nil {
		return errors.Wrap(err, "failed to get session secret")
	}

	delete(secret.Data, id)

	if err := s.updateSessionSecret(secret); err != nil {
		return errors.Wrap(err, "failed to update session secret")
	}

	return nil
}

func (s OCIStore) getSessionSecret() (*corev1.Secret, error) {
	clientset, err := s.GetClientset()
	if err != nil {
		return nil, errors.Wrap(err, "failed to get clientset")
	}

	existingSecret, err := clientset.CoreV1().Secrets(os.Getenv("POD_NAMESPACE")).Get(context.TODO(), SessionSecretName, metav1.GetOptions{})
	if err != nil && !kuberneteserrors.IsNotFound(err) {
		return nil, errors.Wrap(err, "failed to get secret")
	} else if kuberneteserrors.IsNotFound(err) {
		secret := corev1.Secret{
			TypeMeta: metav1.TypeMeta{
				APIVersion: "v1",
				Kind:       "Secret",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      SessionSecretName,
				Namespace: os.Getenv("POD_NAMESPACE"),
			},
			Data: map[string][]byte{},
		}

		createdSecret, err := clientset.CoreV1().Secrets(os.Getenv("POD_NAMESPACE")).Create(context.TODO(), &secret, metav1.CreateOptions{})
		if err != nil {
			return nil, errors.Wrap(err, "failed to create session secret")
		}

		return createdSecret, nil
	}

	return existingSecret, nil
}

func (s OCIStore) updateSessionSecret(secret *corev1.Secret) error {
	clientset, err := s.GetClientset()
	if err != nil {
		return errors.Wrap(err, "failed to get clientset")
	}

	if _, err := clientset.CoreV1().Secrets(os.Getenv("POD_NAMESPACE")).Update(context.TODO(), secret, metav1.UpdateOptions{}); err != nil {
		return errors.Wrap(err, "failed to update session secret")
	}

	return nil
}
