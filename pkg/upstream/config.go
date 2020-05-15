package upstream

import (
	"encoding/base64"
	"fmt"

	"github.com/pkg/errors"
	kotsv1beta1 "github.com/replicatedhq/kots/kotskinds/apis/kots/v1beta1"
	"github.com/replicatedhq/kots/pkg/crypto"
)

func EncryptConfigValues(config *kotsv1beta1.Config, configValues *kotsv1beta1.ConfigValues, installation *kotsv1beta1.Installation) (*kotsv1beta1.ConfigValues, error) {
	if config == nil || configValues == nil {
		return nil, nil
	}

	if installation == nil {
		return nil, errors.New("missing installation")
	}

	cipher, err := crypto.AESCipherFromString(installation.Spec.EncryptionKey)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create cipher from installation spec")
	}

	updated := map[string]kotsv1beta1.ConfigValue{}

	updatedConfigValues := configValues.DeepCopy()
	for name, configValue := range configValues.Spec.Values {
		updated[name] = configValue

		if configValue.ValuePlaintext != "" {
			// ensure it's a password type
			configItemType := ""

			for _, group := range config.Spec.Groups {
				for _, item := range group.Items {
					if item.Name == name {
						configItemType = item.Type
						goto Found
					}
				}
			}
		Found:

			if configItemType == "" {
				return nil, errors.Errorf("Cannot encrypt item %q because item type was not found", name)
			}
			if configItemType != "password" {
				return nil, errors.Errorf("Cannot encrypt item %q because item type was %q (not password)", name, configItemType)
			}

			encrypted := cipher.Encrypt([]byte(configValue.ValuePlaintext))
			encoded := base64.StdEncoding.EncodeToString(encrypted)

			configValue.Value = encoded
			configValue.ValuePlaintext = ""

			updated[name] = configValue
		}
	}

	updatedConfigValues.Spec.Values = updated

	return updatedConfigValues, nil
}
