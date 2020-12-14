package identity

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"

	"github.com/pkg/errors"
	kotsv1beta1 "github.com/replicatedhq/kots/kotskinds/apis/kots/v1beta1"
	"github.com/segmentio/ksuid"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	serializer "k8s.io/apimachinery/pkg/runtime/serializer/json"
	"k8s.io/client-go/kubernetes/scheme"
)

func InitAppIdentityConfig(appSlug string) (string, error) {
	// support for the dev environment where app is in "test" namespace
	host := "kotsadm-postgres"
	if kotsadmNamespace := os.Getenv("POD_NAMESPACE"); kotsadmNamespace != "" {
		host = fmt.Sprintf("%s.%s", host, kotsadmNamespace)
	}

	database := fmt.Sprintf("%s-dex", appSlug)
	user := fmt.Sprintf("%s-dex", appSlug)
	postgresPassword := ksuid.New().String()
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
			Storage: &kotsv1beta1.Storage{
				PostgresConfig: kotsv1beta1.IdentityPostgresConfig{
					Host:     host,
					Database: fmt.Sprintf("%s-dex", appSlug),
					User:     fmt.Sprintf("%s-dex", appSlug),
					Password: postgresPassword,
				},
			},
			ClientID:     appSlug,
			ClientSecret: ksuid.New().String(),
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
	if err := ioutil.WriteFile(identityConfigTmpFile.Name(), buf.Bytes(), 0644); err != nil {
		os.Remove(identityConfigTmpFile.Name())
		return "", errors.Wrap(err, "failed to write config to temp file")
	}

	return identityConfigTmpFile.Name(), nil
}
