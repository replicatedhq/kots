package kotsadm

import (
	"fmt"
	"os"

	"github.com/google/uuid"
	"github.com/manifoldco/promptui"
	"github.com/pkg/errors"
	"golang.org/x/crypto/bcrypt"
	corev1 "k8s.io/api/core/v1"
	kuberneteserrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

func ensureSecrets(deployOptions *DeployOptions, clientset *kubernetes.Clientset) error {
	if err := ensureJWTSessionSecret(deployOptions.Namespace, clientset); err != nil {
		return errors.Wrap(err, "failed to ensure jwt session secret")
	}

	if err := ensurePostgresSecret(deployOptions.Namespace, clientset); err != nil {
		return errors.Wrap(err, "failed to ensure postgres secret")
	}

	if err := ensureSharedPasswordSecret(deployOptions, clientset); err != nil {
		return errors.Wrap(err, "failed to ensure shared password secret")
	}

	if err := ensureS3Secret(deployOptions.Namespace, clientset); err != nil {
		return errors.Wrap(err, "failed to ensure s3 secret")
	}

	return nil
}

func ensureS3Secret(namespace string, clientset *kubernetes.Clientset) error {
	_, err := clientset.CoreV1().Secrets(namespace).Get("kotsadm-minio", metav1.GetOptions{})
	if err != nil {
		if !kuberneteserrors.IsNotFound(err) {
			return errors.Wrap(err, "failed to get existing s3 secret")
		}

		secret := &corev1.Secret{
			TypeMeta: metav1.TypeMeta{
				APIVersion: "v1",
				Kind:       "Secret",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      "kotsadm-minio",
				Namespace: namespace,
			},
			Data: map[string][]byte{
				"accesskey": []byte(minioAccessKey),
				"secretkey": []byte(minioSecret),
			},
		}

		_, err := clientset.CoreV1().Secrets(namespace).Create(secret)
		if err != nil {
			return errors.Wrap(err, "failed to create s3 secret")
		}
	}

	return nil
}

func ensureJWTSessionSecret(namespace string, clientset *kubernetes.Clientset) error {
	_, err := clientset.CoreV1().Secrets(namespace).Get("kotsadm-session", metav1.GetOptions{})
	if err != nil {
		if !kuberneteserrors.IsNotFound(err) {
			return errors.Wrap(err, "failed to get existing session secret")
		}

		secret := &corev1.Secret{
			TypeMeta: metav1.TypeMeta{
				APIVersion: "v1",
				Kind:       "Secret",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      "kotsadm-session",
				Namespace: namespace,
			},
			Data: map[string][]byte{
				"key": []byte(uuid.New().String()),
			},
		}

		_, err := clientset.CoreV1().Secrets(namespace).Create(secret)
		if err != nil {
			return errors.Wrap(err, "failed to create jwt session secret")
		}
	}

	return nil
}

func ensurePostgresSecret(namespace string, clientset *kubernetes.Clientset) error {
	_, err := clientset.CoreV1().Secrets(namespace).Get("kotsadm-postgres", metav1.GetOptions{})
	if err != nil {
		if !kuberneteserrors.IsNotFound(err) {
			return errors.Wrap(err, "failed to get existing postgres secret")
		}

		secret := &corev1.Secret{
			TypeMeta: metav1.TypeMeta{
				APIVersion: "v1",
				Kind:       "Secret",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      "kotsadm-postgres",
				Namespace: namespace,
			},
			Data: map[string][]byte{
				"uri": []byte(fmt.Sprintf("postgresql://kotsadm:%s@kotsadm-postgres/kotsadm?connect_timeout=10&sslmode=disable", postgresPassword)),
			},
		}

		_, err := clientset.CoreV1().Secrets(namespace).Create(secret)
		if err != nil {
			return errors.Wrap(err, "failed to create postgres secret")
		}
	}

	return nil
}

func ensureSharedPasswordSecret(deployOptions *DeployOptions, clientset *kubernetes.Clientset) error {
	if deployOptions.SharedPassword == "" {
		sharedPassword, err := promptForSharedPassword()
		if err != nil {
			return errors.Wrap(err, "failed to prompt for shared password")
		}

		deployOptions.SharedPassword = sharedPassword
	}

	bcryptPassword, err := bcrypt.GenerateFromPassword([]byte(deployOptions.SharedPassword), 10)
	if err != nil {
		return errors.Wrap(err, "failed to bcrypt shared password")
	}

	_, err = clientset.CoreV1().Secrets(deployOptions.Namespace).Get("kotsadm-password", metav1.GetOptions{})
	if err != nil {
		if !kuberneteserrors.IsNotFound(err) {
			return errors.Wrap(err, "failed to get existing password secret")
		}

		secret := &corev1.Secret{
			TypeMeta: metav1.TypeMeta{
				APIVersion: "v1",
				Kind:       "Secret",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      "kotsadm-password",
				Namespace: deployOptions.Namespace,
			},
			Data: map[string][]byte{
				"passwordBcrypt": bcryptPassword,
			},
		}

		_, err := clientset.CoreV1().Secrets(deployOptions.Namespace).Create(secret)
		if err != nil {
			return errors.Wrap(err, "failed to create password secret")
		}
	}

	return nil
}

func promptForSharedPassword() (string, error) {
	templates := &promptui.PromptTemplates{
		Prompt:  "{{ . | bold }} ",
		Valid:   "{{ . | green }} ",
		Invalid: "{{ . | red }} ",
		Success: "{{ . | bold }} ",
	}

	prompt := promptui.Prompt{
		Label:     "Enter a new password to be used for the Admin Console:",
		Templates: templates,
		Mask:      rune('•'),
		Validate: func(input string) error {
			if len(input) < 6 {
				return errors.New("please enter a longer password")
			}

			return nil
		},
	}

	for {
		result, err := prompt.Run()
		if err != nil {
			if err == promptui.ErrInterrupt {
				os.Exit(-1)
			}
			continue
		}

		return result, nil
	}

}
