package redact

import (
	"bytes"
	"context"
	"fmt"

	"github.com/pkg/errors"
	kotsadmtypes "github.com/replicatedhq/kots/pkg/kotsadm/types"
	"github.com/replicatedhq/kots/pkg/util"
	troubleshootv1beta2 "github.com/replicatedhq/troubleshoot/pkg/apis/troubleshoot/v1beta2"
	corev1 "k8s.io/api/core/v1"
	kuberneteserrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	serializer "k8s.io/apimachinery/pkg/runtime/serializer/json"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
)

const (
	defaultRedactSpecName    = "kotsadm-redact-default-spec"
	defaultRedactSpecDataKey = "default-redactor"
	ipv4AddressRegex         = "(?P<mask>\\b(?P<drop>25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)\\.(?P<drop>25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)\\.(?P<drop>25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)\\.(?P<drop>25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)\\b)"
)

func GetDefaultRedactSpecURI() string {
	return fmt.Sprintf("secret/%s/%s/%s", util.PodNamespace, defaultRedactSpecName, defaultRedactSpecDataKey)
}

// CreateRenderedDefaultRedactSpec creates a secret that contains the default redaction yaml spec for the admin console
func CreateRenderedDefaultRedactSpec(clientset kubernetes.Interface) error {
	redactor := getDefaultRedactor()

	s := serializer.NewYAMLSerializer(serializer.DefaultMetaFactory, scheme.Scheme, scheme.Scheme)
	var b bytes.Buffer
	if err := s.Encode(redactor, &b); err != nil {
		return errors.Wrap(err, "failed to serialize default redactor")
	}
	spec := b.Bytes()

	existingSecret, err := clientset.CoreV1().Secrets(util.PodNamespace).Get(context.TODO(), defaultRedactSpecName, metav1.GetOptions{})
	if err != nil && !kuberneteserrors.IsNotFound(err) {
		return errors.Wrap(err, "failed to read default redactor secret")
	} else if kuberneteserrors.IsNotFound(err) {
		secret := &corev1.Secret{
			TypeMeta: metav1.TypeMeta{
				APIVersion: "v1",
				Kind:       "Secret",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      defaultRedactSpecName,
				Namespace: util.PodNamespace,
				Labels:    kotsadmtypes.GetKotsadmLabels(),
			},
			Data: map[string][]byte{
				defaultRedactSpecDataKey: spec,
			},
		}

		_, err = clientset.CoreV1().Secrets(util.PodNamespace).Create(context.TODO(), secret, metav1.CreateOptions{})
		if err != nil {
			return errors.Wrap(err, "failed to create default redactor secret")
		}

		deleteLegacyRedactSpecConfigMap(clientset, defaultRedactSpecName)
		return nil
	}

	if existingSecret.Data == nil {
		existingSecret.Data = map[string][]byte{}
	}
	existingSecret.Data[defaultRedactSpecDataKey] = spec
	existingSecret.ObjectMeta.Labels = kotsadmtypes.GetKotsadmLabels()

	_, err = clientset.CoreV1().Secrets(util.PodNamespace).Update(context.TODO(), existingSecret, metav1.UpdateOptions{})
	if err != nil {
		return errors.Wrap(err, "failed to update default redactor secret")
	}

	deleteLegacyRedactSpecConfigMap(clientset, defaultRedactSpecName)
	return nil
}

func getDefaultRedactor() *troubleshootv1beta2.Redactor {
	return &troubleshootv1beta2.Redactor{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Redactor",
			APIVersion: "troubleshoot.sh/v1beta2",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: "default-redactor",
		},
		Spec: troubleshootv1beta2.RedactorSpec{
			Redactors: []*troubleshootv1beta2.Redact{
				{
					Name: "IP Addresses",
					Removals: troubleshootv1beta2.Removals{
						Regex: []troubleshootv1beta2.Regex{
							{
								Redactor: ipv4AddressRegex,
							},
						},
					},
				},
			},
		},
	}
}
