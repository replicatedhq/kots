package cli

import (
	"encoding/json"
	"fmt"
	"github.com/pkg/errors"
	"github.com/replicatedhq/kots/kotskinds/apis/kots/v1beta1"
	"github.com/replicatedhq/kots/pkg/auth"
	"github.com/replicatedhq/kots/pkg/handlers"
	"github.com/replicatedhq/kots/pkg/k8sutil"
	"github.com/replicatedhq/kots/pkg/logger"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"io/ioutil"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"net/http"
	"os"
	"sigs.k8s.io/yaml"
)

func GetConfigCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:           "config --sequence=1 --appslug=my-app",
		Short:         "Get config values for an application",
		Long:          "",
		SilenceUsage:  false,
		SilenceErrors: false,
		PreRun: func(cmd *cobra.Command, args []string) {
			viper.BindPFlags(cmd.Flags())
		},
		RunE: getConfigCmd,
	}

	cmd.Flags().Int("sequence", -1, "app sequence to retrieve config for")
	cmd.Flags().String("appslug", "", "app slug to retrieve config for")

	return cmd
}

func getConfigCmd(cmd *cobra.Command, args []string) error {
	v := viper.GetViper()

	log := logger.NewCLILogger()

	stopCh := make(chan struct{})
	defer close(stopCh)

	clientset, err := k8sutil.GetClientset()
	if err != nil {
		return errors.Wrap(err, "failed to get clientset")
	}

	namespace := v.GetString("namespace")
	if err := validateNamespace(namespace); err != nil {
		return errors.Wrap(err, "failed to validate namespace")
	}

	appSlug := v.GetString("appslug")
	if appSlug == "" {
		return errors.New("appslug is required")
	}

	appSequence := v.GetInt("sequence")
	if appSequence == -1 {
		return errors.New("sequence is required")
	}

	podName, err := k8sutil.FindKotsadm(clientset, namespace)
	if err != nil {
		return errors.Wrap(err, "failed to find kotsadm pod")
	}

	localPort, errChan, err := k8sutil.PortForward(0, 3000, namespace, podName, false, stopCh, log)
	if err != nil {
		log.FinishSpinnerWithError()
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
		log.FinishSpinnerWithError()
		log.Info("Unable to authenticate to the Admin Console running in the %s namespace. Ensure you have read access to secrets in this namespace and try again.", namespace)
		if v.GetBool("debug") {
			return errors.Wrap(err, "failed to get kotsadm auth slug")
		}
		os.Exit(2) // not returning error here as we don't want to show the entire stack trace to normal users
	}

	url := fmt.Sprintf("http://localhost:%d/api/v1/app/%s/config/%d", localPort, appSlug, appSequence)
	config, err := getConfig(url, authSlug)
	if err != nil {
		return errors.Wrap(err, "failed to get config")
	}

	values := configGroupToValues(config.ConfigGroups)
	configYaml, err := yaml.Marshal(values)
	if err != nil {
		return errors.Wrap(err, "failed to marshal config")
	}

	fmt.Print(string(configYaml))

	return nil
}

func getConfig(url string, authSlug string) (*handlers.CurrentAppConfigResponse, error) {
	newReq, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create request")
	}
	newReq.Header.Add("Content-Type", "application/json")
	newReq.Header.Add("Authorization", authSlug)

	resp, err := http.DefaultClient.Do(newReq)
	if err != nil {
		return nil, errors.Wrap(err, "failed to execute request")
	}
	defer resp.Body.Close()

	b, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, errors.Wrap(err, "failed to read")
	}

	config := &handlers.CurrentAppConfigResponse{}
	if err := json.Unmarshal(b, config); err != nil {
		return nil, errors.Wrap(err, "failed to unmarshal status")
	}

	if !config.Success {
		return nil, fmt.Errorf("failed to get config: %s", config.Error)
	}

	return config, nil
}

func configGroupToValues(groups []v1beta1.ConfigGroup) v1beta1.ConfigValues {
	extractedValues := v1beta1.ConfigValues{
		TypeMeta: v1.TypeMeta{
			APIVersion: "kots.io/v1beta1",
			Kind:       "ConfigValues",
		},
		Spec: v1beta1.ConfigValuesSpec{Values: map[string]v1beta1.ConfigValue{}},
	}

	for _, group := range groups {
		for _, item := range group.Items {
			extractedValues.Spec.Values[item.Name] = v1beta1.ConfigValue{
				Default:  item.Default.String(),
				Value:    item.Value.String(),
				Data:     item.Data,
				Filename: item.Filename,
			}
		}
	}

	return extractedValues
}
