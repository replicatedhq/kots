package upstream

import (
	"io/ioutil"
	"os"
	"path"

	"github.com/google/uuid"
	"github.com/manifoldco/promptui"
	"github.com/pkg/errors"
	"github.com/replicatedhq/kots/pkg/kotsadm"
	kotsadmtypes "github.com/replicatedhq/kots/pkg/kotsadm/types"
	"github.com/replicatedhq/kots/pkg/upstream/types"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes/scheme"
)

type UpstreamSettings struct {
	SharedPassword       string
	SharedPasswordBcrypt string
	JWT                  string
	PostgresPassword     string
	APIEncryptionKey     string

	AutoCreateClusterToken string
	ObjectStoreOptions     kotsadmtypes.ObjectStoreConfig
}

func generateAdminConsoleFiles(renderDir string, sharedPassword string) ([]types.UpstreamFile, error) {
	if _, err := os.Stat(path.Join(renderDir, "admin-console")); os.IsNotExist(err) {
		settings := &UpstreamSettings{
			SharedPassword:         sharedPassword,
			AutoCreateClusterToken: uuid.New().String(),
			ObjectStoreOptions:	kotsadmtypes.DefaultObjectStore(),
		}
		return generateNewAdminConsoleFiles(settings)
	}

	existingFiles, err := ioutil.ReadDir(path.Join(renderDir, "admin-console"))
	if err != nil {
		return nil, errors.Wrap(err, "failed to read existing files")
	}

	settings := &UpstreamSettings{
		AutoCreateClusterToken: uuid.New().String(),
		ObjectStoreOptions:	kotsadmtypes.DefaultObjectStore(),
	}
	if err := loadUpstreamSettingsFromFiles(settings, renderDir, existingFiles); err != nil {
		return nil, errors.Wrap(err, "failed to find existing settings")
	}

	return generateNewAdminConsoleFiles(settings)
}

func loadUpstreamSettingsFromFiles(settings *UpstreamSettings, renderDir string, files []os.FileInfo) error {
	for _, fi := range files {
		data, err := ioutil.ReadFile(path.Join(renderDir, "admin-console", fi.Name()))
		if err != nil {
			return errors.Wrap(err, "failed to read file")
		}

		decode := scheme.Codecs.UniversalDeserializer().Decode
		obj, gvk, err := decode(data, nil, nil)
		if err != nil {
			continue
		}

		if gvk.Group == "" && gvk.Version == "v1" && gvk.Kind == "Secret" {
			if err := loadUpstreamSettingsFromSecret(settings, obj.(*corev1.Secret)); err != nil {
				return errors.Wrap(err, "load upstream settings from secret")
			}
		} else if gvk.Group == "apps" && gvk.Version == "v1" && gvk.Kind == "Deployment" {
			loadUpstreamSettingsFromDeployment(settings, obj.(*appsv1.Deployment))
		}
	}

	return nil
}

func loadUpstreamSettingsFromSecret(settings *UpstreamSettings, secret *corev1.Secret) error {
	switch secret.Name {
	case "kotsadm-password":
		settings.SharedPasswordBcrypt = string(secret.Data["passwordBcrypt"])
	case "kotsadm-session":
		settings.JWT = string(secret.Data["key"])
	case "kotsadm-postgres":
		settings.PostgresPassword = string(secret.Data["password"])
	case "kotsadm-encryption":
		settings.APIEncryptionKey = string(secret.Data["encryptionKey"])
	case "kotsadm-minio":
		if err := settings.ObjectStoreOptions.LoadSecretData(secret.Data); err != nil {
			return errors.Wrap(err, "load kotsadm-minio secret data")
		}
	}

	return nil
}

func loadUpstreamSettingsFromDeployment(settings *UpstreamSettings, deployment *appsv1.Deployment) {
	for _, c := range deployment.Spec.Template.Spec.Containers {
		for _, e := range c.Env {
			switch e.Name {
			case "AUTO_CREATE_CLUSTER_TOKEN", "KOTSADM_TOKEN":
				settings.AutoCreateClusterToken = e.Value
			}
		}
	}
}

func generateNewAdminConsoleFiles(settings *UpstreamSettings) ([]types.UpstreamFile, error) {
	upstreamFiles := []types.UpstreamFile{}

	deployOptions := kotsadmtypes.DeployOptions{
		Namespace:              "default",
		SharedPassword:         settings.SharedPassword,
		SharedPasswordBcrypt:   settings.SharedPasswordBcrypt,
		JWT:                    settings.JWT,
		PostgresPassword:       settings.PostgresPassword,
		APIEncryptionKey:       settings.APIEncryptionKey,
		AutoCreateClusterToken: settings.AutoCreateClusterToken,
		ObjectStoreOptions:     settings.ObjectStoreOptions,
	}

	if deployOptions.SharedPasswordBcrypt == "" && deployOptions.SharedPassword == "" {
		p, err := promptForSharedPassword()
		if err != nil {
			return nil, errors.Wrap(err, "failed to prompt for shared password")
		}

		deployOptions.SharedPassword = p
	}

	adminConsoleDocs, err := kotsadm.YAML(deployOptions)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get minio yaml")
	}
	for n, v := range adminConsoleDocs {
		upstreamFile := types.UpstreamFile{
			Path:    path.Join("admin-console", n),
			Content: v,
		}
		upstreamFiles = append(upstreamFiles, upstreamFile)
	}

	return upstreamFiles, nil
}

func promptForSharedPassword() (string, error) {
	templates := &promptui.PromptTemplates{
		Prompt:  "{{ . | bold }} ",
		Valid:   "{{ . | green }} ",
		Invalid: "{{ . | red }} ",
		Success: "{{ . | bold }} ",
	}

	prompt := promptui.Prompt{
		Label:     "Enter a new password to be used for the Admin Console:",
		Templates: templates,
		Mask:      rune('•'),
		Validate: func(input string) error {
			if len(input) < 6 {
				return errors.New("please enter a longer password")
			}

			return nil
		},
	}

	for {
		result, err := prompt.Run()
		if err != nil {
			if err == promptui.ErrInterrupt {
				os.Exit(-1)
			}
			continue
		}

		return result, nil
	}

}
