package ingress

import (
	"github.com/replicatedhq/kots/pkg/ingress/types"
	kotsadmtypes "github.com/replicatedhq/kots/pkg/kotsadm/types"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

func NodePortFromConfig(nodePortConfig types.NodePortConfig, name string, selector map[string]string, servicePort int, targetPort intstr.IntOrString, additionalLabels map[string]string) *corev1.Service {
	return &corev1.Service{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "Service",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:   name,
			Labels: kotsadmtypes.GetKotsadmLabels(additionalLabels),
		},
		Spec: corev1.ServiceSpec{
			Type:     corev1.ServiceTypeNodePort,
			Selector: selector,
			Ports: []corev1.ServicePort{
				{
					Name:       "default",
					Port:       int32(servicePort),
					TargetPort: targetPort,
					NodePort:   int32(nodePortConfig.Port),
				},
			},
		},
	}
}
