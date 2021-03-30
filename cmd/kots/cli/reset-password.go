package cli

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"strings"

	"github.com/manifoldco/promptui"
	"github.com/pkg/errors"
	"github.com/replicatedhq/kots/pkg/k8sutil"
	"github.com/replicatedhq/kots/pkg/logger"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"golang.org/x/crypto/bcrypt"
	corev1 "k8s.io/api/core/v1"
	kuberneteserrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func ResetPasswordCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:           "reset-password [namespace]",
		Short:         "Change the password on the admin console",
		Long:          `Change the password on the Admin Console`,
		SilenceUsage:  true,
		SilenceErrors: false,
		PreRun: func(cmd *cobra.Command, args []string) {
			viper.BindPFlags(cmd.Flags())
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			v := viper.GetViper()

			log := logger.NewCLILogger()

			// use namespace-as-arg if provided, else use namespace from -n/--namespace
			namespace := v.GetString("namespace")
			if len(args) == 1 {
				namespace = args[0]
			} else if len(args) > 1 {
				fmt.Printf("more than one argument supplied: %+v\n", args)
				os.Exit(1)
			}

			if namespace == "" {
				fmt.Printf("a namespace must be provided as an argument or via -n/--namespace\n")
				os.Exit(1)
			}

			newPassword := ""
			if v.GetBool("new-password-stdin") {
				p, err := passwordFromStdin()
				if err != nil {
					return errors.Wrap(err, "failed to read new password from stdin")
				}
				newPassword = p
			}

			if newPassword == "" {
				log.ActionWithoutSpinner("Reset the admin console password for %s", namespace)
				p, err := promptForNewPassword()
				if err != nil {
					return errors.Wrap(err, "failed to prompt for new password")
				}
				newPassword = p
			}

			if err := setKotsadmPassword(newPassword, namespace); err != nil {
				return errors.Wrap(err, "failed to set new password")
			}

			log.ActionWithoutSpinner("The admin console password has been reset")
			return nil
		},
	}

	cmd.Flags().Bool("new-password-stdin", false, "the new password to use for logging into the admin console. will read the password from stdin.")

	return cmd
}

func passwordFromStdin() (string, error) {
	p, err := ioutil.ReadAll(os.Stdin)
	if err != nil {
		return "", errors.Wrap(err, "failed to read stdin")
	}
	if len(p) < 6 {
		return "", errors.New("invalid password, minimum password length is 6 characters.")
	}
	password := string(p)

	password = strings.TrimSuffix(password, "\n")
	password = strings.TrimSuffix(password, "\r")

	return password, nil
}

func promptForNewPassword() (string, error) {
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

func setKotsadmPassword(password string, namespace string) error {
	bcryptPassword, err := bcrypt.GenerateFromPassword([]byte(password), 10)
	if err != nil {
		return errors.Wrap(err, "failed to create encrypt password")
	}

	clientset, err := k8sutil.GetClientset(kubernetesConfigFlags)
	if err != nil {
		return errors.Wrap(err, "failed to create k8s client")
	}

	existingSecret, err := clientset.CoreV1().Secrets(namespace).Get(context.TODO(), "kotsadm-password", metav1.GetOptions{})
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
				Name:      "kotsadm-password",
				Namespace: namespace,
			},
			Data: map[string][]byte{
				"passwordBcrypt": []byte(bcryptPassword),
			},
		}

		_, err := clientset.CoreV1().Secrets(namespace).Create(context.TODO(), newSecret, metav1.CreateOptions{})
		if err != nil {
			return errors.Wrap(err, "failed to create secret")
		}
	} else {
		existingSecret.Data["passwordBcrypt"] = []byte(bcryptPassword)
		delete(existingSecret.Labels, "numAttempts")
		delete(existingSecret.Labels, "lastFailure")

		_, err := clientset.CoreV1().Secrets(namespace).Update(context.TODO(), existingSecret, metav1.UpdateOptions{})
		if err != nil {
			return errors.Wrap(err, "failed to update secret")
		}
	}

	return nil
}
