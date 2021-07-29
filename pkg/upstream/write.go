package upstream

import (
	"bytes"
	"encoding/base64"
	"io/ioutil"
	"os"
	"path"

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
		renderDir = path.Join(renderDir, u.Name)
	}

	renderDir = path.Join(renderDir, "upstream")

	if options.IncludeAdminConsole {
		adminConsoleFiles, err := GenerateAdminConsoleFiles(renderDir, options)
		if err != nil {
			return errors.Wrap(err, "failed to generate admin console")
		}

		u.Files = append(u.Files, adminConsoleFiles...)
	}

	var previousInstallationContent []byte
	_, err := os.Stat(renderDir)
	if err == nil {
		_, err = os.Stat(path.Join(renderDir, "userdata", "installation.yaml"))
		if err == nil {
			c, err := ioutil.ReadFile(path.Join(renderDir, "userdata", "installation.yaml"))
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

	encryptionKey, err := getEncryptionKey(prevInstallation)
	if err != nil {
		return errors.Wrap(err, "failed to get encryption key")
	}
	u.EncryptionKey = encryptionKey

	for i, file := range u.Files {
		fileRenderPath := path.Join(renderDir, file.Path)
		d, _ := path.Split(fileRenderPath)
		if _, err := os.Stat(d); os.IsNotExist(err) {
			if err := os.MkdirAll(d, 0744); err != nil {
				return errors.Wrap(err, "failed to mkdir")
			}
		}

		if options.EncryptConfig {
			configValues := contentToConfigValues(file.Content)
			if configValues != nil {
				content, err := encryptConfigValues(configValues, encryptionKey)
				if err != nil {
					return errors.Wrap(err, "failed to encrypt config values")
				}
				file.Content = content
				u.Files[i] = file
			}
		}

		identityConfig := contentToIdentityConfig(file.Content)
		if identityConfig != nil {
			content, err := maybeEncryptIdentityConfig(identityConfig, encryptionKey)
			if err != nil {
				return errors.Wrap(err, "failed to encrypt identity config")
			}
			file.Content = content
			u.Files[i] = file
		}

		if err := ioutil.WriteFile(fileRenderPath, file.Content, 0644); err != nil {
			return errors.Wrap(err, "failed to write upstream file")
		}
	}

	channelID, channelName := "", ""
	if prevInstallation != nil && options.PreserveInstallation {
		channelID = prevInstallation.Spec.ChannelID
		channelName = prevInstallation.Spec.ChannelName
	} else {
		channelID = u.ChannelID
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
			ChannelID:     channelID,
			ChannelName:   channelName,
			VersionLabel:  u.VersionLabel,
			ReleaseNotes:  u.ReleaseNotes,
			EncryptionKey: encryptionKey,
		},
	}

	if u.ReleasedAt != nil {
		releasedAt := metav1.NewTime(*u.ReleasedAt)
		installation.Spec.ReleasedAt = &releasedAt
	}

	if _, err := os.Stat(path.Join(renderDir, "userdata")); os.IsNotExist(err) {
		if err := os.MkdirAll(path.Join(renderDir, "userdata"), 0755); err != nil {
			return errors.Wrap(err, "failed to create userdata dir")
		}
	}
	err = ioutil.WriteFile(path.Join(renderDir, "userdata", "installation.yaml"), mustMarshalInstallation(&installation), 0644)
	if err != nil {
		return errors.Wrap(err, "failed to write installation")
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

func encryptConfigValues(configValues *kotsv1beta1.ConfigValues, encryptionKey string) ([]byte, error) {
	cipher, err := crypto.AESCipherFromString(encryptionKey)
	if err != nil {
		return nil, errors.Wrap(err, "failed to load encryption cipher")
	}
	for k, v := range configValues.Spec.Values {
		if v.ValuePlaintext == "" {
			continue
		}

		v.Value = base64.StdEncoding.EncodeToString(cipher.Encrypt([]byte(v.ValuePlaintext)))
		v.ValuePlaintext = ""

		configValues.Spec.Values[k] = v
	}

	s := serializer.NewYAMLSerializer(serializer.DefaultMetaFactory, scheme.Scheme, scheme.Scheme)

	var b bytes.Buffer
	if err := s.Encode(configValues, &b); err != nil {
		return nil, errors.Wrap(err, "failed to encode config values")
	}

	return b.Bytes(), nil
}

func maybeEncryptIdentityConfig(identityConfig *kotsv1beta1.IdentityConfig, encryptionKey string) ([]byte, error) {
	cipher, err := crypto.AESCipherFromString(encryptionKey)
	if err != nil {
		return nil, errors.Wrap(err, "failed to load encryption cipher")
	}

	identityConfig.Spec.ClientSecret.EncryptValue(*cipher)

	if identityConfig.Spec.Storage.PostgresConfig != nil {
		identityConfig.Spec.Storage.PostgresConfig.Password.EncryptValue(*cipher)
	}

	identityConfig.Spec.DexConnectors.EncryptValue(*cipher)

	s := serializer.NewYAMLSerializer(serializer.DefaultMetaFactory, scheme.Scheme, scheme.Scheme)

	var b bytes.Buffer
	if err := s.Encode(identityConfig, &b); err != nil {
		return nil, errors.Wrap(err, "failed to encode identity config")
	}

	return b.Bytes(), nil
}
