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
