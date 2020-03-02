package app

import (
	"github.com/pkg/errors"
	kotsv1beta1 "github.com/replicatedhq/kots/kotskinds/apis/kots/v1beta1"
	kotsconfig "github.com/replicatedhq/kots/pkg/config"
	"github.com/replicatedhq/kots/pkg/logger"
	"github.com/replicatedhq/kots/pkg/template"
	"github.com/replicatedhq/kotsadm/pkg/kotsutil"
	"github.com/replicatedhq/kotsadm/pkg/persistence"
	"k8s.io/client-go/kubernetes/scheme"
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
	if item.Value.String() != "" {
		return false
	}
	if item.Default.String() != "" {
		return false
	}
	return true
}

func needsConfiguration(configSpec string, configValuesSpec string, licenseSpec string) (bool, error) {
	localRegistry := template.LocalRegistry{}
	rendered, err := kotsconfig.TemplateConfig(logger.NewLogger(), configSpec, configValuesSpec, licenseSpec, localRegistry)
	if err != nil {
		return false, errors.Wrap(err, "failed to template config")
	}

	decode := scheme.Codecs.UniversalDeserializer().Decode
	decoded, _, _ := decode([]byte(rendered), nil, nil)
	if err != nil {
		return false, errors.Wrap(err, "failed to decode config")
	}
	renderedConfig := decoded.(*kotsv1beta1.Config)

	for _, group := range renderedConfig.Spec.Groups {
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
		return errors.Wrap(err, "failed to update config values in d")
	}

	return nil
}
