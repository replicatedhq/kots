package kotsstore

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/pkg/errors"
	"github.com/replicatedhq/kots/pkg/k8sutil"
	"github.com/replicatedhq/kots/pkg/persistence"
	"github.com/replicatedhq/kots/pkg/util"
	kuberneteserrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var (
	ErrTooManyAttempts = errors.New("too many attempts")
	passwordSecretName = "kotsadm-password"
)

// GetSharedPasswordBcrypt will return the hash of the current password
// that can be used to validate an auth request. This is in the store pkg,
// but the data may be in the cluster or the database, depending on the
// configuration.
func (s *KOTSStore) GetSharedPasswordBcrypt() ([]byte, error) {
	// again, this isn't the right abstraction for this...  but it's
	// the store we have and could use some refactoring into the
	// store we want

	// for installations runnning in sqlite, we store
	// the bcrypted password in the params table
	if persistence.IsSQlite() {
		return s.getSharedPaswordBcryptFromDatabase()
	}

	// for installations not running in sqlite, we use the k8s api
	clientset, err := k8sutil.GetClientset()
	if err != nil {
		return nil, errors.Wrap(err, "failed to get k8s clientset")
	}

	var shaBytes []byte
	passwordSecret, err := clientset.CoreV1().Secrets(util.PodNamespace).Get(context.TODO(), passwordSecretName, metav1.GetOptions{})
	if err != nil {
		// either no existing password secret or unable to get it
		// so instead we fallback to the environment variable
		shaBytes = []byte(os.Getenv("SHARED_PASSWORD_BCRYPT"))
	} else {
		if passwordSecret.Labels == nil {
			passwordSecret.Labels = map[string]string{}
		}

		numAttempts, _ := strconv.Atoi(passwordSecret.Labels["numAttempts"])
		if numAttempts > 10 {
			return nil, ErrTooManyAttempts
		}

		shaBytes = passwordSecret.Data["passwordBcrypt"]
	}

	return shaBytes, nil
}

// getSharedPaswordBcryptFromDatabase will return the hash password from the
// database instead of the cluster. this is not an interface method
func (s *KOTSStore) getSharedPaswordBcryptFromDatabase() ([]byte, error) {
	db := persistence.MustGetDBSession()

	// check for too many attempts / locked out
	query := `select value from kotsadm_params where key = $1`
	row := db.QueryRow(query, "failed.login.count")
	failedLoginCount := "0"
	if err := row.Scan(&failedLoginCount); err != nil {
		if err != sql.ErrNoRows {
			return nil, errors.Wrap(err, "failed to scan invalid attempt count")
		}
	}

	i, err := strconv.Atoi(failedLoginCount)
	if err != nil {
		return nil, errors.Wrap(err, "failed to parse failed login count")
	}

	if i > 10 {
		return nil, ErrTooManyAttempts
	}

	// get the pasword now
	row = db.QueryRow(query, "password.bcrypt")
	var hash []byte

	// if it doesn't exist, return the one from the environment
	if err := row.Scan(&hash); err != nil {
		return []byte(os.Getenv("SHARED_PASSWORD_BCRYPT")), nil
	}

	return hash, nil
}

func (s *KOTSStore) FlagInvalidPassword() error {
	if persistence.IsSQlite() {
		return s.flagInvalidPasswordInDatabase()
	}

	clientset, err := k8sutil.GetClientset()
	if err != nil {
		return errors.Wrap(err, "failed to get k8s clientset")
	}

	for i := 0; ; i++ {
		secret, err := clientset.CoreV1().Secrets(util.PodNamespace).Get(context.TODO(), passwordSecretName, metav1.GetOptions{})
		if err != nil {
			if kuberneteserrors.IsNotFound(err) {
				return nil
			}
			return errors.Wrap(err, "failed to get password secret")
		}

		if secret.Labels == nil {
			secret.Labels = map[string]string{}
		}

		secret.Labels["lastFailure"] = fmt.Sprintf("%d", time.Now().Unix())
		numAttempts, _ := strconv.Atoi(secret.Labels["numAttempts"])
		secret.Labels["numAttempts"] = strconv.Itoa(numAttempts + 1)

		if _, err := clientset.CoreV1().Secrets(util.PodNamespace).Update(context.TODO(), secret, metav1.UpdateOptions{}); err != nil {
			if kuberneteserrors.IsConflict(err) {
				if i > 2 {
					return errors.New("failed to update password secret due to conflicts")
				}
				continue
			}
			return errors.Wrap(err, "failed to update password secret")
		}

		return nil
	}
}

func (s *KOTSStore) flagInvalidPasswordInDatabase() error {
	db := persistence.MustGetDBSession()

	query := `select value from kotsadm_params where key = $1`
	row := db.QueryRow(query, "failed.login.count")
	failedLoginCount := "0"
	if err := row.Scan(&failedLoginCount); err != nil {
		if err == sql.ErrNoRows {
			query = `insert into kotsadm_params (key, value) values ($1, $2)`
			if _, err := db.Exec(query, "failed.login.count", "1"); err != nil {
				return errors.Wrap(err, "failed to insert failed login count")
			}

			return nil
		} else {
			return errors.Wrap(err, "failed to scan invalid attempt count")
		}
	}

	i, err := strconv.Atoi(failedLoginCount)
	if err != nil {
		return errors.Wrap(err, "failed to parse failed login count")
	}
	i++

	query = `update kotsadm_params set value = $1 where key = $2`
	if _, err := db.Exec(query, strconv.Itoa(i), "failed.login.count"); err != nil {
		return errors.Wrap(err, "failed to update failed login count")
	}

	return nil

}

func (s *KOTSStore) FlagSuccessfulLogin() error {
	if persistence.IsSQlite() {
		return s.flagSuccessfulLoginInDatabase()
	}

	clientset, err := k8sutil.GetClientset()
	if err != nil {
		return errors.Wrap(err, "failed to get k8s clientset")
	}

	for i := 0; ; i++ {
		secret, err := clientset.CoreV1().Secrets(util.PodNamespace).Get(context.TODO(), passwordSecretName, metav1.GetOptions{})
		if err != nil {
			if kuberneteserrors.IsNotFound(err) {
				return nil
			}
			return errors.Wrap(err, "failed to get password secret")
		}

		if secret.Labels == nil {
			secret.Labels = map[string]string{}
		}

		secret.Labels["numAttempts"] = "0"
		if _, err := clientset.CoreV1().Secrets(util.PodNamespace).Update(context.TODO(), secret, metav1.UpdateOptions{}); err != nil {
			if kuberneteserrors.IsConflict(err) {
				if i > 2 {
					return errors.New("failed to update password secret due to conflicts")
				}
				continue
			}
			return errors.Wrap(err, "failed to update password secret")
		}

		return nil
	}
}

func (s *KOTSStore) flagSuccessfulLoginInDatabase() error {
	db := persistence.MustGetDBSession()

	// sqlite has a nice insert or replace, but let's be
	// nice and make sure this is pg copmatible

	query := `delete from kotsadm_params where key = $1`
	if _, err := db.Exec(query, "failed.login.count"); err != nil {
		return errors.Wrap(err, "failed to delete failed login count")
	}

	query = `insert into kotsadm_params (key, value) values ($1, $2)`
	if _, err := db.Exec(query, "failed.login.count", "0"); err != nil {
		return errors.Wrap(err, "failed to reset failed login count")
	}

	return nil
}
