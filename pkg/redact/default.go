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
	defaultRedactSpecConfigMapName = "kotsadm-redact-default-spec"
	defaultRedactSpecDataKey       = "default-redactor"
	ipv4AddressRegex               = "(?P<mask>\\b(?P<drop>25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)\\.(?P<drop>25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)\\.(?P<drop>25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)\\.(?P<drop>25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)\\b)"
)

func GetDefaultRedactSpecURI() string {
	return fmt.Sprintf("configmap/%s/%s/%s", util.PodNamespace, defaultRedactSpecConfigMapName, defaultRedactSpecDataKey)
}

// CreateRenderedDefaultRedactSpec creates a configmap that contains the default redaction yaml spec for the admin console
func CreateRenderedDefaultRedactSpec(clientset kubernetes.Interface) error {
	redactor := getDefaultRedactor()

	s := serializer.NewYAMLSerializer(serializer.DefaultMetaFactory, scheme.Scheme, scheme.Scheme)
	var b bytes.Buffer
	if err := s.Encode(redactor, &b); err != nil {
		return errors.Wrap(err, "failed to serialize default redactor")
	}
	spec := b.String()

	existingConfigMap, err := clientset.CoreV1().ConfigMaps(util.PodNamespace).Get(context.TODO(), defaultRedactSpecConfigMapName, metav1.GetOptions{})
	if err != nil && !kuberneteserrors.IsNotFound(err) {
		return errors.Wrap(err, "failed to read default redactor configmap")
	} else if kuberneteserrors.IsNotFound(err) {
		configMap := &corev1.ConfigMap{
			TypeMeta: metav1.TypeMeta{
				APIVersion: "v1",
				Kind:       "ConfigMap",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      defaultRedactSpecConfigMapName,
				Namespace: util.PodNamespace,
				Labels:    kotsadmtypes.GetKotsadmLabels(),
			},
			Data: map[string]string{
				defaultRedactSpecDataKey: spec,
			},
		}

		_, err = clientset.CoreV1().ConfigMaps(util.PodNamespace).Create(context.TODO(), configMap, metav1.CreateOptions{})
		if err != nil {
			return errors.Wrap(err, "failed to create default redactor configmap")
		}

		return nil
	}

	if existingConfigMap.Data == nil {
		existingConfigMap.Data = map[string]string{}
	}
	existingConfigMap.Data[defaultRedactSpecDataKey] = spec
	existingConfigMap.ObjectMeta.Labels = kotsadmtypes.GetKotsadmLabels()

	_, err = clientset.CoreV1().ConfigMaps(util.PodNamespace).Update(context.TODO(), existingConfigMap, metav1.UpdateOptions{})
	if err != nil {
		return errors.Wrap(err, "failed to update default redactor configmap")
	}

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
