package kotsstore

import (
	"context"
	"database/sql"
	"encoding/json"
	"sync"
	"time"

	"github.com/pkg/errors"
	"github.com/replicatedhq/kots/pkg/k8sutil"
	"github.com/replicatedhq/kots/pkg/logger"
	"github.com/replicatedhq/kots/pkg/persistence"
	sessiontypes "github.com/replicatedhq/kots/pkg/session/types"
	usertypes "github.com/replicatedhq/kots/pkg/user/types"
	"github.com/replicatedhq/kots/pkg/util"
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

var (
	sessionLock = sync.Mutex{}
)

type SessionMetadata struct {
	Roles []string
}

func (s *KOTSStore) migrateSessionsFromPostgres() error {
	logger.Debug("migrating sessions from postgres")

	db := persistence.MustGetDBSession()
	query := `select id, metadata, issued_at, expire_at from session`
	rows, err := db.Query(query)
	if err != nil {
		return errors.Wrap(err, "failed to query rows")
	}

	sessionSecret, err := s.getSessionSecret()
	if err != nil {
		return errors.Wrap(err, "failed to get session secret")
	}

	for rows.Next() {
		session := sessiontypes.Session{}

		var issuedAt sql.NullTime
		var expiresAt time.Time
		var metadataStr string
		if err := rows.Scan(&session.ID, &metadataStr, &issuedAt, &expiresAt); err != nil {
			return errors.Wrap(err, "failed to get session")
		}

		if metadataStr != "" {
			metadata := SessionMetadata{}
			if err := json.Unmarshal([]byte(metadataStr), &metadata); err != nil {
				return errors.Wrap(err, "failed to unmarshal session metadata")
			}
			session.HasRBAC = true
			session.Roles = metadata.Roles
		}

		// sessions created before this change will not have IssuedAt
		if issuedAt.Valid {
			session.IssuedAt = issuedAt.Time
		} else {
			session.IssuedAt = session.ExpiresAt.AddDate(0, 0, -14)
		}

		session.ExpiresAt = expiresAt

		b, err := json.Marshal(session)
		if err != nil {
			return errors.Wrap(err, "failed to encoded session")
		}

		if sessionSecret.Data == nil {
			sessionSecret.Data = map[string][]byte{}
		}

		sessionSecret.Data[session.ID] = b
	}

	err = s.saveSessionSecret(sessionSecret)
	if err != nil {
		return errors.Wrap(err, "failed to update session secre")
	}

	query = `delete from session`
	if _, err := db.Exec(query); err != nil {
		return errors.Wrap(err, "failed to delete sessions from pg")
	}

	return nil

}

func (s *KOTSStore) CreateSession(forUser *usertypes.User, issuedAt time.Time, expiresAt time.Time, roles []string) (*sessiontypes.Session, error) {
	sessionLock.Lock()
	defer sessionLock.Unlock()

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

	if err := s.saveSessionSecret(sessionSecret); err != nil {
		return nil, errors.Wrap(err, "failed to update session")
	}

	return &session, nil
}

func (s *KOTSStore) GetSession(id string) (*sessiontypes.Session, error) {
	sessionLock.Lock()
	defer sessionLock.Unlock()

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

func (s *KOTSStore) DeleteSession(id string) error {
	sessionLock.Lock()
	defer sessionLock.Unlock()

	s.sessionSecret = nil

	secret, err := s.getSessionSecret()
	if err != nil {
		return errors.Wrap(err, "failed to get session secret")
	}

	delete(secret.Data, id)

	if err := s.saveSessionSecret(secret); err != nil {
		return errors.Wrap(err, "failed to update session secret")
	}

	return nil
}

func (s *KOTSStore) getSessionSecret() (*corev1.Secret, error) {
	if s.sessionSecret != nil && time.Now().Before(s.sessionExpiration) {
		return s.sessionSecret, nil
	}

	clientset, err := k8sutil.GetClientset()
	if err != nil {
		return nil, errors.Wrap(err, "failed to get clientset")
	}

	secret := corev1.Secret{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "Secret",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      SessionSecretName,
			Namespace: util.PodNamespace,
		},
		Data: map[string][]byte{},
	}

	existingSecret, err := clientset.CoreV1().Secrets(util.PodNamespace).Get(context.TODO(), SessionSecretName, metav1.GetOptions{})
	if err == nil {
		secret.Data = existingSecret.DeepCopy().Data
	} else if err != nil && !kuberneteserrors.IsNotFound(err) {
		if canIgnoreEtcdError(err) && s.sessionSecret != nil {
			return s.sessionSecret, nil
		}
		return nil, errors.Wrap(err, "failed to get secret")
	} else if kuberneteserrors.IsNotFound(err) {
		_, err := clientset.CoreV1().Secrets(util.PodNamespace).Create(context.TODO(), &secret, metav1.CreateOptions{})
		if err != nil {
			return nil, errors.Wrap(err, "failed to create session secret")
		}
	}

	s.sessionExpiration = time.Now().Add(1 * time.Minute)
	s.sessionSecret = &secret

	return &secret, nil
}

func (s *KOTSStore) saveSessionSecret(secret *corev1.Secret) error {
	clientset, err := k8sutil.GetClientset()
	if err != nil {
		return errors.Wrap(err, "failed to get clientset")
	}

	existingSecret, err := clientset.CoreV1().Secrets(util.PodNamespace).Get(context.TODO(), SessionSecretName, metav1.GetOptions{})
	if err == nil {
		existingSecret.Data = secret.DeepCopy().Data
		if _, err := clientset.CoreV1().Secrets(util.PodNamespace).Update(context.TODO(), existingSecret, metav1.UpdateOptions{}); err != nil {
			return errors.Wrap(err, "failed to update session secret")
		}
	} else if err != nil && !kuberneteserrors.IsNotFound(err) {
		if canIgnoreEtcdError(err) && s.sessionSecret != nil {
			return nil
		}
		return errors.Wrap(err, "failed to get secret for update")
	} else if kuberneteserrors.IsNotFound(err) {
		_, err := clientset.CoreV1().Secrets(util.PodNamespace).Create(context.TODO(), secret, metav1.CreateOptions{})
		if err != nil {
			if canIgnoreEtcdError(err) && s.sessionSecret != nil {
				return nil
			}
			return errors.Wrap(err, "failed to create session secret")
		}
	}

	return nil
}
