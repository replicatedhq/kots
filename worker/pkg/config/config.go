package config

import (
	"log"
	"reflect"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ssm"
	"github.com/pkg/errors"

	"github.com/spf13/viper"
)

const EncryptedFlag = "encrypted"

type Config struct {
	LogLevel  string `mapstructure:"log_level"`
	LogFormat string `mapstructure:"log_format"`

	UseEC2Parameters string `mapstructure:"use_ec2_parameters"`
	AWSRegion        string `mapstructure:"aws_region"`

	PostgresURI string `mapstructure:"postgres_uri" ssm:"/shipcloud/postgres/uri,encrypted"`

	GithubPrivateKey    string `mapstructure:"github_private_key" ssm:"/shipcloud/github/app_private_key,encrypted"`
	GithubIntegrationID int    `mapstructure:"github_integration_id"`

	S3BucketName string `mapstructure:"s3_bucket_name" ssm:"/shipcloud/s3/ship_output_bucket"`

	DBPollInterval        time.Duration `mapstructure:"db_poll_interval"`
	WatchCreationInterval time.Duration `mapstructure:"watch_creation_interval"`
	InitServerAddress     string        `mapstructure:"init_server_address"`
	UpdateServerAddress   string        `mapstructure:"update_server_address"`
	EditServerAddress     string        `mapstructure:"edit_server_address"`
	WatchServerAddress    string        `mapstructure:"watch_server_address"`
	AnalyzeServerAddress  string        `mapstructure:"analyze_server_address"`

	AnalyzeImage      string `mapstructure:"analyze_image"`
	AnalyzeTag        string `mapstructure:"analyze_tag"`
	AnalyzePullPolicy string `mapstructure:"analyze_pull_policy"`

	TroubleshootImage      string `mapstructure:"troubleshoot_image"`
	TroubleshootTag        string `mapstructure:"troubleshoot_tag"`
	TroubleshootPullPolicy string `mapstructure:"troubleshoot_pull_policy"`

	ShipImage      string `mapstructure:"ship_image"`
	ShipTag        string `mapstructure:"ship_tag"`
	ShipPullPolicy string `mapstructure:"ship_pull_policy"`

	SMTPHost     string `mapstructure:"smtp_host" ssm:"/shipcloud/smtp/host"`
	SMTPFrom     string `mapstructure:"smtp_from" ssm:"/shipcloud/smtp/from"`
	SMTPFromName string `mapstructure:"smtp_from_name" ssm:"/shipcloud/smtp/from_name"`
	SMTPPort     int    `mapstructure:"smtp_port"`
	SMTPUser     string `mapstructure:"smtp_user" ssm:"/shipcloud/smtp/user"`
	SMTPPassword string `mapstructure:"smtp_password" ssm:"/shipcloud/smtp/password,encrypted"`

	GithubToken string `mapstructure:"github_token" ssm:"/shipcloud/github_token,encrypted"`

	S3Endpoint        string `mapstructure:"s3_endpoint" ssm:"/shipcloud/s3/endpooint"`
	S3BucketEndpoint  string `mapstructure:"s3_bucket_endpoint" ssm:"/shipcloud/s3/bucket_endpoint"`
	S3AccessKeyID     string `mapstructure:"s3_access_key_id" ssm:"/shipcloud/s3/access_key_id"`
	S3SecretAccessKey string `mapstructure:"s3_secret_access_key" ssm:"/shipcloud/s3/secret_access_key"`
}

type GetSSMParamsFunc func([]*string) (map[string]string, error)

func New() *Config {
	return &Config{
		LogLevel:              "info",
		LogFormat:             "json",
		InitServerAddress:     ":3000",
		WatchServerAddress:    ":3000",
		UpdateServerAddress:   ":3000",
		AnalyzeServerAddress:  ":3000",
		EditServerAddress:     ":3000",
		DBPollInterval:        time.Second * 2,
		WatchCreationInterval: time.Second * 5,
		PostgresURI:           "postgresql://",
		GithubPrivateKey:      "<<not a key>>",
		GithubIntegrationID:   0,
		S3BucketName:          "shipbucket",
		SMTPHost:              "mail",
		SMTPFrom:              "ship@replicated.com",
		SMTPFromName:          "Replicated Ship",
		SMTPPort:              587,
		SMTPUser:              "",
		SMTPPassword:          "",
		S3Endpoint:            "",
		S3BucketEndpoint:      "",
		S3AccessKeyID:         "",
		S3SecretAccessKey:     "",
	}
}

func BindEnv(v *viper.Viper, key string) {
	t := reflect.TypeOf(Config{})
	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		v.BindEnv(field.Tag.Get(key))
	}
}

func UnmarshalSSM(config *Config, getSSMParams GetSSMParamsFunc) error {
	t := reflect.TypeOf(Config{})
	target := reflect.ValueOf(config)

	ssmNames := []*string{}
	ssmNameToFieldMap := map[string]string{}
	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		ssmTag := field.Tag.Get("ssm")
		if ssmTag != "" {
			ssmName, _ := parseSSMStructTag(ssmTag)
			if ssmName != "" {
				ssmNameToFieldMap[ssmName] = field.Name
				ssmNames = append(ssmNames, aws.String(ssmName))
			}
		}
	}

	ssmParams, err := getSSMParams(ssmNames)
	if err != nil {
		return errors.Wrap(err, "get ssm params")
	}
	for ssmName, value := range ssmParams {
		if value != "" {
			targetField := target.Elem().FieldByName(ssmNameToFieldMap[ssmName])
			if targetField.IsValid() {
				targetField.SetString(value)
			}
		}
	}

	return nil
}

func parseSSMStructTag(tag string) (string, bool) {
	parts := strings.Split(tag, ",")
	paramName := parts[0]
	encrypted := false
	if len(parts) > 1 && parts[1] == EncryptedFlag {
		encrypted = true
	}
	return paramName, encrypted
}

func GetSSMParams(sess *session.Session) GetSSMParamsFunc {
	return func(ssmNames []*string) (map[string]string, error) {
		svc := ssm.New(sess)

		params := map[string]string{}
		batch := chunkSlice(ssmNames, 10)

		for _, names := range batch {
			input := &ssm.GetParametersInput{
				Names:          names,
				WithDecryption: aws.Bool(true),
			}
			output, err := svc.GetParameters(input)
			if err != nil {
				return params, err
			}

			for _, p := range output.InvalidParameters {
				log.Printf("Ssm param %s invalid", *p)
			}

			for _, p := range output.Parameters {
				params[*p.Name] = *p.Value
			}
		}

		return params, nil
	}
}

func chunkSlice(s []*string, n int) [][]*string {
	var chunked [][]*string
	for i := 0; i < len(s); i += n {
		end := i + n
		if end > len(s) {
			end = len(s)
		}
		chunked = append(chunked, s[i:end])
	}
	return chunked
}
