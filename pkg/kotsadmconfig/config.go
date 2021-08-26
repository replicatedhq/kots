package kotsadmconfig

import (
	"context"

	"github.com/pkg/errors"
	kotsv1beta1 "github.com/replicatedhq/kots/kotskinds/apis/kots/v1beta1"
	"github.com/replicatedhq/kots/pkg/config"
	kotsconfig "github.com/replicatedhq/kots/pkg/config"
	"github.com/replicatedhq/kots/pkg/k8sutil"
	"github.com/replicatedhq/kots/pkg/kotsutil"
	"github.com/replicatedhq/kots/pkg/logger"
	"github.com/replicatedhq/kots/pkg/persistence"
	registrytypes "github.com/replicatedhq/kots/pkg/registry/types"
	"github.com/replicatedhq/kots/pkg/template"
	"github.com/replicatedhq/kots/pkg/util"
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

func NeedsConfiguration(appSlug string, sequence int64, isAirgap bool, kotsKinds *kotsutil.KotsKinds, registrySettings registrytypes.RegistrySettings) (bool, error) {
	log := logger.NewCLILogger()

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
	configValues, err := config.UnmarshalConfigValuesContent([]byte(configValuesSpec))
	if err != nil {
		log.Error(errors.Wrap(err, "failed to create config values"))
		configValues = map[string]template.ItemValue{}
	}

	localRegistry := template.LocalRegistry{
		Host:      registrySettings.Hostname,
		Namespace: registrySettings.Namespace,
		Username:  registrySettings.Username,
		Password:  registrySettings.Password,
		ReadOnly:  registrySettings.IsReadOnly,
	}

	versionInfo := template.VersionInfoFromInstallation(sequence, isAirgap, kotsKinds.Installation.Spec)
	appInfo := template.ApplicationInfo{Slug: appSlug}

	// rendered, err := kotsconfig.TemplateConfig(logger.NewCLILogger(), configSpec, configValuesSpec, licenseSpec, appSpec, identityConfigSpec, localRegistry, util.PodNamespace)
	config, err := kotsconfig.TemplateConfigObjects(kotsKinds.Config, configValues, kotsKinds.License, &kotsKinds.KotsApplication, localRegistry, &versionInfo, &appInfo, kotsKinds.IdentityConfig, util.PodNamespace)
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

// UpdateConfigValuesInDB it gets the config values from filesInDir and
// updates the app version config values in the db for the given sequence and app id
func UpdateConfigValuesInDB(filesInDir string, appID string, sequence int64) error {
	kotsKinds, err := kotsutil.LoadKotsKindsFromPath(filesInDir)
	if err != nil {
		return errors.Wrap(err, "failed to read kots kinds")
	}

	configValues, err := kotsKinds.Marshal("kots.io", "v1beta1", "ConfigValues")
	if err != nil {
		return errors.Wrap(err, "failed to marshal configvalues spec")
	}

	db := persistence.MustGetDBSession()
	query := `update app_version set config_values = $1 where app_id = $2 and sequence = $3`
	_, err = db.Exec(query, configValues, appID, sequence)
	if err != nil {
		return errors.Wrap(err, "failed to update config values in db")
	}

	return nil
}

func ReadConfigValuesFromInClusterSecret() (string, error) {
	log := logger.NewCLILogger()

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
