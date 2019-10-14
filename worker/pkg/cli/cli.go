package cli

import (
	"os"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/pkg/errors"

	"github.com/replicatedhq/kotsadm/worker/pkg/config"
	"github.com/spf13/viper"
)

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() error {
	v := viper.New()
	v.AutomaticEnv()
	config.BindEnv(v, "mapstructure")

	c := config.New()
	v.Unmarshal(c)

	if os.Getenv("USE_EC2_PARAMETERS") != "" {
		sess := session.New(aws.NewConfig().WithCredentialsChainVerboseErrors(true))
		if err := config.UnmarshalSSM(c, config.GetSSMParams(sess)); err != nil {
			return errors.Wrap(err, "unmarshal ssm")
		}
	}

	return RootCmd(c, os.Stdout).Execute()
}
