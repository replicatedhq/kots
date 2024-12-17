package appstate

import (
	"reflect"
	"testing"

	"github.com/replicatedhq/kots/pkg/appstate/types"
	v1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/version"
	discoveryfake "k8s.io/client-go/discovery/fake"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/fake"
)

func mockClientsetK8sVersion(expectedMajor string, expectedMinor string) kubernetes.Interface {
	clientset := fake.NewSimpleClientset(
		// Defaul backend service and endpoint
		&v1.Service{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "default-http-backend",
				Namespace: metav1.NamespaceSystem,
			},
			Spec: v1.ServiceSpec{
				Ports: []v1.ServicePort{
					{
						Name: "http",
						Port: 80,
					},
				},
			},
		},
		&v1.Endpoints{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "default-http-backend",
				Namespace: metav1.NamespaceSystem,
			},
			Subsets: []v1.EndpointSubset{
				{
					Ports: []v1.EndpointPort{
						{
							Name: "http",
							Port: 80,
						},
					},
					Addresses: []v1.EndpointAddress{
						{
							IP: "192.0.0.2",
						},
					},
				},
			},
		},

		// LoadBalancer service and endpoint
		&v1.Service{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "app-lb",
				Namespace: "",
			},
			Spec: v1.ServiceSpec{
				Type: v1.ServiceTypeLoadBalancer,
				Ports: []v1.ServicePort{
					{
						Name: "http",
						Port: 8080,
					},
				},
			},
		},
		&v1.Endpoints{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "app-lb",
				Namespace: "",
			},
			Subsets: []v1.EndpointSubset{
				{
					Ports: []v1.EndpointPort{
						{
							Name: "http",
							Port: 8080,
						},
					},
					Addresses: []v1.EndpointAddress{
						{
							IP: "172.0.0.2",
						},
					},
				},
			},
		},

		// NodePort service and endpoint
		&v1.Service{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "app-nodeport",
				Namespace: "",
			},
			Spec: v1.ServiceSpec{
				Type: v1.ServiceTypeNodePort,
				Ports: []v1.ServicePort{
					{
						Name: "http",
						Port: 8080,
					},
				},
			},
		},
		&v1.Endpoints{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "app-nodeport",
				Namespace: "",
			},
			Subsets: []v1.EndpointSubset{
				{
					Ports: []v1.EndpointPort{
						{
							Name: "http",
							Port: 8080,
						},
					},
					Addresses: []v1.EndpointAddress{
						{
							IP: "172.0.0.2",
						},
					},
				},
			},
		},
	)
	clientset.Discovery().(*discoveryfake.FakeDiscovery).FakedServerVersion = &version.Info{
		Major: expectedMajor,
		Minor: expectedMinor,
	}
	return clientset
}

func TestCalculateIngressState(t *testing.T) {
	type args struct {
		clientset kubernetes.Interface
		r         *networkingv1.Ingress
	}
	tests := []struct {
		name string
		args args
		want types.State
	}{
		{
			name: "expect ready state when ingress with k8s version < 1.22 and no default backend",
			args: args{
				clientset: mockClientsetK8sVersion("1", "21"),
				r: &networkingv1.Ingress{
					Spec: networkingv1.IngressSpec{},
					Status: networkingv1.IngressStatus{
						LoadBalancer: networkingv1.IngressLoadBalancerStatus{
							Ingress: []networkingv1.IngressLoadBalancerIngress{
								{
									IP: "192.0.0.1",
								},
							},
						},
					},
				},
			},
			want: types.StateReady,
		}, {
			name: "expect ready state when there is a load balancer and an IP address",
			args: args{
				clientset: mockClientsetK8sVersion("1", "23"),
				r: &networkingv1.Ingress{
					Spec: networkingv1.IngressSpec{
						Rules: []networkingv1.IngressRule{
							{
								IngressRuleValue: networkingv1.IngressRuleValue{
									HTTP: &networkingv1.HTTPIngressRuleValue{
										Paths: []networkingv1.HTTPIngressPath{
											{
												Backend: networkingv1.IngressBackend{
													Service: &networkingv1.IngressServiceBackend{
														Name: "app-lb",
													},
												},
											},
										},
									},
								},
							},
						},
					},
					Status: networkingv1.IngressStatus{
						LoadBalancer: networkingv1.IngressLoadBalancerStatus{
							Ingress: []networkingv1.IngressLoadBalancerIngress{
								{
									IP: "192.0.0.1",
								},
							},
						},
					},
				},
			},
			want: types.StateReady,
		}, {
			name: "expect ready state when there is no LoadBalancer and no address is assigned",
			args: args{
				clientset: mockClientsetK8sVersion("1", "23"),
				r: &networkingv1.Ingress{
					Spec: networkingv1.IngressSpec{
						Rules: []networkingv1.IngressRule{
							{
								IngressRuleValue: networkingv1.IngressRuleValue{
									HTTP: &networkingv1.HTTPIngressRuleValue{
										Paths: []networkingv1.HTTPIngressPath{
											{
												Backend: networkingv1.IngressBackend{
													Service: &networkingv1.IngressServiceBackend{
														Name: "app-nodeport",
													},
												},
											},
										},
									},
								},
							},
						},
					},
					Status: networkingv1.IngressStatus{},
				},
			},
			want: types.StateReady,
		}, {
			name: "expect unavailable state when there is a LoadBalancer but no address is assigned",
			args: args{
				clientset: mockClientsetK8sVersion("1", "23"),
				r: &networkingv1.Ingress{
					Spec: networkingv1.IngressSpec{
						Rules: []networkingv1.IngressRule{
							{
								IngressRuleValue: networkingv1.IngressRuleValue{
									HTTP: &networkingv1.HTTPIngressRuleValue{
										Paths: []networkingv1.HTTPIngressPath{
											{
												Backend: networkingv1.IngressBackend{
													Service: &networkingv1.IngressServiceBackend{
														Name: "app-lb",
													},
												},
											},
										},
									},
								},
							},
						},
					},
				},
			},
			want: types.StateUnavailable,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := CalculateIngressState(tt.args.clientset, tt.args.r); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("CalculateIngressState() = %v, want %v", got, tt.want)
			}
		})
	}
}
