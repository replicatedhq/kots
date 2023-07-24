package ingress

import (
	kotsadmtypes "github.com/replicatedhq/kots/pkg/kotsadm/types"
	kotsv1beta1 "github.com/replicatedhq/kotskinds/apis/kots/v1beta1"
	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func IngressFromConfig(namespace string, ingressConfig kotsv1beta1.IngressResourceConfig, name string, serviceName string, servicePort int32, additionalLabels map[string]string) *networkingv1.Ingress {
	ingressTLS := []networkingv1.IngressTLS{}
	if ingressConfig.TLSSecretName != "" {
		tls := networkingv1.IngressTLS{
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

	pathType := networkingv1.PathTypeImplementationSpecific

	return &networkingv1.Ingress{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "networking.k8s.io/v1",
			Kind:       "Ingress",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:        name,
			Namespace:   namespace,
			Labels:      kotsadmtypes.GetKotsadmLabels(additionalLabels),
			Annotations: annotations,
		},
		Spec: networkingv1.IngressSpec{
			Rules: []networkingv1.IngressRule{
				{
					Host: ingressConfig.Host,
					IngressRuleValue: networkingv1.IngressRuleValue{
						HTTP: &networkingv1.HTTPIngressRuleValue{
							Paths: []networkingv1.HTTPIngressPath{
								{
									Path:     ingressConfig.Path,
									PathType: &pathType,
									Backend: networkingv1.IngressBackend{
										Service: &networkingv1.IngressServiceBackend{
											Name: serviceName,
											Port: networkingv1.ServiceBackendPort{
												Number: servicePort,
											},
										},
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
