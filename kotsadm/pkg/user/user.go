package user

import (
	"context"
	"os"

	"github.com/pkg/errors"
	"github.com/replicatedhq/kots/kotsadm/pkg/persistence"
	"golang.org/x/crypto/bcrypt"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
)

type User struct {
	ID string
}

func LogIn(password string) (*User, error) {
	cfg, err := config.GetConfig()
	if err != nil {
		return nil, errors.Wrap(err, "failed to get cluster config")
	}

	clientset, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create kubernetes clientset")
	}

	var shaBytes []byte
	existingPassword, err := clientset.CoreV1().Secrets(os.Getenv("POD_NAMESPACE")).Get(context.TODO(), "kotsadm-password", metav1.GetOptions{})
	if err != nil {
		// either no existing password secret or unable to get it
		// so instead we fallback to the environment variable
		shaBytes = []byte(os.Getenv("SHARED_PASSWORD_BCRYPT"))
	} else {
		shaBytes = existingPassword.Data["passwordBcrypt"]
	}

	if err := bcrypt.CompareHashAndPassword(shaBytes, []byte(password)); err != nil {
		if err == bcrypt.ErrMismatchedHashAndPassword {
			return nil, nil
		}

		return nil, errors.Wrap(err, "failed to compare password")
	}

	return &User{
		ID: "000000",
	}, nil
}

func LogOut(id string) error {
	db := persistence.MustGetPGSession()
	query := `delete from session where id = $1`

	_, err := db.Exec(query, id)
	if err != nil {
		return errors.Wrap(err, "failed to exec")
	}

	return nil
}
