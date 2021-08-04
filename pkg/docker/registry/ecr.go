package registry

import (
	"encoding/base64"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ecr"
	"github.com/pkg/errors"
)

type Login struct {
	Username string
	Password string
}

func IsECREndpoint(host string) bool {
	return strings.HasSuffix(host, ".amazonaws.com")
}

func GetECRLogin(ecrEndpoint, username, password string) (*Login, error) {
	token, err := GetECRBasicAuthToken(ecrEndpoint, username, password)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get basic auth token")
	}

	decoded, err := base64.StdEncoding.DecodeString(token)
	if err != nil {
		return nil, errors.Wrap(err, "failed to decode ECR token")
	}

	parts := strings.Split(string(decoded), ":")
	if len(parts) != 2 {
		return nil, errors.Wrap(err, "decode ECR token has invalid format")
	}

	return &Login{Username: parts[0], Password: parts[1]}, nil
}

func GetECRBasicAuthToken(ecrEndpoint, username, password string) (string, error) {
	registry, zone, err := parseECREndpoint(ecrEndpoint)
	if err != nil {
		return "", errors.Wrap(err, "failed to parse ECR endpoint")
	}

	ecrService := getECRService(username, password, zone)

	ecrToken, err := ecrService.GetAuthorizationToken(&ecr.GetAuthorizationTokenInput{
		RegistryIds: []*string{
			&registry,
		},
	})
	if err != nil {
		return "", errors.Wrap(err, "failed to get ECR token")
	}

	if len(ecrToken.AuthorizationData) == 0 {
		return "", errors.Errorf("Repo %s not accessible with specified credentials", ecrEndpoint)
	}

	return *ecrToken.AuthorizationData[0].AuthorizationToken, nil
}

func getECRService(accessKeyID, secretAccessKey, zone string) *ecr.ECR {
	awsConfig := &aws.Config{Region: aws.String(zone)}
	if accessKeyID != "" && secretAccessKey != "" {
		awsConfig.Credentials = credentials.NewStaticCredentials(accessKeyID, secretAccessKey, "")
	}
	return ecr.New(session.New(awsConfig))
}

func parseECREndpoint(endpoint string) (registry, zone string, err error) {
	splitEndpoint := strings.Split(endpoint, ".")
	if len(splitEndpoint) < 6 {
		return "", "", errors.Errorf("invalid ECR endpoint: %s", endpoint)
	}

	if splitEndpoint[1] != "dkr" || splitEndpoint[2] != "ecr" {
		return "", "", errors.Errorf("only dkr and ecr endpoints are supported: %s", endpoint)
	}

	return splitEndpoint[0], splitEndpoint[3], nil
}
