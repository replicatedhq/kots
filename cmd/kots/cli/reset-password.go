package cli

import (
	"os"
	"path/filepath"

	"github.com/manifoldco/promptui"
	"github.com/pkg/errors"
	"github.com/replicatedhq/kots/pkg/logger"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"golang.org/x/crypto/bcrypt"
	corev1 "k8s.io/api/core/v1"
	kuberneteserrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
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

			log := logger.NewLogger()

			if len(args) == 0 {
				cmd.Help()
				os.Exit(1)
			}

			log.ActionWithoutSpinner("Reset the admin console password for %s", args[0])
			newPassword, err := promptForNewPassword()
			if err != nil {
				os.Exit(1)
			}

			bcryptPassword, err := bcrypt.GenerateFromPassword([]byte(newPassword), 10)
			if err != nil {
				return err
			}

			cfg, err := config.GetConfig()
			if err != nil {
				return err
			}

			clientset, err := kubernetes.NewForConfig(cfg)
			if err != nil {
				return err
			}

			existingSecret, err := clientset.CoreV1().Secrets(args[0]).Get("kotsadm-password", metav1.GetOptions{})
			if err != nil {
				if !kuberneteserrors.IsNotFound(err) {
					return err
				}

				newSecret := &corev1.Secret{
					TypeMeta: metav1.TypeMeta{
						APIVersion: "v1",
						Kind:       "Secret",
					},
					ObjectMeta: metav1.ObjectMeta{
						Name:      "kotsadm-password",
						Namespace: args[0],
					},
					Data: map[string][]byte{
						"passwordBcrypt": []byte(bcryptPassword),
					},
				}

				_, err := clientset.CoreV1().Secrets(args[0]).Create(newSecret)
				if err != nil {
					return err
				}
			} else {
				existingSecret.Data["passwordBcrypt"] = []byte(bcryptPassword)

				_, err := clientset.CoreV1().Secrets(args[0]).Update(existingSecret)
				if err != nil {
					return err
				}
			}

			log.ActionWithoutSpinner("The admin console password has been reset")
			return nil
		},
	}

	cmd.Flags().String("kubeconfig", filepath.Join(homeDir(), ".kube", "config"), "the kubeconfig to use")

	return cmd
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
