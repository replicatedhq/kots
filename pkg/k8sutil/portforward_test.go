package k8sutil

import (
	"fmt"
	"net"
	"testing"

	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes/fake"
)

func TestIsPortAvailable(t *testing.T) {
	type args struct {
		port int
	}
	tests := []struct {
		name             string
		args             args
		nodePortServices []corev1.Service
		portsToOpen      []int
		want             bool
		wantErr          bool
	}{
		{
			name: "port is available",
			args: args{
				port: 41125,
			},
			nodePortServices: []corev1.Service{},
			portsToOpen:      []int{},
			want:             true,
			wantErr:          false,
		},
		{
			name: "port is not available due to nodeport service",
			args: args{
				port: 41125,
			},
			nodePortServices: []corev1.Service{
				{
					Spec: corev1.ServiceSpec{
						Type: corev1.ServiceTypeNodePort,
						Ports: []corev1.ServicePort{
							{
								NodePort: 41125,
							},
						},
					},
				},
			},
			portsToOpen: []int{},
			want:        false,
			wantErr:     false,
		},
		{
			name: "port is not available because it's open on the host",
			args: args{
				port: 41125,
			},
			nodePortServices: []corev1.Service{},
			portsToOpen:      []int{41125},
			want:             false,
			wantErr:          false,
		},
		{
			name: "port is available because it does not conflict with nodeport services or open ports on the host",
			args: args{
				port: 41125,
			},
			nodePortServices: []corev1.Service{
				{
					Spec: corev1.ServiceSpec{
						Type: corev1.ServiceTypeNodePort,
						Ports: []corev1.ServicePort{
							{
								NodePort: 41126,
							},
						},
					},
				},
			},
			portsToOpen: []int{41127},
			want:        true,
			wantErr:     false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			for _, port := range tt.portsToOpen {
				listener, err := net.Listen("tcp4", fmt.Sprintf("127.0.0.1:%d", port))
				require.NoError(t, err)
				defer listener.Close()
			}

			clientset := fake.NewSimpleClientset(&corev1.ServiceList{
				Items: tt.nodePortServices,
			})

			got, err := IsPortAvailable(clientset, tt.args.port)
			if (err != nil) != tt.wantErr {
				t.Errorf("IsPortAvailable() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("IsPortAvailable() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestFindFreePort(t *testing.T) {
	tests := []struct {
		name             string
		nodePortServices []corev1.Service
		portsToOpen      []int
		wantErr          bool
	}{
		{
			name:             "basic - no conflicts",
			nodePortServices: []corev1.Service{},
			portsToOpen:      []int{},
			wantErr:          false,
		},
		{
			name: "with nodeport services",
			nodePortServices: []corev1.Service{
				{
					Spec: corev1.ServiceSpec{
						Type: corev1.ServiceTypeNodePort,
						Ports: []corev1.ServicePort{
							{
								NodePort: 41125,
							},
						},
					},
				},
			},
			portsToOpen: []int{},
			wantErr:     false,
		},
		{
			name:             "with open ports",
			nodePortServices: []corev1.Service{},
			portsToOpen:      []int{41126, 41127},
			wantErr:          false,
		},
		{
			name: "with nodeport services and open ports",
			nodePortServices: []corev1.Service{
				{
					Spec: corev1.ServiceSpec{
						Type: corev1.ServiceTypeNodePort,
						Ports: []corev1.ServicePort{
							{
								NodePort: 41125,
							},
						},
					},
				},
			},
			portsToOpen: []int{41126, 41127},
			wantErr:     false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			for _, port := range tt.portsToOpen {
				listener, err := net.Listen("tcp4", fmt.Sprintf(":%d", port))
				require.NoError(t, err)
				defer listener.Close()
			}

			clientset := fake.NewSimpleClientset(&corev1.ServiceList{
				Items: tt.nodePortServices,
			})

			got, err := FindFreePort(clientset)
			if (err != nil) != tt.wantErr {
				t.Errorf("FindFreePort() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			for _, port := range tt.portsToOpen {
				if got == port {
					t.Errorf("FindFreePort() = %v, port must not match an open port", got)
				}
			}

			for _, service := range tt.nodePortServices {
				for _, port := range service.Spec.Ports {
					if got == int(port.NodePort) {
						t.Errorf("FindFreePort() = %v, port must not match a nodeport service", got)
					}
				}
			}
		})
	}
}
