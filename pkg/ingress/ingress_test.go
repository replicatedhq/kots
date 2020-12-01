package ingress

import (
	"testing"

	"github.com/replicatedhq/kots/pkg/ingress/types"
)

func Test_getIngressConfigAddress(t *testing.T) {
	tests := []struct {
		name          string
		ingressConfig types.IngressConfig
		want          string
	}{
		{
			name: "no tls",
			ingressConfig: types.IngressConfig{
				Host: "somebigbank.com",
				Path: "/test",
			},
			want: "http://somebigbank.com/test",
		},
		{
			name: "with tls",
			ingressConfig: types.IngressConfig{
				Host:          "somebigbank.com",
				Path:          "/test",
				TLSSecretName: "kotsadm-tls",
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
