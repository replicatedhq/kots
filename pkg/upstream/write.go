package upstream

import (
	"bytes"
	"io/ioutil"
	"os"
	"path"

	"github.com/pkg/errors"
	kotsv1beta1 "github.com/replicatedhq/kots/kotskinds/apis/kots/v1beta1"
	"github.com/replicatedhq/kots/pkg/crypto"
	"github.com/replicatedhq/kots/pkg/upstream/types"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/serializer/json"
	"k8s.io/client-go/kubernetes/scheme"
)

func WriteUpstream(u *types.Upstream, options types.WriteOptions) error {
	renderDir := options.RootDir
	if options.CreateAppDir {
		renderDir = path.Join(renderDir, u.Name)
	}

	renderDir = path.Join(renderDir, "upstream")

	if options.IncludeAdminConsole {
		adminConsoleFiles, err := generateAdminConsoleFiles(renderDir, options.SharedPassword)
		if err != nil {
			return errors.Wrap(err, "failed to generate admin console")
		}

		u.Files = append(u.Files, adminConsoleFiles...)
	}

	var previousValuesContent []byte
	var previousInstallationContent []byte
	_, err := os.Stat(renderDir)
	if err == nil {
		// if there's already a config values yaml, we need to save
		_, err := os.Stat(path.Join(renderDir, "userdata", "config.yaml"))
		if err == nil {
			c, err := ioutil.ReadFile(path.Join(renderDir, "userdata", "config.yaml"))
			if err != nil {
				return errors.Wrap(err, "failed to read existing config values")
			}

			previousValuesContent = c
		}

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

	for _, file := range u.Files {
		fileRenderPath := path.Join(renderDir, file.Path)
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

	if previousValuesContent != nil {
		for i, f := range u.Files {
			if f.Path == path.Join("userdata", "config.yaml") {
				mergedValues, err := mergeValues(previousValuesContent, f.Content)
				if err != nil {
					return errors.Wrap(err, "failed to merge config values")
				}

				err = ioutil.WriteFile(path.Join(renderDir, "userdata", "config.yaml"), mergedValues, 0644)
				if err != nil {
					return errors.Wrap(err, "failed to replace configg values with previous config values")
				}

				updatedValues := types.UpstreamFile{
					Path:    f.Path,
					Content: mergedValues,
				}

				u.Files[i] = updatedValues
			}
		}
	}

	// Write the installation status (update cursor, etc)
	// but preserving the encryption key, if there already is one
	encryptionKey, err := getEncryptionKey(previousInstallationContent)
	if err != nil {
		return errors.Wrap(err, "failed to get encryption key")
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
			VersionLabel:  u.VersionLabel,
			ReleaseNotes:  u.ReleaseNotes,
			EncryptionKey: encryptionKey,
		},
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

func getEncryptionKey(previousInstallationContent []byte) (string, error) {
	if previousInstallationContent == nil {
		cipher, err := crypto.NewAESCipher()
		if err != nil {
			return "", errors.Wrap(err, "failed to create new AES cipher")
		}

		return cipher.ToString(), nil
	}

	decode := scheme.Codecs.UniversalDeserializer().Decode

	prevObj, _, err := decode(previousInstallationContent, nil, nil)
	if err != nil {
		return "", errors.Wrap(err, "failed to decode previous installation")
	}
	installation := prevObj.(*kotsv1beta1.Installation)

	return installation.Spec.EncryptionKey, nil
}

func mergeValues(previousValues []byte, applicationDeliveredValues []byte) ([]byte, error) {
	decode := scheme.Codecs.UniversalDeserializer().Decode

	prevObj, _, err := decode(previousValues, nil, nil)
	if err != nil {
		return nil, errors.Wrap(err, "failed to decode previous values")
	}
	prevValues := prevObj.(*kotsv1beta1.ConfigValues)

	applicationValuesObj, _, err := decode(applicationDeliveredValues, nil, nil)
	if err != nil {
		return nil, errors.Wrap(err, "failed to decode application delivered values")
	}
	applicationValues := applicationValuesObj.(*kotsv1beta1.ConfigValues)

	for name, value := range applicationValues.Spec.Values {
		_, ok := prevValues.Spec.Values[name]
		if !ok {
			prevValues.Spec.Values[name] = value
		}
	}

	s := json.NewYAMLSerializer(json.DefaultMetaFactory, scheme.Scheme, scheme.Scheme)

	var b bytes.Buffer
	if err := s.Encode(prevValues, &b); err != nil {
		return nil, errors.Wrap(err, "failed to encode merged values")
	}

	return b.Bytes(), nil
}

func mustMarshalInstallation(installation *kotsv1beta1.Installation) []byte {
	s := json.NewYAMLSerializer(json.DefaultMetaFactory, scheme.Scheme, scheme.Scheme)

	var b bytes.Buffer
	if err := s.Encode(installation, &b); err != nil {
		panic(err)
	}

	return b.Bytes()
}
