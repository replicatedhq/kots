package ingress

import (
	"github.com/replicatedhq/kots/pkg/ingress/types"
	kotsadmtypes "github.com/replicatedhq/kots/pkg/kotsadm/types"
	extensionsv1beta1 "k8s.io/api/extensions/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

func IngressFromConfig(ingressConfig types.IngressConfig, name string, serviceName string, servicePort int, additionalLabels map[string]string) *extensionsv1beta1.Ingress {
	return &extensionsv1beta1.Ingress{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1beta1",
			Kind:       "Ingress",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:        name,
			Labels:      kotsadmtypes.GetKotsadmLabels(additionalLabels),
			Annotations: ingressConfig.Annotations,
		},
		Spec: extensionsv1beta1.IngressSpec{
			Rules: []extensionsv1beta1.IngressRule{
				{
					Host: ingressConfig.Host,
					IngressRuleValue: extensionsv1beta1.IngressRuleValue{
						HTTP: &extensionsv1beta1.HTTPIngressRuleValue{
							Paths: []extensionsv1beta1.HTTPIngressPath{
								{
									Path: ingressConfig.Path,
									Backend: extensionsv1beta1.IngressBackend{
										ServiceName: serviceName,
										ServicePort: intstr.FromInt(servicePort),
									},
								},
							},
						},
					},
				},
			},
			TLS: ingressConfig.TLS,
		},
	}
}
