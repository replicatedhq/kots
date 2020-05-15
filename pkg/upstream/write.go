package upstream

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"

	"github.com/pkg/errors"
	kotsv1beta1 "github.com/replicatedhq/kots/kotskinds/apis/kots/v1beta1"
	"github.com/replicatedhq/kots/pkg/crypto"
	"github.com/replicatedhq/kots/pkg/upstream/types"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/serializer/json"
	serializer "k8s.io/apimachinery/pkg/runtime/serializer/json"
	"k8s.io/client-go/kubernetes/scheme"
)

func WriteUpstream(u *types.Upstream, options types.WriteOptions) error {
	renderDir := options.RootDir
	if options.CreateAppDir {
		renderDir = filepath.Join(renderDir, u.Name)
	}

	renderDir = filepath.Join(renderDir, "upstream")

	if options.IncludeAdminConsole {
		adminConsoleFiles, err := generateAdminConsoleFiles(renderDir, options.SharedPassword)
		if err != nil {
			return errors.Wrap(err, "failed to generate admin console")
		}

		u.Files = append(u.Files, adminConsoleFiles...)
	}

	var previousInstallationContent []byte
	_, err := os.Stat(renderDir)
	if err == nil {
		_, err = os.Stat(filepath.Join(renderDir, "userdata", "installation.yaml"))
		if err == nil {
			c, err := ioutil.ReadFile(filepath.Join(renderDir, "userdata", "installation.yaml"))
			if err != nil {
				return errors.Wrap(err, "failed to read existing installation")
			}

			previousInstallationContent = c
		}

		if err := os.RemoveAll(renderDir); err != nil {
			return errors.Wrap(err, "failed to remove previous content in upstream")
		}
	}

	var prevInstallation *kotsv1beta1.Installation
	if previousInstallationContent != nil {
		decode := scheme.Codecs.UniversalDeserializer().Decode

		prevObj, _, err := decode(previousInstallationContent, nil, nil)
		if err != nil {
			return errors.Wrap(err, "failed to decode previous installation")
		}
		prevInstallation = prevObj.(*kotsv1beta1.Installation)
	}

	for _, file := range u.Files {
		fileRenderPath := filepath.Join(renderDir, file.Path)
		d, _ := path.Split(fileRenderPath)
		if _, err := os.Stat(d); os.IsNotExist(err) {
			if err := os.MkdirAll(d, 0744); err != nil {
				return errors.Wrap(err, "failed to mkdir")
			}
		}

		if err := ioutil.WriteFile(fileRenderPath, file.Content, 0644); err != nil {
			return errors.Wrap(err, "failed to write upstream file")
		}
	}

	// Write the installation status (update cursor, etc)
	// but preserving the encryption key, if there already is one
	encryptionKey, err := getEncryptionKey(prevInstallation)
	if err != nil {
		return errors.Wrap(err, "failed to get encryption key")
	}

	var channelName string
	if prevInstallation != nil && options.PreserveInstallation {
		channelName = prevInstallation.Spec.ChannelName
	} else {
		channelName = u.ChannelName
	}

	installation := kotsv1beta1.Installation{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "kots.io/v1beta1",
			Kind:       "Installation",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: u.Name,
		},
		Spec: kotsv1beta1.InstallationSpec{
			UpdateCursor:  u.UpdateCursor,
			ChannelName:   channelName,
			VersionLabel:  u.VersionLabel,
			ReleaseNotes:  u.ReleaseNotes,
			EncryptionKey: encryptionKey,
		},
	}
	if _, err := os.Stat(filepath.Join(renderDir, "userdata")); os.IsNotExist(err) {
		if err := os.MkdirAll(filepath.Join(renderDir, "userdata"), 0755); err != nil {
			return errors.Wrap(err, "failed to create userdata dir")
		}
	}
	err = ioutil.WriteFile(filepath.Join(renderDir, "userdata", "installation.yaml"), mustMarshalInstallation(&installation), 0644)
	if err != nil {
		return errors.Wrap(err, "failed to write installation")
	}

	// finally, load the config, config values from what's on disk, and encrypt any plain text
	// we have to do this so late because the encryption key is not generated until very
	// late in the process
	config, configValues, _, err := findConfig(renderDir)
	if err != nil {
		return errors.Wrap(err, "failed to find config in dir")
	}

	updatedConfigValues, err := EncryptConfigValues(config, configValues, &installation)
	if err != nil {
		return errors.Wrap(err, "failed to find encrypt config values")
	}

	s := serializer.NewYAMLSerializer(serializer.DefaultMetaFactory, scheme.Scheme, scheme.Scheme)

	var b bytes.Buffer
	if err := s.Encode(updatedConfigValues, &b); err != nil {
		return errors.Wrap(err, "failed to encode config values")
	}

	if err := ioutil.WriteFile(filepath.Join(renderDir, "userdata", "config.yaml"), b.Bytes(), 0644); err != nil {
		return errors.Wrap(err, "failed to write config values")
	}

	return nil
}

func getEncryptionKey(prevInstallation *kotsv1beta1.Installation) (string, error) {
	if prevInstallation == nil {
		cipher, err := crypto.NewAESCipher()
		if err != nil {
			return "", errors.Wrap(err, "failed to create new AES cipher")
		}

		return cipher.ToString(), nil
	}

	return prevInstallation.Spec.EncryptionKey, nil
}

func mustMarshalInstallation(installation *kotsv1beta1.Installation) []byte {
	s := json.NewYAMLSerializer(json.DefaultMetaFactory, scheme.Scheme, scheme.Scheme)

	var b bytes.Buffer
	if err := s.Encode(installation, &b); err != nil {
		panic(err)
	}

	return b.Bytes()
}

func findConfig(localPath string) (*kotsv1beta1.Config, *kotsv1beta1.ConfigValues, *kotsv1beta1.Installation, error) {
	if localPath == "" {
		return nil, nil, nil, nil
	}

	var config *kotsv1beta1.Config
	var values *kotsv1beta1.ConfigValues
	var installation *kotsv1beta1.Installation

	err := filepath.Walk(localPath,
		func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}

			if info.IsDir() {
				return nil
			}

			content, err := ioutil.ReadFile(path)
			if err != nil {
				return err
			}

			decode := scheme.Codecs.UniversalDeserializer().Decode
			obj, gvk, err := decode(content, nil, nil)
			if err != nil {
				return nil
			}

			if gvk.Group == "kots.io" && gvk.Version == "v1beta1" && gvk.Kind == "Config" {
				config = obj.(*kotsv1beta1.Config)
			} else if gvk.Group == "kots.io" && gvk.Version == "v1beta1" && gvk.Kind == "ConfigValues" {
				values = obj.(*kotsv1beta1.ConfigValues)
			} else if gvk.Group == "kots.io" && gvk.Version == "v1beta1" && gvk.Kind == "Installation" {
				installation = obj.(*kotsv1beta1.Installation)
			}

			return nil
		})

	if err != nil {
		return nil, nil, nil, errors.Wrap(err, "failed to walk local dir")
	}

	return config, values, installation, nil
}
