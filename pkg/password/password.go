package password

import (
	"context"
	"sync"
	"time"

	"github.com/pkg/errors"
	"github.com/replicatedhq/kots/pkg/kotsadm/types"
	"github.com/replicatedhq/kots/pkg/logger"
	"github.com/replicatedhq/kots/pkg/store"
	"github.com/replicatedhq/kots/pkg/util"
	"golang.org/x/crypto/bcrypt"
	corev1 "k8s.io/api/core/v1"
	kuberneteserrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

// passwordLock - mutex to prevent multiple password changes at the same time
var passwordLock = sync.Mutex{}

var (
	ErrCurrentPasswordDoesNotMatch  = errors.New("The current password provided is incorrect.")
	ErrNewPasswordTooShort          = errors.New("The new password must be at least 6 characters.")
	ErrNewPasswordShouldBeDifferent = errors.New("The new password must be different from the current password.")
)

// ValidatePasswordInput - will validate length and complexity of new password and check if it is different from current password
func ValidatePasswordInput(currentPassword string, newPassword string) error {
	if len(newPassword) < 6 {
		return ErrNewPasswordTooShort
	}

	if newPassword == currentPassword {
		return ErrNewPasswordShouldBeDifferent
	}
	return nil
}

// ValidateCurrentPassword - will compare the password with the stored password and return an error if they don't match
func ValidateCurrentPassword(kotsStore store.Store, currentPassword string) error {
	passwordLock.Lock()
	defer passwordLock.Unlock()

	shaBytes, err := kotsStore.GetSharedPasswordBcrypt()
	if err != nil {
		return errors.Wrap(err, "failed to get current shared password bcrypt")
	}

	if err := bcrypt.CompareHashAndPassword(shaBytes, []byte(currentPassword)); err != nil {
		if err == bcrypt.ErrMismatchedHashAndPassword {
			return ErrCurrentPasswordDoesNotMatch
		}

		return errors.Wrap(err, "failed to compare current password")
	}

	return nil
}

// ChangePassword - will change the password in the kotsadm secret
func ChangePassword(clientset kubernetes.Interface, namespace string, newPassword string) error {
	passwordLock.Lock()
	defer passwordLock.Unlock()

	shaBytes, err := bcrypt.GenerateFromPassword([]byte(newPassword), 10)
	if err != nil {
		return errors.Wrap(err, "failed to generate new encrypted password")
	}

	if err := setSharedPasswordBcrypt(clientset, namespace, shaBytes); err != nil {
		return errors.Wrap(err, "failed to set new shared password bcrypt")
	}

	return nil
}

// setSharedPasswordBcrypt - set the shared password bcrypt hash in the kotsadm secret
func setSharedPasswordBcrypt(clientset kubernetes.Interface, namespace string, bcryptPassword []byte) error {
	secretData := map[string][]byte{
		"passwordBcrypt":    []byte(bcryptPassword),
		"passwordUpdatedAt": []byte(time.Now().Format(time.RFC3339)),
	}

	existingPasswordSecret, err := clientset.CoreV1().Secrets(namespace).Get(context.TODO(), util.PasswordSecretName, metav1.GetOptions{})
	if err != nil {
		if !kuberneteserrors.IsNotFound(err) {
			return errors.Wrap(err, "failed to lookup secret")
		}

		newSecret := &corev1.Secret{
			TypeMeta: metav1.TypeMeta{
				APIVersion: "v1",
				Kind:       "Secret",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      util.PasswordSecretName,
				Namespace: namespace,
			},
			Data: secretData,
		}

		_, err := clientset.CoreV1().Secrets(namespace).Create(context.TODO(), newSecret, metav1.CreateOptions{})
		if err != nil {
			return errors.Wrap(err, "failed to create secret")
		}
	} else {
		existingPasswordSecret.Data = secretData

		delete(existingPasswordSecret.Labels, "numAttempts")
		delete(existingPasswordSecret.Labels, "lastFailure")

		_, err := clientset.CoreV1().Secrets(namespace).Update(context.TODO(), existingPasswordSecret, metav1.UpdateOptions{})
		if err != nil {
			return errors.Wrap(err, "failed to update secret")
		}
	}

	deleteAllSessions(clientset, namespace)
	return nil
}

// deleteAllSessions - delete all sessions in the session secret, log errors if they occur
func deleteAllSessions(clientset kubernetes.Interface, namespace string) {
	sessionSecret := &corev1.Secret{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "Secret",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      util.SessionsSecretName,
			Namespace: namespace,
			Labels:    types.GetKotsadmLabels(),
		},
		Data: map[string][]byte{},
	}

	_, err := clientset.CoreV1().Secrets(namespace).Update(context.TODO(), sessionSecret, metav1.UpdateOptions{})
	if err != nil {
		// as the password is already changed, log the error but don't fail (false positive case)
		logger.Errorf("failed to delete all sessions: %s", namespace, util.SessionsSecretName, err)
	}
}
