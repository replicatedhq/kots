package cli

import (
	"encoding/base64"
	"fmt"
	"net/url"
	"os"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ecr"
	"github.com/pkg/errors"
	"github.com/replicatedhq/kots/pkg/docker/registry"
	"github.com/replicatedhq/kots/pkg/kotsadm"
	kotsadmtypes "github.com/replicatedhq/kots/pkg/kotsadm/types"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
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

			airgapArchive := args[0]
			endpoint := args[1]

			username := v.GetString("registry-username")
			password := v.GetString("registry-password")
			if isECR(endpoint) && username != "AWS" {
				var err error
				username, password, err = getECRLogin(endpoint, username, password)
				if err != nil {
					return errors.Wrap(err, "failed get ecr login")
				}
			}

			options := kotsadmtypes.PushImagesOptions{
				KotsadmTag: v.GetString("kotsadm-tag"),
				Registry: registry.RegistryOptions{
					Endpoint: args[1],
					Username: username,
					Password: password,
				},
				ProgressWriter: os.Stdout,
			}

			err := kotsadm.PushImages(airgapArchive, options)
			if err != nil {
				return errors.Wrap(err, "failed to push images")
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

func isECR(endpoint string) bool {
	if !strings.HasPrefix(endpoint, "http") {
		// url.Parse doesn't work without scheme
		endpoint = fmt.Sprintf("https://%s", endpoint)
	}

	parsed, err := url.Parse(endpoint)
	if err != nil {
		return false
	}

	return strings.HasSuffix(parsed.Hostname(), ".amazonaws.com")
}

func getECRLogin(endpoint string, keyID string, accessKey string) (string, string, error) {
	registry, zone, err := parseECREndpoint(endpoint)
	if err != nil {
		return "", "", errors.Wrap(err, "failed to parse ECR endpoint")
	}

	ecrService := getECRService(keyID, accessKey, zone)

	ecrToken, err := ecrService.GetAuthorizationToken(&ecr.GetAuthorizationTokenInput{
		RegistryIds: []*string{
			&registry,
		},
	})
	if err != nil {
		return "", "", errors.Wrap(err, "failed to get ecr token")
	}

	if len(ecrToken.AuthorizationData) == 0 {
		return "", "", errors.Errorf("repo %s not accessible with specified credentials", endpoint)
	}

	decoded, err := base64.StdEncoding.DecodeString(*ecrToken.AuthorizationData[0].AuthorizationToken)
	if err != nil {
		return "", "", errors.Wrap(err, "failed to decode ecr token")
	}

	parts := strings.Split(string(decoded), ":")
	if len(parts) != 2 {
		return "", "", errors.New("ecr token is not in user:password format")
	}

	username := parts[0]
	password := parts[1]

	return username, password, nil
}

func getECRService(accessKeyID, secretAccessKey, zone string) *ecr.ECR {
	awsConfig := &aws.Config{Region: aws.String(zone)}
	awsConfig.Credentials = credentials.NewStaticCredentials(accessKeyID, secretAccessKey, "")
	return ecr.New(session.New(awsConfig))
}

func parseECREndpoint(endpoint string) (registry, zone string, err error) {
	splitEndpoint := strings.Split(endpoint, ".")
	if len(splitEndpoint) < 6 {
		return "", "", errors.Errorf("invalid ecr endpoint: %s", endpoint)
	}

	if splitEndpoint[1] != "dkr" || splitEndpoint[2] != "ecr" {
		return "", "", errors.Errorf("only dkr and ecr endpoints are supported: %s", endpoint)
	}

	return splitEndpoint[0], splitEndpoint[3], nil
}
