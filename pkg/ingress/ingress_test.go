package ingress

import (
	"testing"

	"github.com/replicatedhq/kots/pkg/ingress/types"
	extensions "k8s.io/api/extensions/v1beta1"
)

func Test_getIngressConfigAddress(t *testing.T) {
	tests := []struct {
		name          string
		ingressConfig types.IngressConfig
		want          string
	}{
		{
			name: "address",
			ingressConfig: types.IngressConfig{
				Address: "http://somebigbank.com",
				Path:    "/test",
			},
			want: "http://somebigbank.com",
		},
		{
			name: "no address",
			ingressConfig: types.IngressConfig{
				Host: "somebigbank.com",
				Path: "/test",
			},
			want: "http://somebigbank.com/test",
		},
		{
			name: "no address tls",
			ingressConfig: types.IngressConfig{
				Host: "somebigbank.com",
				Path: "/test",
				TLS: []extensions.IngressTLS{
					{
						Hosts:      []string{"somebigbank.com"},
						SecretName: "kotsadm-tls",
					},
				},
			},
			want: "https://somebigbank.com/test",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := getIngressConfigAddress(tt.ingressConfig); got != tt.want {
				t.Errorf("getIngressConfigAddress() = %v, want %v", got, tt.want)
			}
		})
	}
}
