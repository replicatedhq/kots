package cli

import (
	"fmt"
	"io/ioutil"
	"net/url"

	"github.com/replicatedhq/troubleshoot/pkg/preflight"

	"github.com/pkg/errors"
	"github.com/replicatedhq/kots/pkg/kotsutil"
	"github.com/replicatedhq/kots/pkg/logger"
	"github.com/replicatedhq/kots/pkg/template"
	"github.com/replicatedhq/troubleshoot/pkg/oci"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func PreflightCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:           "preflight",
		Short:         "Run preflights",
		Long:          `Run preflights without a running KOTS installation`,
		SilenceUsage:  false,
		SilenceErrors: false,
		PreRun: func(cmd *cobra.Command, args []string) {
			viper.BindPFlags(cmd.Flags())
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			log := logger.NewCLILogger(cmd.OutOrStdout())
			v := viper.GetViper()
			v.BindPFlags(cmd.Flags())

			preflightFile, err := cmd.Flags().GetString("preflight-spec")
			if err != nil {
				log.Errorf("failed to get preflight flag: %v\n", err)
				return err
			}

			configFile, err := cmd.Flags().GetString("config-spec")
			if err != nil {
				log.Errorf("failed to get config flag: %v\n", err)
				return err
			}

			err = renderAndRunPreflight(v.GetBool("interactive"), v.GetString("output"), v.GetString("format"), preflightFile, configFile)
			if err != nil {
				log.Errorf("failed to render and run preflight %s: %v\n", preflightFile, err)
				return err
			}
			return nil
		},
	}

	preflight.AddFlags(cmd.PersistentFlags())
	cmd.Flags().String("preflight-spec", "", "the filename or url of the Preflight spec")
	cmd.Flags().String("config-spec", "", "the filename of the Config spec")

	return cmd
}

func renderAndRunPreflight(interactive bool, output, format, preflightFile, configFile string) error {
	isUrl := true
	parsed, err := url.ParseRequestURI(preflightFile)
	if err != nil {
		isUrl = false
	}

	var prefBytes []byte
	if isUrl && parsed != nil && parsed.Scheme == "oci" {
		// attempt to pull
		content, err := oci.PullPreflightFromOCI(preflightFile)
		if err != nil {
			if err == oci.ErrNoRelease {
				return errors.Errorf("no release found for %s.\nCheck the oci:// uri for errors or contact the application vendor for support.", preflightFile)
			}

			return errors.Wrap(err, "failed to pull preflight from oci")
		}

		prefBytes = content
	} else {
		prefBytes, err = ioutil.ReadFile(preflightFile)
		if err != nil {
			return errors.Wrap(err, fmt.Sprintf("failed to read preflight file: %s", preflightFile))
		}
	}
	var renderedPreflight string
	renderedPreflight = string(prefBytes)

	// only try to template if a config file is provided
	if configFile != "" {
		fb, err := ioutil.ReadFile(configFile)
		if err != nil {
			return errors.Wrap(err, fmt.Sprintf("failed to read config file: %s", configFile))
		}

		config, err := kotsutil.LoadConfigFromBytes(fb)
		if err != nil {
			return errors.Wrap(err, "failed to load config from bytes")
		}

		vals := make(map[string]template.ItemValue)
		builderOptions := template.BuilderOptions{
			ConfigGroups:    config.Spec.Groups,
			ExistingValues:  vals,
			LocalRegistry:   template.LocalRegistry{},
			License:         nil,
			Application:     nil,
			VersionInfo:     nil,
			ApplicationInfo: nil,
			IdentityConfig:  nil,
			Namespace:       "",
			DecryptValues:   true,
		}

		builder, _, err := template.NewBuilder(builderOptions)
		if err != nil {
			return errors.Wrap(err, "failed to create new config context template builder")
		}

		renderedPreflight, err = builder.RenderTemplate("preflight", string(prefBytes))
		if err != nil {
			return errors.Wrap(err, "failed to render templates in preflight")
		}
	}

	return preflight.RunPreflights(interactive, output, format, renderedPreflight)
}
