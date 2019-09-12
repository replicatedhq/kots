package upstream

import (
	"io/ioutil"
	"os"
	"path"

	"github.com/manifoldco/promptui"
	"github.com/pkg/errors"
	"github.com/replicatedhq/kots/pkg/kotsadm"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes/scheme"
)

func generateAdminConsoleFiles(renderDir string, sharedPassword string) ([]UpstreamFile, error) {
	if _, err := os.Stat(path.Join(renderDir, "admin-console")); os.IsNotExist(err) {
		return generateNewAdminConsoleFiles(sharedPassword, "", "", "", "", "")
	}

	existingFiles, err := ioutil.ReadDir(path.Join(renderDir, "admin-console"))
	if err != nil {
		return nil, errors.Wrap(err, "failed to read existing files")
	}

	sharedPasswordBcrypt, err := findFileAndReadSecret("kotsadm-password", "passwordBcrypt", renderDir, existingFiles)
	if err != nil {
		return nil, errors.Wrap(err, "failed to find existing shared password")
	}
	s3AccessKey, err := findFileAndReadSecret("kotsadm-minio", "accesskey", renderDir, existingFiles)
	if err != nil {
		return nil, errors.Wrap(err, "failed to find existing s3 access key")
	}
	s3SecretKey, err := findFileAndReadSecret("kotsadm-minio", "secretkey", renderDir, existingFiles)
	if err != nil {
		return nil, errors.Wrap(err, "failed to find existing s3 secret key")
	}
	jwt, err := findFileAndReadSecret("kotsadm-session", "key", renderDir, existingFiles)
	if err != nil {
		return nil, errors.Wrap(err, "failed to find existing jwt")
	}
	pgPassword, err := findPostgresPassword(renderDir, existingFiles)
	if err != nil {
		return nil, errors.Wrap(err, "failed to find existing jwt")
	}

	return generateNewAdminConsoleFiles("", sharedPasswordBcrypt, s3AccessKey, s3SecretKey, jwt, pgPassword)
}

func findPostgresPassword(renderDir string, files []os.FileInfo) (string, error) {
	for _, fi := range files {
		data, err := ioutil.ReadFile(path.Join(renderDir, "admin-console", fi.Name()))
		if err != nil {
			return "", errors.Wrap(err, "failed to read file")
		}

		decode := scheme.Codecs.UniversalDeserializer().Decode
		obj, gvk, err := decode(data, nil, nil)
		if err != nil {
			return "", errors.Wrap(err, "failed to decode")
		}

		if gvk.Group != "apps" || gvk.Version != "v1" || gvk.Kind != "StatefulSet" {
			continue
		}

		statefulset := obj.(*appsv1.StatefulSet)

		if statefulset.Name == "kotsadm-postgres" {
			env := statefulset.Spec.Template.Spec.Containers[0].Env
			for _, ev := range env {
				if ev.Name == "POSTGRES_PASSWORD" {
					return ev.Value, nil
				}
			}
		}
	}

	return "", nil
}

func findFileAndReadSecret(secretName string, key string, renderDir string, files []os.FileInfo) (string, error) {
	for _, fi := range files {
		data, err := ioutil.ReadFile(path.Join(renderDir, "admin-console", fi.Name()))
		if err != nil {
			return "", errors.Wrap(err, "failed to read file")
		}

		decode := scheme.Codecs.UniversalDeserializer().Decode
		obj, gvk, err := decode(data, nil, nil)
		if err != nil {
			return "", errors.Wrap(err, "failed to decode")
		}
		if gvk.Group != "" || gvk.Version != "v1" || gvk.Kind != "Secret" {
			continue
		}

		secret := obj.(*corev1.Secret)

		if secret.Name == secretName {
			return string(secret.Data[key]), nil
		}
	}

	return "", nil
}

func generateNewAdminConsoleFiles(sharedPassword string, sharedPasswordBcrypt string, s3AccessKey string, s3SecretKey string, jwt string, pgPassword string) ([]UpstreamFile, error) {
	upstreamFiles := []UpstreamFile{}

	deployOptions := kotsadm.DeployOptions{
		Namespace:            "default",
		SharedPassword:       sharedPassword,
		SharedPasswordBcrypt: sharedPasswordBcrypt,
		S3AccessKey:          s3AccessKey,
		S3SecretKey:          s3SecretKey,
		JWT:                  jwt,
		PostgresPassword:     pgPassword,
		Hostname:             "localhost:8800",
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
		upstreamFile := UpstreamFile{
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
