package ingress

import (
	"testing"

	kotsv1beta1 "github.com/replicatedhq/kotskinds/apis/kots/v1beta1"
)

func Test_getIngressConfigAddress(t *testing.T) {
	tests := []struct {
		name          string
		ingressConfig kotsv1beta1.IngressResourceConfig
		want          string
	}{
		{
			name: "no tls",
			ingressConfig: kotsv1beta1.IngressResourceConfig{
				Host: "somebigbank.com",
				Path: "/test",
			},
			want: "http://somebigbank.com/test",
		},
		{
			name: "with tls",
			ingressConfig: kotsv1beta1.IngressResourceConfig{
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
