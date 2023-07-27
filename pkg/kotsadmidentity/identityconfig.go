package identity

import (
	"bytes"
	"os"

	"github.com/pkg/errors"
	kotsv1beta1 "github.com/replicatedhq/kotskinds/apis/kots/v1beta1"
	"github.com/segmentio/ksuid"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	serializer "k8s.io/apimachinery/pkg/runtime/serializer/json"
	"k8s.io/client-go/kubernetes/scheme"
)

func InitAppIdentityConfig(appSlug string) (string, error) {
	identityConfig := &kotsv1beta1.IdentityConfig{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "kots.io/v1beta1",
			Kind:       "IdentityConfig",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: "identity",
		},
		Spec: kotsv1beta1.IdentityConfigSpec{
			ClientID:     appSlug,
			ClientSecret: &kotsv1beta1.StringValueOrEncrypted{Value: ksuid.New().String()},
		},
	}

	s := serializer.NewYAMLSerializer(serializer.DefaultMetaFactory, scheme.Scheme, scheme.Scheme)

	buf := bytes.NewBuffer(nil)
	if err := s.Encode(identityConfig, buf); err != nil {
		return "", errors.Wrap(err, "failed to encode config")
	}

	identityConfigTmpFile, err := os.CreateTemp("", "kots")
	if err != nil {
		return "", errors.Wrap(err, "failed to create temp file")
	}
	_ = identityConfigTmpFile.Close()

	if err := os.WriteFile(identityConfigTmpFile.Name(), buf.Bytes(), 0644); err != nil {
		os.Remove(identityConfigTmpFile.Name())
		return "", errors.Wrap(err, "failed to write config to temp file")
	}

	return identityConfigTmpFile.Name(), nil
}
