package upstream

import (
	"io/fs"
	"os"
	"path"

	"github.com/google/uuid"
	"github.com/pkg/errors"
	"github.com/replicatedhq/kots/pkg/kotsadm"
	kotsadmtypes "github.com/replicatedhq/kots/pkg/kotsadm/types"
	"github.com/replicatedhq/kots/pkg/upstream/types"
	"github.com/replicatedhq/kots/pkg/util"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes/scheme"
)

type UpstreamSettings struct {
	Namespace              string
	SharedPassword         string
	SharedPasswordBcrypt   string
	S3AccessKey            string
	S3SecretKey            string
	JWT                    string
	RqlitePassword         string
	APIEncryptionKey       string
	HTTPProxyEnvValue      string
	HTTPSProxyEnvValue     string
	NoProxyEnvValue        string
	AutoCreateClusterToken string
	IsOpenShift            bool
	IsGKEAutopilot         bool
	IncludeMinio           bool
	IsMinimalRBAC          bool
	MigrateToMinioXl       bool
	CurrentMinioImage      string
	AdditionalNamespaces   []string

	RegistryConfig kotsadmtypes.RegistryConfig
}

func GenerateAdminConsoleFiles(renderDir string, options types.WriteOptions) ([]types.UpstreamFile, error) {
	if options.Namespace == "" {
		options.Namespace = "default"
	}

	if _, err := os.Stat(path.Join(renderDir, "admin-console")); os.IsNotExist(err) {
		settings := &UpstreamSettings{
			Namespace:              options.Namespace,
			SharedPassword:         options.SharedPassword,
			AutoCreateClusterToken: uuid.New().String(),
			HTTPProxyEnvValue:      options.HTTPProxyEnvValue,
			HTTPSProxyEnvValue:     options.HTTPSProxyEnvValue,
			NoProxyEnvValue:        options.NoProxyEnvValue,
			IsOpenShift:            options.IsOpenShift,
			IsGKEAutopilot:         options.IsGKEAutopilot,
			IncludeMinio:           options.IncludeMinio,
			MigrateToMinioXl:       options.MigrateToMinioXl,
			CurrentMinioImage:      options.CurrentMinioImage,
			IsMinimalRBAC:          options.IsMinimalRBAC,
			AdditionalNamespaces:   options.AdditionalNamespaces,
			RegistryConfig:         options.RegistryConfig,
		}
		return generateNewAdminConsoleFiles(settings)
	}

	entries, err := os.ReadDir(path.Join(renderDir, "admin-console"))
	if err != nil {
		return nil, errors.Wrap(err, "failed to read existing files")
	}
	existingFiles := make([]fs.FileInfo, 0, len(entries))
	for _, entry := range entries {
		info, err := entry.Info()
		if err != nil {
			return nil, errors.Wrap(err, "failed to read existing files")
		}
		existingFiles = append(existingFiles, info)
	}

	settings := &UpstreamSettings{
		Namespace:              options.Namespace,
		AutoCreateClusterToken: uuid.New().String(),
		IsOpenShift:            options.IsOpenShift,
		IsGKEAutopilot:         options.IsGKEAutopilot,
		IncludeMinio:           options.IncludeMinio,
		MigrateToMinioXl:       options.MigrateToMinioXl,
		CurrentMinioImage:      options.CurrentMinioImage,
		IsMinimalRBAC:          options.IsMinimalRBAC,
		AdditionalNamespaces:   options.AdditionalNamespaces,
		RegistryConfig:         options.RegistryConfig,
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
		data, err := os.ReadFile(path.Join(renderDir, "admin-console", fi.Name()))
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
		} else if gvk.Group == "apps" && gvk.Version == "v1" && gvk.Kind == "StatefulSet" {
			loadUpstreamSettingsFromStatefulSet(settings, obj.(*appsv1.StatefulSet))
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
	case "kotsadm-rqlite":
		settings.RqlitePassword = string(secret.Data["password"])
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

func loadUpstreamSettingsFromStatefulSet(settings *UpstreamSettings, statefulset *appsv1.StatefulSet) {
	for _, c := range statefulset.Spec.Template.Spec.Containers {
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
		Namespace:              settings.Namespace,
		SharedPassword:         settings.SharedPassword,
		SharedPasswordBcrypt:   settings.SharedPasswordBcrypt,
		S3AccessKey:            settings.S3AccessKey,
		S3SecretKey:            settings.S3SecretKey,
		JWT:                    settings.JWT,
		RqlitePassword:         settings.RqlitePassword,
		APIEncryptionKey:       settings.APIEncryptionKey,
		AutoCreateClusterToken: settings.AutoCreateClusterToken,
		HTTPProxyEnvValue:      settings.HTTPProxyEnvValue,
		HTTPSProxyEnvValue:     settings.HTTPSProxyEnvValue,
		NoProxyEnvValue:        settings.NoProxyEnvValue,
		IsOpenShift:            settings.IsOpenShift,
		IsGKEAutopilot:         settings.IsGKEAutopilot,
		IncludeMinio:           settings.IncludeMinio,
		MigrateToMinioXl:       settings.MigrateToMinioXl,
		CurrentMinioImage:      settings.CurrentMinioImage,
		EnsureRBAC:             true,
		IsMinimalRBAC:          settings.IsMinimalRBAC,
		AdditionalNamespaces:   settings.AdditionalNamespaces,
		RegistryConfig:         settings.RegistryConfig,
	}

	if deployOptions.SharedPasswordBcrypt == "" && deployOptions.SharedPassword == "" {
		p, err := util.PromptForNewPassword()
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
