package cli

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"strings"

	"github.com/pkg/errors"
	"github.com/replicatedhq/kots/pkg/docker/registry"
	"github.com/replicatedhq/kots/pkg/k8sutil"
	"github.com/replicatedhq/kots/pkg/kotsadm"
	kotsadmtypes "github.com/replicatedhq/kots/pkg/kotsadm/types"
	"github.com/replicatedhq/kots/pkg/logger"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	kuberneteserrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func AdminPushImagesCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:           "push-images [airgap filename] [registry host]",
		Short:         "Push admin console images",
		Long:          "Push admin console images from airgap bundle to a private registry",
		SilenceUsage:  true,
		SilenceErrors: false,
		PreRun: func(cmd *cobra.Command, args []string) {
			viper.BindPFlags(cmd.Flags())
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			v := viper.GetViper()

			if len(args) != 2 {
				cmd.Help()
				os.Exit(1)
			}

			log := logger.NewCLILogger(cmd.OutOrStdout())

			imageSource := args[0]
			endpoint := args[1]

			namespace := v.GetString("namespace")
			if namespace == "" {
				namespace = metav1.NamespaceDefault
			}

			username := v.GetString("registry-username")
			password := v.GetString("registry-password")
			if username == "" && password == "" {
				hostname, err := getHostnameFromEndpoint(endpoint)
				if err != nil {
					return errors.Wrap(err, "failed get hostname from endpoint")
				}

				u, p, err := getRegistryCredentialsFromSecret(hostname, v.GetString("namespace"))
				if err != nil {
					if !kuberneteserrors.IsNotFound(err) {
						log.Info("Failed to find registry credentials, will try to push anonymously: %v", err)
					}
				} else {
					username, password = u, p
				}
			}

			if registry.IsECREndpoint(endpoint) && username != "AWS" {
				var err error
				login, err := registry.GetECRLogin(endpoint, username, password)
				if err != nil {
					return errors.Wrap(err, "failed get ecr login")
				}
				username = login.Username
				password = login.Password
			}

			options := kotsadmtypes.PushImagesOptions{
				KotsadmTag: v.GetString("kotsadm-tag"),
				Registry: registry.RegistryOptions{
					Endpoint: endpoint,
					Username: username,
					Password: password,
				},
				ProgressWriter: os.Stdout,
			}

			_, err := os.Stat(imageSource)
			if err == nil {
				err := kotsadm.PushImages(imageSource, options)
				if err != nil {
					return errors.Wrap(err, "failed to push images")
				}
			} else if os.IsNotExist(err) {
				err := kotsadm.CopyImages(imageSource, options, namespace)
				if err != nil {
					return errors.Wrap(err, "failed to push images")
				}
			} else {
				return errors.Wrap(err, "failed to stat file")
			}

			return nil
		},
	}

	cmd.Flags().String("registry-username", "", "user name to use to authenticate with the registry")
	cmd.Flags().String("registry-password", "", "password to use to authenticate with the registry")

	cmd.Flags().String("kotsadm-tag", "", "set to override the tag of kotsadm. this may create an incompatible deployment because the version of kots and kotsadm are designed to work together")
	cmd.Flags().MarkHidden("kotsadm-tag")

	return cmd
}

func getRegistryCredentialsFromSecret(endpoint string, namespace string) (username string, password string, err error) {
	if namespace == "" {
		namespace = metav1.NamespaceDefault
	}

	clientset, err := k8sutil.GetClientset()
	if err != nil {
		err = errors.Wrap(err, "failed to get clientset")
		return
	}

	secret, err := clientset.CoreV1().Secrets(namespace).Get(context.TODO(), kotsadmtypes.PrivateKotsadmRegistrySecret, metav1.GetOptions{})
	if err != nil {
		err = errors.Wrap(err, "failed to get secret")
		return
	}

	dockerConfigJson := secret.Data[".dockerconfigjson"]
	if len(dockerConfigJson) == 0 {
		err = errors.New("no .dockerconfigjson found in secret")
		return
	}

	endpoint = strings.Split(endpoint, "/")[0]
	credentials, err := registry.GetCredentialsForRegistryFromConfigJSON(dockerConfigJson, endpoint)
	if err != nil {
		err = errors.Wrap(err, "failed to get credentials")
		return
	}

	username = credentials.Username
	password = credentials.Password
	return
}

func getHostnameFromEndpoint(endpoint string) (string, error) {
	if !strings.HasPrefix(endpoint, "http") {
		// url.Parse doesn't work without scheme
		endpoint = fmt.Sprintf("https://%s", endpoint)
	}

	parsed, err := url.Parse(endpoint)
	if err != nil {
		return "", errors.Wrap(err, "failed to parse endpoint")
	}

	return parsed.Hostname(), nil
}
