package kotsadmconfig

import (
	"testing"

	kotsv1beta1 "github.com/replicatedhq/kotskinds/apis/kots/v1beta1"
	"github.com/replicatedhq/kotskinds/multitype"
	"github.com/stretchr/testify/require"
)

func Test_updateAppConfigValues(t *testing.T) {
	tests := []struct {
		name         string
		values       map[string]kotsv1beta1.ConfigValue
		configGroups []kotsv1beta1.ConfigGroup
		want         map[string]kotsv1beta1.ConfigValue
	}{
		{
			name: "update config values",
			values: map[string]kotsv1beta1.ConfigValue{
				"secretName-1": {
					Value:          "111",
					RepeatableItem: "secretName",
				},
				"secretName-2": {
					Value:          "456",
					RepeatableItem: "secretName",
				},
				"podName": {
					Value: "test-pod",
				},
			},
			configGroups: []kotsv1beta1.ConfigGroup{
				{
					Name: "secret",
					Items: []kotsv1beta1.ConfigItem{
						{
							Name: "secretName",
							ValuesByGroup: kotsv1beta1.ValuesByGroup{
								"Secrets": {
									"secretName-1": "123",
									"secretName-2": "456",
								},
							},
						},
					},
				},
				{
					Name: "pod",
					Items: []kotsv1beta1.ConfigItem{
						{
							Name:  "podName",
							Value: multitype.BoolOrString{Type: 0, StrVal: "real-pod"},
						},
					},
				},
			},
			want: map[string]kotsv1beta1.ConfigValue{
				"podName": {
					Value: "real-pod",
				},
				"secretName": {},
				"secretName-1": {
					Value:          "123",
					RepeatableItem: "secretName",
				},
				"secretName-2": {
					Value:          "456",
					RepeatableItem: "secretName",
				},
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			req := require.New(t)
			updatedValues := UpdateAppConfigValues(test.values, test.configGroups)

			req.Equal(test.want, updatedValues)
		})
	}
}
