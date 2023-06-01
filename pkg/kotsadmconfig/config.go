package kotsadmconfig

import (
	"context"
	"os"

	"github.com/pkg/errors"
	kotsv1beta1 "github.com/replicatedhq/kots/kotskinds/apis/kots/v1beta1"
	kotsconfig "github.com/replicatedhq/kots/pkg/config"
	"github.com/replicatedhq/kots/pkg/k8sutil"
	"github.com/replicatedhq/kots/pkg/kotsutil"
	"github.com/replicatedhq/kots/pkg/logger"
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
