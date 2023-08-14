package reporting

import (
	"reflect"
	"testing"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/version"
	discoveryfake "k8s.io/client-go/discovery/fake"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/fake"
)

type mockClientsetForDistributionOpts struct {
	objects       []runtime.Object
	k8sVersion    string
	groupVersions []string
}

func mockClientsetForDistribution(opts *mockClientsetForDistributionOpts) kubernetes.Interface {
	clientset := fake.NewSimpleClientset(opts.objects...)
	resources := []*metav1.APIResourceList{}
	for _, groupVersion := range opts.groupVersions {
		resources = append(resources, &metav1.APIResourceList{
			GroupVersion: groupVersion,
		})
	}
	clientset.Discovery().(*discoveryfake.FakeDiscovery).Resources = resources
	clientset.Discovery().(*discoveryfake.FakeDiscovery).FakedServerVersion = &version.Info{
		GitVersion: opts.k8sVersion,
	}
	return clientset
}

func TestGetDistribution(t *testing.T) {
	type args struct {
		clientset kubernetes.Interface
	}
	tests := []struct {
		name string
		args args
		want Distribution
	}{
		{
			name: "openshift from api groups and resources",
			args: args{
				clientset: mockClientsetForDistribution(&mockClientsetForDistributionOpts{
					groupVersions: []string{"apps.openshift.io/v1"},
					k8sVersion:    "v1.26.0",
				}),
			},
			want: OpenShift,
		},
		{
			name: "tanzu from api groups and resources",
			args: args{
				clientset: mockClientsetForDistribution(&mockClientsetForDistributionOpts{
					groupVersions: []string{"run.tanzu.vmware.com/v1"},
					k8sVersion:    "v1.26.0",
				}),
			},
			want: Tanzu,
		},
		{
			name: "kind from provider id",
			args: args{
				clientset: mockClientsetForDistribution(&mockClientsetForDistributionOpts{
					objects: []runtime.Object{
						&corev1.Node{
							Spec: corev1.NodeSpec{
								ProviderID: "kind:foo",
							},
						},
					},
					k8sVersion: "v1.26.0",
				}),
			},
			want: Kind,
		},
		{
			name: "digitalocean from provider id",
			args: args{
				clientset: mockClientsetForDistribution(&mockClientsetForDistributionOpts{
					objects: []runtime.Object{
						&corev1.Node{
							Spec: corev1.NodeSpec{
								ProviderID: "digitalocean:foo",
							},
						},
					},
				}),
			},
			want: DigitalOcean,
		},
		{
			name: "kurl from labels",
			args: args{
				clientset: mockClientsetForDistribution(&mockClientsetForDistributionOpts{
					objects: []runtime.Object{
						&corev1.Node{
							ObjectMeta: metav1.ObjectMeta{
								Labels: map[string]string{
									"kurl.sh/cluster": "true",
								},
							},
						},
					},
				}),
			},
			want: Kurl,
		},
		{
			name: "microk8s from labels",
			args: args{
				clientset: mockClientsetForDistribution(&mockClientsetForDistributionOpts{
					objects: []runtime.Object{
						&corev1.Node{
							ObjectMeta: metav1.ObjectMeta{
								Labels: map[string]string{
									"microk8s.io/cluster": "true",
								},
							},
						},
					},
				}),
			},
			want: MicroK8s,
		},
		{
			name: "azure from labels",
			args: args{
				clientset: mockClientsetForDistribution(&mockClientsetForDistributionOpts{
					objects: []runtime.Object{
						&corev1.Node{
							ObjectMeta: metav1.ObjectMeta{
								Labels: map[string]string{
									"kubernetes.azure.com/role": "foo",
								},
							},
						},
					},
				}),
			},
			want: AKS,
		},
		{
			name: "minikube from labels",
			args: args{
				clientset: mockClientsetForDistribution(&mockClientsetForDistributionOpts{
					objects: []runtime.Object{
						&corev1.Node{
							ObjectMeta: metav1.ObjectMeta{
								Labels: map[string]string{
									"minikube.k8s.io/version": "123",
								},
							},
						},
					},
				}),
			},
			want: Minikube,
		},
		{
			name: "gke from version",
			args: args{
				clientset: mockClientsetForDistribution(&mockClientsetForDistributionOpts{
					k8sVersion: "v1.26.0-gke.1",
				}),
			},
			want: GKE,
		},
		{
			name: "eks from version",
			args: args{
				clientset: mockClientsetForDistribution(&mockClientsetForDistributionOpts{
					k8sVersion: "v1.26.0-eks-1-123",
				}),
			},
			want: EKS,
		},
		{
			name: "rke2 from version",
			args: args{
				clientset: mockClientsetForDistribution(&mockClientsetForDistributionOpts{
					k8sVersion: "v1.26.0+rke2",
				}),
			},
			want: RKE2,
		},
		{
			name: "k3s from version",
			args: args{
				clientset: mockClientsetForDistribution(&mockClientsetForDistributionOpts{
					k8sVersion: "v1.26.0+k3s",
				}),
			},
			want: K3s,
		},
		{
			name: "k0s from version",
			args: args{
				clientset: mockClientsetForDistribution(&mockClientsetForDistributionOpts{
					k8sVersion: "v1.26.0+k0s",
				}),
			},
			want: K0s,
		},
		{
			name: "unknown from version",
			args: args{
				clientset: mockClientsetForDistribution(&mockClientsetForDistributionOpts{
					k8sVersion: "v1.26.0",
				}),
			},
			want: UnknownDistribution,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := GetDistribution(tt.args.clientset); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("GetDistribution() = %v, want %v", got, tt.want)
			}
		})
	}
}
