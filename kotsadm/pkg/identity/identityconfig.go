package identity

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"

	"github.com/pkg/errors"
	kotsv1beta1 "github.com/replicatedhq/kots/kotskinds/apis/kots/v1beta1"
	"github.com/replicatedhq/kots/pkg/crypto"
	"github.com/segmentio/ksuid"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	serializer "k8s.io/apimachinery/pkg/runtime/serializer/json"
	"k8s.io/client-go/kubernetes/scheme"
)

func AppIdentityNeedsBootstrap(appSlug string) (bool, error) {
	user := fmt.Sprintf("%s-dex", appSlug)
	exists, err := postgresUserExists(user)
	if err != nil {
		return false, errors.Wrap(err, "failed to create dex postgres database")
	}

	if exists {
		return false, nil
	}

	return true, nil
}

func InitAppIdentityConfig(appSlug string, storage kotsv1beta1.Storage, cipher crypto.AESCipher) (string, error) {
	// support for the dev environment where app is in "test" namespace
	host := "kotsadm-postgres"
	if kotsadmNamespace := os.Getenv("POD_NAMESPACE"); kotsadmNamespace != "" {
		host = fmt.Sprintf("%s.%s", host, kotsadmNamespace)
	}

	var postgresPassword string
	if storage.PostgresConfig != nil {
		var err error
		postgresPassword, err = storage.PostgresConfig.Password.GetValue(cipher)
		if err != nil {
			return "", errors.Wrap(err, "failed to get password value")
		}
	}
	if postgresPassword == "" {
		postgresPassword = ksuid.New().String()
	}

	database := fmt.Sprintf("%s-dex", appSlug)
	user := fmt.Sprintf("%s-dex", appSlug)
	err := createDexPostgresDatabase(database, user, postgresPassword)
	if err != nil {
		return "", errors.Wrap(err, "failed to create dex postgres database")
	}

	identityConfig := &kotsv1beta1.IdentityConfig{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "kots.io/v1beta1",
			Kind:       "IdentityConfig",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: "identity",
		},
		Spec: kotsv1beta1.IdentityConfigSpec{
			Storage: kotsv1beta1.Storage{
				PostgresConfig: &kotsv1beta1.IdentityPostgresConfig{
					Host:     host,
					Database: database,
					User:     user,
					Password: &kotsv1beta1.StringValueOrEncrypted{Value: postgresPassword},
				},
			},
			ClientID:     appSlug,
			ClientSecret: &kotsv1beta1.StringValueOrEncrypted{Value: ksuid.New().String()},
		},
	}

	s := serializer.NewYAMLSerializer(serializer.DefaultMetaFactory, scheme.Scheme, scheme.Scheme)

	buf := bytes.NewBuffer(nil)
	if err := s.Encode(identityConfig, buf); err != nil {
		return "", errors.Wrap(err, "failed to encode config")
	}

	identityConfigTmpFile, err := ioutil.TempFile("", "kots")
	if err != nil {
		return "", errors.Wrap(err, "failed to create temp file")
	}
	_ = identityConfigTmpFile.Close()

	if err := ioutil.WriteFile(identityConfigTmpFile.Name(), buf.Bytes(), 0644); err != nil {
		os.Remove(identityConfigTmpFile.Name())
		return "", errors.Wrap(err, "failed to write config to temp file")
	}

	return identityConfigTmpFile.Name(), nil
}
