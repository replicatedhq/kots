package client

import (
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"testing"
)

func Test_getLabelSelector(t *testing.T) {
	tests := []struct {
		name             string
		appLabelSelector metav1.LabelSelector
		want             string
	}{
		{
			name: "no requirements",
			appLabelSelector: metav1.LabelSelector{
				MatchLabels:      nil,
				MatchExpressions: nil,
			},
			want: "",
		},
		{
			name: "one requirement",
			appLabelSelector: metav1.LabelSelector{
				MatchLabels: map[string]string{
					"kots.io/label": "abc",
				},
				MatchExpressions: nil,
			},
			want: "kots.io/label=abc",
		},
		{
			name: "two requirements",
			appLabelSelector: metav1.LabelSelector{
				MatchLabels: map[string]string{
					"kots.io/label": "abc",
					"otherlabel":    "xyz",
				},
				MatchExpressions: nil,
			},
			want: "kots.io/label=abc,otherlabel=xyz",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.Equal(t, tt.want, getLabelSelector(&tt.appLabelSelector))
		})
	}
}
