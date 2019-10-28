package types

import (
	"reflect"
	"testing"
)

func TestStatusInformerString_Parse(t *testing.T) {
	tests := []struct {
		name    string
		str     string
		want    StatusInformer
		wantErr bool
	}{
		{
			name: "kind/name",
			str:  "deploy/sentry-web",
			want: StatusInformer{
				Kind: "deploy",
				Name: "sentry-web",
			},
		},
		{
			name: "namespace/kind/name",
			str:  "default/deploy/sentry-web",
			want: StatusInformer{
				Namespace: "default",
				Kind:      "deploy",
				Name:      "sentry-web",
			},
		},
		{
			name:    "no match",
			str:     "sentry-web",
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := StatusInformerString(tt.str).Parse()
			if (err != nil) != tt.wantErr {
				t.Errorf("StatusInformerString.Parse() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("StatusInformerString.Parse() = %v, want %v", got, tt.want)
			}
		})
	}
}
