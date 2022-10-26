package kotsstore

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/pkg/errors"
	"github.com/replicatedhq/kots/pkg/k8sutil"
	"github.com/replicatedhq/kots/pkg/persistence"
	"github.com/replicatedhq/kots/pkg/util"
	"github.com/rqlite/gorqlite"
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
	query := `select value from kotsadm_params where key = ?`
	rows, err := db.QueryOneParameterized(gorqlite.ParameterizedStatement{
		Query:     query,
		Arguments: []interface{}{"failed.login.count"},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to query login count: %v: %v", err, rows.Err)
	}

	failedLoginCount := "0"
	if rows.Next() {
		if err := rows.Scan(&failedLoginCount); err != nil {
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
	rows, err = db.QueryOneParameterized(gorqlite.ParameterizedStatement{
		Query:     query,
		Arguments: []interface{}{"password.bcrypt"},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to query password: %v: %v", err, rows.Err)
	}
	if !rows.Next() {
		// doesn't exist, return the one from the environment
		return []byte(os.Getenv("SHARED_PASSWORD_BCRYPT")), nil
	}

	var hash []byte
	if err := rows.Scan(&hash); err != nil {
		return nil, errors.Wrap(err, "failed to scan password")
	}

	return hash, nil
}

func (s *KOTSStore) FlagInvalidPassword() error {
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

	query := `select value from kotsadm_params where key = ?`
	rows, err := db.QueryOneParameterized(gorqlite.ParameterizedStatement{
		Query:     query,
		Arguments: []interface{}{"failed.login.count"},
	})
	if err != nil {
		return fmt.Errorf("failed to query: %v: %v", err, rows.Err)
	}

	if !rows.Next() {
		query = `insert into kotsadm_params (key, value) values (?, ?)`
		wr, err := db.WriteOneParameterized(gorqlite.ParameterizedStatement{
			Query:     query,
			Arguments: []interface{}{"failed.login.count", "1"},
		})
		if err != nil {
			return fmt.Errorf("failed to insert failed login count: %v: %v", err, wr.Err)
		}
		return nil
	}

	failedLoginCount := "0"
	if err := rows.Scan(&failedLoginCount); err != nil {
		return errors.Wrap(err, "failed to scan invalid attempt count")
	}

	i, err := strconv.Atoi(failedLoginCount)
	if err != nil {
		return errors.Wrap(err, "failed to parse failed login count")
	}
	i++

	wr, err := db.WriteOneParameterized(gorqlite.ParameterizedStatement{
		Query:     `update kotsadm_params set value = ? where key = ?`,
		Arguments: []interface{}{strconv.Itoa(i), "failed.login.count"},
	})
	if err != nil {
		return fmt.Errorf("failed to update failed login count: %v: %v", err, wr.Err)
	}

	return nil

}

func (s *KOTSStore) FlagSuccessfulLogin() error {
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

	wr, err := db.WriteOneParameterized(gorqlite.ParameterizedStatement{
		Query:     `delete from kotsadm_params where key = ?`,
		Arguments: []interface{}{"failed.login.count"},
	})
	if err != nil {
		return fmt.Errorf("failed to delete failed login count: %v: %v", err, wr.Err)
	}

	wr, err = db.WriteOneParameterized(gorqlite.ParameterizedStatement{
		Query:     `insert into kotsadm_params (key, value) values (?, ?)`,
		Arguments: []interface{}{"failed.login.count", "0"},
	})
	if err != nil {
		return fmt.Errorf("failed to reset failed login count: %v: %v", err, wr.Err)
	}

	return nil
}

// GetPasswordUpdatedAt - returns the time the password was last updated
func (s *KOTSStore) GetPasswordUpdatedAt() (*time.Time, error) {
	clientset, err := k8sutil.GetClientset()
	if err != nil {
		return nil, errors.Wrap(err, "failed to get k8s clientset")
	}

	passwordSecret, err := clientset.CoreV1().Secrets(util.PodNamespace).Get(context.TODO(), util.PasswordSecretName, metav1.GetOptions{})
	if err != nil {
		if kuberneteserrors.IsNotFound(err) {
			//  similar to fallback case when password secret is not found and uses the default password from environment variable
			return nil, nil
		}
		return nil, errors.Wrap(err, "failed to get password secret")
	}

	var passwordUpdatedAt *time.Time
	updatedAtBytes, ok := passwordSecret.Data["passwordUpdatedAt"]
	if ok {
		updatedAt, err := time.Parse(time.RFC3339, string(updatedAtBytes))
		if err != nil {
			return nil, errors.Wrap(err, "failed to parse passwordUpdatedAt")
		}
		passwordUpdatedAt = &updatedAt
	}

	return passwordUpdatedAt, nil
}
