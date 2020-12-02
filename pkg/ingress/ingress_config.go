package ingress

import (
	kotsv1beta1 "github.com/replicatedhq/kots/kotskinds/apis/kots/v1beta1"
	kotsadmtypes "github.com/replicatedhq/kots/pkg/kotsadm/types"
	extensions "k8s.io/api/extensions/v1beta1"
	extensionsv1beta1 "k8s.io/api/extensions/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

func IngressFromConfig(ingressConfig kotsv1beta1.IngressResourceConfig, name string, serviceName string, servicePort int, additionalLabels map[string]string) *extensionsv1beta1.Ingress {
	ingressTLS := []extensions.IngressTLS{}
	if ingressConfig.TLSSecretName != "" {
		tls := extensions.IngressTLS{
			SecretName: ingressConfig.TLSSecretName,
		}
		if ingressConfig.Host != "" {
			tls.Hosts = append(tls.Hosts, ingressConfig.Host)
		}
		ingressTLS = append(ingressTLS, tls)
	}

	annotations := map[string]string{
		"nginx.ingress.kubernetes.io/proxy-body-size": "100m",
	}
	for k, v := range ingressConfig.Annotations {
		annotations[k] = v
	}

	return &extensionsv1beta1.Ingress{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1beta1",
			Kind:       "Ingress",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:        name,
			Labels:      kotsadmtypes.GetKotsadmLabels(additionalLabels),
			Annotations: annotations,
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
			TLS: ingressTLS,
		},
	}
}
