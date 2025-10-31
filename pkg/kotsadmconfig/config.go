package kotsadmconfig

import (
	"context"
	"encoding/base64"
	"fmt"
	"os"
	"strconv"

	"github.com/pkg/errors"
	kotsconfig "github.com/replicatedhq/kots/pkg/config"
	"github.com/replicatedhq/kots/pkg/crypto"
	"github.com/replicatedhq/kots/pkg/k8sutil"
	"github.com/replicatedhq/kots/pkg/kotsutil"
	"github.com/replicatedhq/kots/pkg/logger"
	registrytypes "github.com/replicatedhq/kots/pkg/registry/types"
	"github.com/replicatedhq/kots/pkg/template"
	"github.com/replicatedhq/kots/pkg/util"
	kotsv1beta1 "github.com/replicatedhq/kotskinds/apis/kots/v1beta1"
	"github.com/replicatedhq/kotskinds/multitype"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func IsRequiredItem(item kotsv1beta1.ConfigItem) bool {
	if !item.Required {
		return false
	}
	if item.Hidden || item.When == "false" {
		return false
	}
	return true
}

func IsUnsetItem(item kotsv1beta1.ConfigItem) bool {
	if item.Repeatable {
		for _, count := range item.CountByGroup {
			if count > 0 {
				return false
			}
		}
	}
	if item.Value.String() != "" {
		return false
	}
	if item.Default.String() != "" {
		return false
	}
	return true
}

// NeedsConfiguration returns true if the app has required config values that are not set
func NeedsConfiguration(appSlug string, sequence int64, isAirgap bool, kotsKinds *kotsutil.KotsKinds, registrySettings registrytypes.RegistrySettings) (bool, error) {
	log := logger.NewCLILogger(os.Stdout)

	configSpec, err := kotsKinds.Marshal("kots.io", "v1beta1", "Config")
	if err != nil {
		return false, errors.Wrap(err, "failed to marshal config spec")
	}

	if configSpec == "" {
		return false, nil
	}

	configValuesSpec, err := kotsKinds.Marshal("kots.io", "v1beta1", "ConfigValues")
	if err != nil {
		return false, errors.Wrap(err, "failed to marshal configvalues spec")
	}
	configValues, err := kotsconfig.UnmarshalConfigValuesContent([]byte(configValuesSpec))
	if err != nil {
		log.Error(errors.Wrap(err, "failed to create config values"))
		configValues = map[string]template.ItemValue{}
	}

	versionInfo := template.VersionInfoFromInstallationSpec(sequence, isAirgap, kotsKinds.Installation.Spec)
	appInfo := template.ApplicationInfo{Slug: appSlug}

	// rendered, err := kotsconfig.TemplateConfig(logger.NewCLILogger(os.Stdout), configSpec, configValuesSpec, licenseSpec, appSpec, identityConfigSpec, localRegistry, util.PodNamespace)
	config, err := kotsconfig.TemplateConfigObjects(kotsKinds.Config, configValues, kotsKinds.License, &kotsKinds.KotsApplication, registrySettings, &versionInfo, &appInfo, kotsKinds.IdentityConfig, util.PodNamespace, true)
	if err != nil {
		return false, errors.Wrap(err, "failed to template config")
	}

	if config == nil {
		return false, nil
	}

	for _, group := range config.Spec.Groups {
		if group.When == "false" {
			continue
		}
		for _, item := range group.Items {
			if IsRequiredItem(item) && IsUnsetItem(item) {
				return true, nil
			}
		}
	}
	return false, nil
}

func GetMissingRequiredConfig(configGroups []kotsv1beta1.ConfigGroup) ([]string, []string) {
	requiredItems := make([]string, 0, 0)
	requiredItemsTitles := make([]string, 0, 0)
	for _, group := range configGroups {
		if group.When == "false" {
			continue
		}
		for _, item := range group.Items {
			if IsRequiredItem(item) && IsUnsetItem(item) {
				requiredItems = append(requiredItems, item.Name)
				if item.Title != "" {
					requiredItemsTitles = append(requiredItemsTitles, item.Title)
				} else {
					requiredItemsTitles = append(requiredItemsTitles, item.Name)
				}
			}
		}
	}

	return requiredItems, requiredItemsTitles
}

func UpdateAppConfigValues(values map[string]kotsv1beta1.ConfigValue, configGroups []kotsv1beta1.ConfigGroup) map[string]kotsv1beta1.ConfigValue {
	for _, group := range configGroups {
		for _, item := range group.Items {
			if item.Type == "file" {
				v := values[item.Name]
				v.Filename = item.Filename
				values[item.Name] = v
			}
			if item.Value.Type == multitype.Bool {
				updatedValue := item.Value.BoolVal
				v := values[item.Name]
				v.Value = strconv.FormatBool(updatedValue)
				values[item.Name] = v
			} else if item.Value.Type == multitype.String {
				updatedValue := item.Value.String()
				if item.Type == "password" {
					// encrypt using the key
					// if the decryption succeeds, don't encrypt again
					_, err := util.DecryptConfigValue(updatedValue)
					if err != nil {
						updatedValue = base64.StdEncoding.EncodeToString(crypto.Encrypt([]byte(updatedValue)))
					}
				}

				v := values[item.Name]
				v.Value = updatedValue
				values[item.Name] = v
			}
			for _, repeatableValues := range item.ValuesByGroup {
				// clear out all variadic values for this group first
				for name, value := range values {
					if value.RepeatableItem == item.Name {
						delete(values, name)
					}
				}
				// add variadic groups back in declaratively
				for itemName, valueItem := range repeatableValues {
					v := values[itemName]
					v.Value = fmt.Sprintf("%v", valueItem)
					v.RepeatableItem = item.Name
					values[itemName] = v
				}
			}
		}
	}
	return values
}

// this is where config values that are passed to the install command are read from
func ReadConfigValuesFromInClusterSecret() (string, error) {
	log := logger.NewCLILogger(os.Stdout)

	clientset, err := k8sutil.GetClientset()
	if err != nil {
		return "", errors.Wrap(err, "failed to get k8s client set")
	}

	configValuesSecrets, err := clientset.CoreV1().Secrets(util.PodNamespace).List(context.TODO(), metav1.ListOptions{
		LabelSelector: "kots.io/automation=configvalues",
	})
	if err != nil {
		return "", errors.Wrap(err, "failed to list configvalues secrets")
	}

	// just get the first
	for _, configValuesSecret := range configValuesSecrets.Items {
		configValues, ok := configValuesSecret.Data["configvalues"]
		if !ok {
			log.Error(errors.Errorf("config values secret %q does not contain config values key", configValuesSecret.Name))
			continue
		}

		// delete it, these are one time use secrets
		err = clientset.CoreV1().Secrets(configValuesSecret.Namespace).Delete(context.TODO(), configValuesSecret.Name, metav1.DeleteOptions{})
		if err != nil {
			log.Error(errors.Errorf("error deleting config values secret: %v", err))
		}

		return string(configValues), nil
	}

	return "", nil
}
