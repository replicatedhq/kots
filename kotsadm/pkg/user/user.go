package user

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"sync"
	"time"

	"github.com/pkg/errors"
	"github.com/replicatedhq/kots/kotsadm/pkg/logger"
	usertypes "github.com/replicatedhq/kots/kotsadm/pkg/user/types"
	"golang.org/x/crypto/bcrypt"
	kuberneteserrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
)

var (
	loginMutex         sync.Mutex
	passwordSecretName = "kotsadm-password"
	ErrInvalidPassword = errors.New("invalid password")
	ErrTooManyAttempts = errors.New("too many attempts")
)

func LogIn(password string) (*usertypes.User, error) {
	cfg, err := config.GetConfig()
	if err != nil {
		return nil, errors.Wrap(err, "failed to get cluster config")
	}

	clientset, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create kubernetes clientset")
	}

	var shaBytes []byte
	passwordSecret, err := clientset.CoreV1().Secrets(os.Getenv("POD_NAMESPACE")).Get(context.TODO(), passwordSecretName, metav1.GetOptions{})
	if err != nil {
		// either no existing password secret or unable to get it
		// so instead we fallback to the environment variable
		shaBytes = []byte(os.Getenv("SHARED_PASSWORD_BCRYPT"))
	} else {
		numAttempts, _ := strconv.Atoi(passwordSecret.Labels["numAttempts"])
		if numAttempts > 10 {
			return nil, ErrTooManyAttempts
		}

		shaBytes = passwordSecret.Data["passwordBcrypt"]
	}

	if err := bcrypt.CompareHashAndPassword(shaBytes, []byte(password)); err != nil {
		if err == bcrypt.ErrMismatchedHashAndPassword {
			if err := flagInvalidPassword(clientset); err != nil {
				logger.Infof("failed to flag failed login: %v", err)
			}
			return nil, ErrInvalidPassword
		}

		return nil, errors.Wrap(err, "failed to compare password")
	}

	if err := flagSuccessfulLogin(clientset); err != nil {
		logger.Error(errors.Wrap(err, "failed to flag successful login"))
	}

	return &usertypes.User{
		ID: "000000",
	}, nil
}

func flagSuccessfulLogin(clientset kubernetes.Interface) error {
	loginMutex.Lock()
	defer loginMutex.Unlock()

	for i := 0; ; i++ {
		secret, err := clientset.CoreV1().Secrets(os.Getenv("POD_NAMESPACE")).Get(context.TODO(), passwordSecretName, metav1.GetOptions{})
		if err != nil {
			if kuberneteserrors.IsNotFound(err) {
				return nil
			}
			return errors.Wrap(err, "failed to get password secret")
		}

		secret.Labels["lastLogin"] = fmt.Sprintf("%d", time.Now().Unix())
		secret.Labels["numAttempts"] = "0"
		if _, err := clientset.CoreV1().Secrets(os.Getenv("POD_NAMESPACE")).Update(context.TODO(), secret, metav1.UpdateOptions{}); err != nil {
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

func flagInvalidPassword(clientset kubernetes.Interface) error {
	loginMutex.Lock()
	defer loginMutex.Unlock()

	for i := 0; ; i++ {
		secret, err := clientset.CoreV1().Secrets(os.Getenv("POD_NAMESPACE")).Get(context.TODO(), passwordSecretName, metav1.GetOptions{})
		if err != nil {
			if kuberneteserrors.IsNotFound(err) {
				return nil
			}
			return errors.Wrap(err, "failed to get password secret")
		}

		secret.Labels["lastFailure"] = fmt.Sprintf("%d", time.Now().Unix())
		numAttempts, _ := strconv.Atoi(secret.Labels["numAttempts"])
		secret.Labels["numAttempts"] = strconv.Itoa(numAttempts + 1)

		if _, err := clientset.CoreV1().Secrets(os.Getenv("POD_NAMESPACE")).Update(context.TODO(), secret, metav1.UpdateOptions{}); err != nil {
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
