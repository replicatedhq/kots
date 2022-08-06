package helm

import (
	"bytes"
	"context"
	"fmt"

	"github.com/pkg/errors"
	kotsv1beta1 "github.com/replicatedhq/kots/kotskinds/apis/kots/v1beta1"
	kotsscheme "github.com/replicatedhq/kots/kotskinds/client/kotsclientset/scheme"
	apptypes "github.com/replicatedhq/kots/pkg/app/types"
	"github.com/replicatedhq/kots/pkg/k8sutil"
	corev1 "k8s.io/api/core/v1"
	kuberneteserrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	serializer "k8s.io/apimachinery/pkg/runtime/serializer/json"
	"k8s.io/client-go/kubernetes/scheme"
)

func init() {
	kotsscheme.AddToScheme(scheme.Scheme)
}

func GetTempConfigValues(helmApp *apptypes.HelmApp) (*kotsv1beta1.ConfigValues, error) {
	clientSet, err := k8sutil.GetClientset()
	if err != nil {
		return nil, errors.Wrap(err, "failed to get clientset")
	}

	configValues := &kotsv1beta1.ConfigValues{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "kots.io/v1beta1",
			Kind:       "ConfigValues",
		},
		Spec: kotsv1beta1.ConfigValuesSpec{
			Values: map[string]kotsv1beta1.ConfigValue{},
		},
	}

	// Note that this must be chart name, not release name
	secretName := fmt.Sprintf("kots-%s-temp-values", helmApp.Release.Chart.Name())
	secret, err := clientSet.CoreV1().Secrets(helmApp.Namespace).Get(context.TODO(), secretName, metav1.GetOptions{})
	if err != nil {
		return configValues, errors.Wrap(err, "failed to get secret")
	}

	encodedValues, ok := secret.Data["values"]
	if !ok {
		return configValues, nil
	}

	decode := scheme.Codecs.UniversalDeserializer().Decode
	decoded, gvk, err := decode(encodedValues, nil, nil)
	if err != nil {
		return configValues, errors.Wrap(err, "failed to decode config values")
	}

	if gvk.Group != "kots.io" || gvk.Version != "v1beta1" || gvk.Kind != "ConfigValues" {
		return configValues, errors.Errorf("%q is not a valid ConfigValues GVK", gvk.String())
	}

	return decoded.(*kotsv1beta1.ConfigValues), nil
}

// TODO: this function is not thread safe
func SetTempConfigValues(helmApp *apptypes.HelmApp, revision int64, configValues *kotsv1beta1.ConfigValues) error {
	s := serializer.NewYAMLSerializer(serializer.DefaultMetaFactory, scheme.Scheme, scheme.Scheme)
	var b bytes.Buffer
	if err := s.Encode(configValues, &b); err != nil {
		return errors.Wrap(err, "failed to encode config values")
	}

	clientSet, err := k8sutil.GetClientset()
	if err != nil {
		return errors.Wrap(err, "failed to get clientset")
	}

	// Note that this must be chart name, not release name
	secretName := fmt.Sprintf("kots-%s-temp-values", helmApp.Release.Chart.Name())
	secret, err := clientSet.CoreV1().Secrets(helmApp.Namespace).Get(context.TODO(), secretName, metav1.GetOptions{})
	if err != nil && !kuberneteserrors.IsNotFound(err) {
		return errors.Wrap(err, "failed to get secret")
	}

	if kuberneteserrors.IsNotFound(err) {
		secret = &corev1.Secret{
			TypeMeta: metav1.TypeMeta{
				APIVersion: "v1",
				Kind:       "Secret",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      secretName,
				Namespace: helmApp.Namespace,
				// Annotations: secretAnnotations,
				Labels: map[string]string{
					"releaseName":  helmApp.Release.Name,
					"baseRevision": fmt.Sprintf("%d", revision),
				},
			},
			Type: corev1.SecretTypeOpaque,
			Data: map[string][]byte{
				"values": b.Bytes(),
			},
		}

		_, err = clientSet.CoreV1().Secrets(helmApp.Namespace).Create(context.TODO(), secret, metav1.CreateOptions{})
		if err != nil {
			return errors.Wrap(err, "failed to create config values secret")
		}

		return nil
	}

	secret.ObjectMeta.Labels["baseRevision"] = fmt.Sprintf("%d", revision)
	secret.Data["values"] = b.Bytes()

	_, err = clientSet.CoreV1().Secrets(helmApp.Namespace).Update(context.TODO(), secret, metav1.UpdateOptions{})
	if err != nil {
		return errors.Wrap(err, "failed to update dockerhub secret")
	}

	return nil
}

func deleteTempConfigValues(helmApp *apptypes.HelmApp) error {
	clientSet, err := k8sutil.GetClientset()
	if err != nil {
		return errors.Wrap(err, "failed to get clientset")
	}

	// Note that this must be chart name, not release name
	secretName := fmt.Sprintf("kots-%s-temp-values", helmApp.Release.Chart.Name())
	err = clientSet.CoreV1().Secrets(helmApp.Namespace).Delete(context.TODO(), secretName, metav1.DeleteOptions{})
	if err != nil && !kuberneteserrors.IsNotFound(err) {
		return errors.Wrap(err, "failed to delete secret")
	}

	return nil
}
