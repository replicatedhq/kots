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
	"github.com/replicatedhq/kots/pkg/persistence"
	"github.com/replicatedhq/kots/pkg/template"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/scheme"
)

type ConfigOptions struct {
	ConfigSpec         string
	ConfigValuesSpec   string
	LicenseSpec        string
	IdentityConfigSpec string
	RegistryHost       string
	RegistryNamespace  string
	RegistryUser       string
	RegistryPassword   string
}

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
	if item.Value.String() != "" {
		return false
	}
	if item.Default.String() != "" {
		return false
	}
	return true
}

func NeedsConfiguration(opts ConfigOptions) (bool, error) {
	if opts.ConfigSpec == "" {
		return false, nil
	}

	localRegistry := template.LocalRegistry{
		Host:      opts.RegistryHost,
		Namespace: opts.RegistryNamespace,
		Username:  opts.RegistryUser,
		Password:  opts.RegistryPassword,
	}

	rendered, err := kotsconfig.TemplateConfig(logger.NewCLILogger(), opts.ConfigSpec, opts.ConfigValuesSpec, opts.LicenseSpec, opts.IdentityConfigSpec, localRegistry)
	if err != nil {
		return false, errors.Wrap(err, "failed to template config")
	}

	decode := scheme.Codecs.UniversalDeserializer().Decode
	decoded, gvk, err := decode([]byte(rendered), nil, nil)
	if err != nil {
		return false, errors.Wrap(err, "failed to decode config")
	}
	if gvk.Group != "kots.io" || gvk.Version != "v1beta1" || gvk.Kind != "Config" {
		return false, errors.Errorf("unexpected gvk found in metadata: %s/%s/%s", gvk.Group, gvk.Version, gvk.Kind)
	}

	renderedConfig := decoded.(*kotsv1beta1.Config)

	for _, group := range renderedConfig.Spec.Groups {
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

	db := persistence.MustGetPGSession()
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

	configValuesSecrets, err := clientset.CoreV1().Secrets(os.Getenv("POD_NAMESPACE")).List(context.TODO(), metav1.ListOptions{
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
