package cli

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"

	"github.com/pkg/errors"
	"github.com/replicatedhq/kots/pkg/api/handlers/types"
	"github.com/replicatedhq/kots/pkg/auth"
	"github.com/replicatedhq/kots/pkg/crypto"
	"github.com/replicatedhq/kots/pkg/handlers"
	"github.com/replicatedhq/kots/pkg/k8sutil"
	"github.com/replicatedhq/kots/pkg/logger"
	"github.com/replicatedhq/kotskinds/apis/kots/v1beta1"
	"github.com/replicatedhq/kotskinds/multitype"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
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

	cmd.Flags().Int64("sequence", -1, "app sequence to retrieve config for")
	cmd.Flags().String("appslug", "", "app slug to retrieve config for")
	cmd.Flags().Bool("decrypt", false, "decrypt encrypted config items")
	cmd.Flags().Bool("current", false, "get config values for the currently deployed version of the app")

	return cmd
}

func getConfigCmd(cmd *cobra.Command, args []string) error {
	v := viper.GetViper()

	log := logger.NewCLILogger(cmd.OutOrStdout())

	stopCh := make(chan struct{})
	defer close(stopCh)

	clientset, err := k8sutil.GetClientset()
	if err != nil {
		return errors.Wrap(err, "failed to get clientset")
	}

	namespace, err := getNamespaceOrDefault(v.GetString("namespace"))
	if err != nil {
		return errors.Wrap(err, "failed to get namespace")
	}

	getPodName := func() (string, error) {
		return k8sutil.FindKotsadm(clientset, namespace)
	}

	appSlug := v.GetString("appslug")
	appSequence := v.GetInt64("sequence")
	decrypt := v.GetBool("decrypt")
	current := v.GetBool("current")

	localPort, errChan, err := k8sutil.PortForward(0, 3000, namespace, getPodName, false, stopCh, log)
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

	getAppsURL := fmt.Sprintf("http://localhost:%d/api/v1/apps", localPort)
	apps, err := getApps(getAppsURL, authSlug)
	if err != nil {
		return errors.Wrap(err, "failed to get apps")
	}

	if appSlug == "" {
		if len(apps.Apps) != 1 {
			return errors.New("appslug is required")
		}
		appSlug = apps.Apps[0].Slug
	}

	var foundApp *types.ResponseApp
	for _, a := range apps.Apps {
		if a.Slug == appSlug {
			foundApp = &a
			break
		}
	}
	if foundApp == nil {
		return errors.Errorf("app %s not found", appSlug)
	}

	if appSequence == -1 {
		if current && foundApp.Downstream.CurrentVersion != nil {
			appSequence = foundApp.Downstream.CurrentVersion.ParentSequence // this is the sequence of the currently deployed version
		} else {
			appSequence = foundApp.CurrentSequence // this is the sequence of the latest available version
		}
	}

	getConfigURL := fmt.Sprintf("http://localhost:%d/api/v1/app/%s/config/%d", localPort, appSlug, appSequence)
	config, err := getConfig(getConfigURL, authSlug)
	if err != nil {
		return errors.Wrap(err, "failed to get config")
	}

	if decrypt {
		config.ConfigGroups, err = decryptGroups(clientset, namespace, config.ConfigGroups)
		if err != nil {
			return errors.Wrap(err, "failed to decrypt config")
		}
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

func decryptGroups(clientset kubernetes.Interface, namespace string, groups []v1beta1.ConfigGroup) ([]v1beta1.ConfigGroup, error) {
	err := crypto.InitFromSecret(clientset, namespace)
	if err != nil {
		return nil, errors.Wrapf(err, "find password encryption information in %s", namespace)
	}

	outGroups := []v1beta1.ConfigGroup{}
	for _, group := range groups {
		outGroup := group.DeepCopy()
		for idx, item := range group.Items {
			if item.Type == "password" {
				// attempt to decrypt the password's value and default
				outGroup.Items[idx].Value = extractString(item.Value.String())
				outGroup.Items[idx].Default = extractString(item.Default.String())
			}
		}
		outGroups = append(outGroups, *outGroup)
	}

	return outGroups, nil
}

func extractString(item string) multitype.BoolOrString {
	if item == "" {
		return multitype.BoolOrString{}
	}
	decrypted, err := decryptString(item)
	if err != nil {
		return multitype.BoolOrString{} // don't fail if we can't decrypt
	}
	return multitype.BoolOrString{
		Type:   multitype.String,
		StrVal: decrypted,
	}
}

func decryptString(input string) (string, error) {
	decoded, err := base64.StdEncoding.DecodeString(input)
	if err != nil {
		return "", errors.Wrap(err, "failed to base64 decode")
	}

	decrypted, err := crypto.Decrypt(decoded)
	if err != nil {
		return "", errors.Wrap(err, "failed to decrypt")
	}

	return string(decrypted), nil
}
