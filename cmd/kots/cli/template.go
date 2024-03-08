package cli

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/chzyer/readline"
	"github.com/pkg/errors"
	"github.com/replicatedhq/kots/pkg/logger"
	"github.com/replicatedhq/kots/pkg/pull"
	"github.com/replicatedhq/kots/pkg/template"
	kotsv1beta1 "github.com/replicatedhq/kotskinds/apis/kots/v1beta1"
	"github.com/replicatedhq/kotskinds/client/kotsclientset/scheme"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

type templateReplSession struct {
	b *template.Builder
	l *logger.CLILogger
}

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

			licenseFile := v.GetString("license-file")
			configFile := v.GetString("config-values")
			interactive := v.GetBool("interactive")

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

			// when no args are provided, render all mode, similar to helm template
			// we will utilize pull command to fetch and render manifests from upstream
			if len(args) == 0 && !interactive {
				err := pullAndRender(license.Spec.AppSlug, licenseFile, configFile)
				if err != nil {
					return errors.Wrap(err, "failed to render all templates")
				}
				return nil
			}

			// interactive mode
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

			log := logger.NewCLILogger(cmd.OutOrStdout())
			log.Initialize()

			if interactive {
				err := runInteractive(&builder, log)
				if err != nil {
					return errors.Wrap(err, "failed to run interactive mode")
				}
				return nil
			}

			// non-interactive mode
			rendered, err := builder.String(args[0])
			if err != nil {
				return errors.Wrap(err, "failed to render template")
			}

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

func createReplSession(builder *template.Builder, log *logger.CLILogger) *templateReplSession {
	return &templateReplSession{
		b: builder,
		l: log,
	}
}

func runInteractive(b *template.Builder, log *logger.CLILogger) error {
	repl := createReplSession(b, log)
	return repl.run()
}

func (r *templateReplSession) run() error {
	rl, err := readline.NewEx(&readline.Config{
		Prompt:            "> ",
		InterruptPrompt:   "^C",
		EOFPrompt:         "exit",
		HistorySearchFold: true,
		Stdin:             os.Stdin,
		Stdout:            os.Stdout,
		Stderr:            os.Stderr,
	})
	if err != nil {
		return errors.Wrap(err, "failed to initialize console")
	}
	defer rl.Close()

	for {
		line, err := rl.Readline()
		if err == readline.ErrInterrupt {
			if len(line) == 0 {
				break
			} else {
				continue
			}
		} else if err == io.EOF {
			break
		}

		out, exit, err := r.handle(line)
		if exit {
			break
		}
		r.l.Info(out)
	}

	return nil
}

func (r *templateReplSession) handle(line string) (string, bool, error) {
	switch {
	case strings.TrimSpace(line) == "exit":
		return "", true, nil
	case strings.TrimSpace(line) == "help":
		return r.help(), false, nil
	default:
		rendered, err := r.b.String(line)
		return rendered, false, err
	}
}

func (r *templateReplSession) help() string {
	return `
	Go to https://docs.replicated.com/reference/template-functions-about for a list of available template functions.
	Available commands:
 		help - display this help message
  		exit - exit the interactive console
`
}

func pullAndRender(appSlug string, licensePath string, configPath string) error {
	tempDir, err := os.MkdirTemp("", "kots-template")
	if err != nil {
		return errors.Wrap(err, "failed to create temp directory to render templates")
	}
	defer os.RemoveAll(tempDir)

	pullOptions := pull.PullOptions{
		RootDir:             tempDir,
		AppSlug:             appSlug,
		LicenseFile:         ExpandDir(licensePath),
		ConfigFile:          ExpandDir(configPath),
		Silent:              true,
		ExcludeAdminConsole: true,
	}

	upstream := pull.RewriteUpstream(appSlug)
	_, err = pull.Pull(upstream, pullOptions)

	if err != nil && err != pull.ErrConfigNeeded {
		return errors.Wrap(err, "failed to pull upstream")
	}

	// iterate over kotsKinds directory in tempDir and print all YAML contents
	kotsKindsDir := filepath.Join(tempDir, "kotsKinds")
	files, err := os.ReadDir(kotsKindsDir)
	if err != nil {
		return errors.Wrap(err, "failed to read kotsKinds directory")
	}

	var manifetsToRender []string
	for _, file := range files {
		if file.IsDir() {
			continue
		}
		content, err := os.ReadFile(filepath.Join(kotsKindsDir, file.Name()))
		if err != nil {
			return errors.Wrap(err, "failed to read file")
		}
		manifetsToRender = append(manifetsToRender, string(content))
	}
	for _, m := range manifetsToRender {
		fmt.Printf("---\n%s\n", m)
	}

	return nil
}
