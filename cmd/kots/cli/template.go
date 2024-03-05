package cli

import (
	"os"

	"github.com/pkg/errors"
	"github.com/replicatedhq/kots/pkg/logger"
	"github.com/replicatedhq/kots/pkg/pull"
	"github.com/replicatedhq/kots/pkg/template"
	kotsv1beta1 "github.com/replicatedhq/kotskinds/apis/kots/v1beta1"
	"github.com/replicatedhq/kotskinds/client/kotsclientset/scheme"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func TemplateCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:           "template",
		Short:         "Render template values based on given contexts (e.g. License, Config)",
		Long:          "Render template values based on given contexts (e.g. License, Config)",
		SilenceUsage:  true,
		SilenceErrors: false,
		PreRun: func(cmd *cobra.Command, args []string) {
			viper.BindPFlags(cmd.Flags())
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			v := viper.GetViper()

			if len(args) == 0 {
				cmd.Help()
				os.Exit(1)
			}

			licenseFile := v.GetString("license-file")
			configFile := v.GetString("config-values")
			// interactive := v.GetBool("interactive")

			license, err := parseLicenseFile(licenseFile)
			if err != nil {
				return errors.Wrap(err, "failed to parse --license-file")
			}

			config, err := pull.ParseConfigValuesFromFile(configFile)
			if err != nil {
				return errors.Wrap(err, "failed to parse --config-values")
			}

			configCtx, err := createConfigContext(config)
			if err != nil {
				return errors.Wrap(err, "failed to create config context")
			}

			// TODO: support other contexts
			builderOptions := template.BuilderOptions{
				ExistingValues: configCtx,
				License:        license,
				DecryptValues:  true,
			}

			builder, _, err := template.NewBuilder(builderOptions)
			if err != nil {
				return errors.Wrap(err, "failed to create template builder")
			}
			rendered, err := builder.String(args[0])
			if err != nil {
				return errors.Wrap(err, "failed to render template")
			}

			log := logger.NewCLILogger(cmd.OutOrStdout())
			log.Initialize()
			log.Info(rendered)

			return nil
		},
	}

	cmd.Flags().String("license-file", "", "path to a license file to use when download a replicated app")
	cmd.Flags().String("config-values", "", "path to a manifest containing config values (must be apiVersion: kots.io/v1beta1, kind: ConfigValues)")
	cmd.Flags().Bool("interactive", false, "provides an interactive command-line console for evaluating template values")

	cmd.MarkFlagRequired("license-file")
	cmd.MarkFlagRequired("config-values")

	return cmd
}

func parseLicenseFile(licenseFile string) (*kotsv1beta1.License, error) {
	licenseData, err := os.ReadFile(licenseFile)
	if err != nil {
		return nil, errors.Wrap(err, "failed to read license file")
	}
	decode := scheme.Codecs.UniversalDeserializer().Decode
	decoded, gvk, err := decode(licenseData, nil, nil)
	if err != nil {
		return nil, errors.Wrap(err, "failed to decode license file")
	}
	if gvk.Group != "kots.io" || gvk.Version != "v1beta1" || gvk.Kind != "License" {
		return nil, errors.New("license file is not a Replicated license")
	}

	license := decoded.(*kotsv1beta1.License)

	return license, nil
}

func createConfigContext(configValues *kotsv1beta1.ConfigValues) (map[string]template.ItemValue, error) {
	ctx := map[string]template.ItemValue{}

	if configValues == nil {
		return ctx, nil
	}

	for k, v := range configValues.Spec.Values {
		ctx[k] = template.ItemValue{
			Value:          v.Value,
			Default:        v.Default,
			Filename:       v.Filename,
			RepeatableItem: v.RepeatableItem,
		}
	}
	return ctx, nil
}
