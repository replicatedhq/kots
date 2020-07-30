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
	S3AccessKey          string
	S3SecretKey          string
	JWT                  string
	PostgresPassword     string
	APIEncryptionKey     string
	HTTPProxyEnvValue    string
	HTTPSProxyEnvValue   string
	NoProxyEnvValue      string

	AutoCreateClusterToken string
}

func generateAdminConsoleFiles(renderDir string, options types.WriteOptions) ([]types.UpstreamFile, error) {
	if _, err := os.Stat(path.Join(renderDir, "admin-console")); os.IsNotExist(err) {
		settings := &UpstreamSettings{
			SharedPassword:         options.SharedPassword,
			AutoCreateClusterToken: uuid.New().String(),
			HTTPProxyEnvValue:      options.HTTPProxyEnvValue,
			HTTPSProxyEnvValue:     options.HTTPSProxyEnvValue,
			NoProxyEnvValue:        options.NoProxyEnvValue,
		}
		return generateNewAdminConsoleFiles(settings)
	}

	existingFiles, err := ioutil.ReadDir(path.Join(renderDir, "admin-console"))
	if err != nil {
		return nil, errors.Wrap(err, "failed to read existing files")
	}

	settings := &UpstreamSettings{
		AutoCreateClusterToken: uuid.New().String(),
	}
	if err := loadUpstreamSettingsFromFiles(settings, renderDir, existingFiles); err != nil {
		return nil, errors.Wrap(err, "failed to find existing settings")
	}

	if options.HTTPProxyEnvValue != "" {
		settings.HTTPProxyEnvValue = options.HTTPProxyEnvValue
	}
	if options.HTTPSProxyEnvValue != "" {
		settings.HTTPSProxyEnvValue = options.HTTPSProxyEnvValue
	}
	if options.NoProxyEnvValue != "" {
		settings.NoProxyEnvValue = options.NoProxyEnvValue
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
			loadUpstreamSettingsFromSecret(settings, obj.(*corev1.Secret))
		} else if gvk.Group == "apps" && gvk.Version == "v1" && gvk.Kind == "Deployment" {
			loadUpstreamSettingsFromDeployment(settings, obj.(*appsv1.Deployment))
		}
	}

	return nil
}

func loadUpstreamSettingsFromSecret(settings *UpstreamSettings, secret *corev1.Secret) {
	switch secret.Name {
	case "kotsadm-password":
		settings.SharedPasswordBcrypt = string(secret.Data["passwordBcrypt"])
	case "kotsadm-minio":
		settings.S3AccessKey = string(secret.Data["accesskey"])
		settings.S3SecretKey = string(secret.Data["secretkey"])
	case "kotsadm-session":
		settings.JWT = string(secret.Data["key"])
	case "kotsadm-postgres":
		settings.PostgresPassword = string(secret.Data["password"])
	case "kotsadm-encryption":
		settings.APIEncryptionKey = string(secret.Data["encryptionKey"])
	}
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
		S3AccessKey:            settings.S3AccessKey,
		S3SecretKey:            settings.S3SecretKey,
		JWT:                    settings.JWT,
		PostgresPassword:       settings.PostgresPassword,
		APIEncryptionKey:       settings.APIEncryptionKey,
		AutoCreateClusterToken: settings.AutoCreateClusterToken,
		HTTPProxyEnvValue:      settings.HTTPProxyEnvValue,
		HTTPSProxyEnvValue:     settings.HTTPSProxyEnvValue,
		NoProxyEnvValue:        settings.NoProxyEnvValue,
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
