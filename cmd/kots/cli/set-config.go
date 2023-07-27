package cli

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	cursor "github.com/ahmetalpbalkan/go-cursor"
	"github.com/pkg/errors"
	"github.com/replicatedhq/kots/pkg/auth"
	"github.com/replicatedhq/kots/pkg/k8sutil"
	configtypes "github.com/replicatedhq/kots/pkg/kotsadmconfig/types"
	"github.com/replicatedhq/kots/pkg/logger"
	"github.com/replicatedhq/kots/pkg/print"
	kotsv1beta1 "github.com/replicatedhq/kotskinds/apis/kots/v1beta1"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8syaml "sigs.k8s.io/yaml"
)

func SetConfigCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:           "config [appSlug] [KEY_1=VAL_1 ... KEY_N=VAL_N]",
		Short:         "Set config items for application",
		Long:          "",
		SilenceUsage:  true,
		SilenceErrors: false,
		PreRun: func(cmd *cobra.Command, args []string) {
			viper.BindPFlags(cmd.Flags())
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			v := viper.GetViper()

			if len(args) < 1 {
				cmd.Help()
				os.Exit(1)
			}

			fmt.Print(cursor.Hide())
			defer fmt.Print(cursor.Show())

			log := logger.NewCLILogger(cmd.OutOrStdout())
			appSlug := args[0]
			namespace, err := getNamespaceOrDefault(v.GetString("namespace"))
			if err != nil {
				return errors.Wrap(err, "failed to get namespace")
			}

			if v.GetBool("skip-preflights") && !v.GetBool("deploy") {
				log.Info("--skip-preflights will be ignored because --deploy is not set")
			}

			configValues, err := getConfigValuesFromArgs(v, args)
			if err != nil {
				return errors.Wrap(err, "failed to create config values from arguments")
			}

			clientset, err := k8sutil.GetClientset()
			if err != nil {
				return errors.Wrap(err, "failed to get clientset")
			}

			getPodName := func() (string, error) {
				return k8sutil.WaitForKotsadm(clientset, namespace, time.Second*5)
			}

			stopCh := make(chan struct{})
			defer close(stopCh)

			log.ActionWithoutSpinner("Updating %s configuration...", appSlug)

			localPort, errChan, err := k8sutil.PortForward(0, 3000, namespace, getPodName, false, stopCh, log)
			if err != nil {
				return errors.Wrap(err, "failed to start port forwarding")
			}

			go func() {
				select {
				case err := <-errChan:
					if err != nil {
						log.Error(err)
					}
				case <-stopCh:
				}
			}()

			authSlug, err := auth.GetOrCreateAuthSlug(clientset, namespace)
			if err != nil {
				return errors.Wrap(err, "failed to get kotsadm auth slug")
			}

			merge := v.GetBool("merge")
			if !merge && v.GetString("config-file") == "" {
				merge = true
			}

			requestPayload := map[string]interface{}{
				"configValues":   configValues,
				"merge":          merge,
				"deploy":         v.GetBool("deploy"),
				"skipPreflights": v.GetBool("skip-preflights"),
			}

			requestBody, err := json.Marshal(requestPayload)
			if err != nil {
				return errors.Wrap(err, "failed to marshal request json")
			}

			url := fmt.Sprintf("http://localhost:%d/api/v1/app/%s/config/values", localPort, url.QueryEscape(appSlug))
			newRequest, err := http.NewRequest("POST", url, bytes.NewBuffer(requestBody))
			if err != nil {
				return errors.Wrap(err, "failed to create http request")
			}
			newRequest.Header.Add("Authorization", authSlug)
			newRequest.Header.Add("Content-Type", "application/json")

			resp, err := http.DefaultClient.Do(newRequest)
			if err != nil {
				return errors.Wrap(err, "failed to execute http request")
			}
			defer resp.Body.Close()

			respBody, err := io.ReadAll(resp.Body)
			if err != nil {
				log.FinishSpinnerWithError()
				return errors.Wrap(err, "failed to read server response")
			}

			response := struct {
				Error            string                                   `json:"error"`
				ValidationErrors []configtypes.ConfigGroupValidationError `json:"validationErrors,omitempty"`
			}{}
			_ = json.Unmarshal(respBody, &response)

			if resp.StatusCode != http.StatusOK {
				if resp.StatusCode == http.StatusNotFound {
					return errors.Errorf("app with slug %s not found", appSlug)
				} else {
					if len(response.ValidationErrors) > 0 {
						print.ConfigValidationErrors(log, response.ValidationErrors)
						return errors.New(response.Error)
					}
					return errors.Wrapf(errors.New(response.Error), "unexpected status code from %v", resp.StatusCode)
				}
			}

			log.ActionWithoutSpinner("Done")

			return nil
		},
	}

	cmd.Flags().String("key", "", "name of a single key to set. This flag requires --value or --value-from-file flags")
	cmd.Flags().String("value", "", "the value to set for the key specified in the --key flag. This flag cannot be used with --value-from-file flag.")
	cmd.Flags().String("value-from-file", "", "path to the file containing the value to set for the key specified in the --key flag. This flag cannot be used with --value flag.")
	cmd.Flags().String("config-file", "", "path to a manifest containing config values (must be apiVersion: kots.io/v1beta1, kind: ConfigValues)")
	cmd.Flags().Bool("merge", false, "when set to true, only keys specified in config file will be updated. This flag can only be used when --config-file flag is used.")

	cmd.Flags().Bool("deploy", false, "when set, automatically deploy the latest version with the new configuration")
	cmd.Flags().Bool("skip-preflights", false, "set to true to skip preflight checks when deploying new version")

	return cmd
}

func getConfigValuesFromArgs(v *viper.Viper, args []string) ([]byte, error) {
	if fileName := v.GetString("config-file"); fileName != "" {
		if len(args) > 1 || v.GetString("key") != "" {
			return nil, errors.New("--config-file cannot be used with other key/value arguments")
		}

		data, err := os.ReadFile(fileName)
		if err != nil {
			return nil, errors.Wrap(err, "failed to load config from file")
		}
		return data, nil
	}

	configValues := &kotsv1beta1.ConfigValues{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "kots.io/v1beta1",
			Kind:       "ConfigValues",
		},
		Spec: kotsv1beta1.ConfigValuesSpec{
			Values: map[string]kotsv1beta1.ConfigValue{},
		},
	}

	key := v.GetString("key")
	value := v.GetString("value")
	valueFile := v.GetString("value-from-file")
	if key != "" {
		if value != "" && valueFile != "" {
			return nil, errors.New("only one of --value or --value-from-file can be specified")
		}

		if value != "" {
			configValues.Spec.Values[key] = kotsv1beta1.ConfigValue{
				Value: value,
			}
		} else if valueFile != "" {
			data, err := os.ReadFile(valueFile)
			if err != nil {
				return nil, errors.Wrap(err, "failed to load value from file")
			}
			configValues.Spec.Values[key] = kotsv1beta1.ConfigValue{
				Value: string(data),
			}
		} else {
			return nil, errors.New("--key flag requires either --value or --value-from-file flag")
		}
	}

	for i := 1; i < len(args); i++ {
		parts := strings.SplitN(args[i], "=", 2)
		if len(parts) != 2 {
			return nil, errors.Errorf("argument should have KEY=VALUE format: %s", args[i])
		}

		configValues.Spec.Values[parts[0]] = kotsv1beta1.ConfigValue{
			Value: parts[1],
		}
	}

	b, err := k8syaml.Marshal(configValues)
	if err != nil {
		return nil, errors.Wrap(err, "failed to marshal config values")
	}

	return b, nil
}
