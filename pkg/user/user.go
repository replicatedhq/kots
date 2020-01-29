package user

import (
	"os"

	"github.com/pkg/errors"
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

	existingPassword, err := clientset.CoreV1().Secrets(os.Getenv("POD_NAMESPACE")).Get("kotsadm-password", metav1.GetOptions{})
	if err != nil {
		// either no existing password secret or unable to get it
		return nil, errors.Wrap(err, "unable to get kotsadm-password secret")
	}

	if err := bcrypt.CompareHashAndPassword(existingPassword.Data["passwordBcrypt"], []byte(password)); err != nil {
		if err == bcrypt.ErrMismatchedHashAndPassword {
			return nil, nil
		}

		return nil, errors.Wrap(err, "failed to compare password")
	}

	return &User{
		ID: "000000",
	}, nil
}
