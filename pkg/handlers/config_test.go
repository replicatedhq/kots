package handlers

import (
	"testing"

	kotsv1beta1 "github.com/replicatedhq/kotskinds/apis/kots/v1beta1"
	"github.com/replicatedhq/kotskinds/multitype"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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
			updatedValues := updateAppConfigValues(test.values, test.configGroups)

			req.Equal(test.want, updatedValues)
		})
	}
}

func Test_mergeConfigValues(t *testing.T) {
	tests := []struct {
		name           string
		config         kotsv1beta1.Config
		existingValues kotsv1beta1.ConfigValues
		newValues      kotsv1beta1.ConfigValues
		want           kotsv1beta1.ConfigValuesSpec
		wantErr        bool
	}{
		{
			name: "merge some fields",
			config: kotsv1beta1.Config{
				TypeMeta: metav1.TypeMeta{
					APIVersion: "kots.io/v1beta1",
					Kind:       "Config",
				},
				Spec: kotsv1beta1.ConfigSpec{
					Groups: []kotsv1beta1.ConfigGroup{
						{
							Name: "pod",
							Items: []kotsv1beta1.ConfigItem{
								{
									Name: "podName",
								},
							},
						},
						{
							Name: "secrets",
							Items: []kotsv1beta1.ConfigItem{
								{
									Name: "SecretName",
									ValuesByGroup: kotsv1beta1.ValuesByGroup{
										"secrets": {
											"secretName-1": "123",
											"secretName-2": "456",
										},
									},
								},
							},
						},
					},
				},
			},
			existingValues: kotsv1beta1.ConfigValues{
				TypeMeta: metav1.TypeMeta{
					APIVersion: "kots.io/v1beta1",
					Kind:       "ConfigValues",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name: "test",
				},
				Spec: kotsv1beta1.ConfigValuesSpec{
					Values: map[string]kotsv1beta1.ConfigValue{
						"secretName-1": {
							Value:          "123",
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
				},
			},
			newValues: kotsv1beta1.ConfigValues{
				TypeMeta: metav1.TypeMeta{
					APIVersion: "kots.io/v1beta1",
					Kind:       "ConfigValues",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name: "test",
				},
				Spec: kotsv1beta1.ConfigValuesSpec{
					Values: map[string]kotsv1beta1.ConfigValue{
						"secretName-1": {
							Value:          "1234",
							RepeatableItem: "secretName",
						},
						"podName": {
							Value: "real-pod",
						},
					},
				},
			},
			want: kotsv1beta1.ConfigValuesSpec{
				Values: map[string]kotsv1beta1.ConfigValue{
					"secretName-1": {
						Value:          "1234",
						RepeatableItem: "secretName",
					},
					"secretName-2": {
						Value:          "456",
						RepeatableItem: "secretName",
					},
					"podName": {
						Value: "real-pod",
					},
				},
			},
			wantErr: false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			req := require.New(t)
			result, err := mergeConfigValues(&test.config, &test.existingValues, &test.newValues)
			req.NoError(err)

			req.Equal(test.want, result.Spec)
		})
	}
}

func Test_updateConfigObject(t *testing.T) {
	tests := []struct {
		name         string
		config       kotsv1beta1.Config
		updateValues kotsv1beta1.ConfigValues
		want         kotsv1beta1.Config
	}{
		{
			name: "update some values",
			config: kotsv1beta1.Config{
				TypeMeta: metav1.TypeMeta{
					APIVersion: "kots.io/v1beta1",
					Kind:       "Config",
				},
				Spec: kotsv1beta1.ConfigSpec{
					Groups: []kotsv1beta1.ConfigGroup{
						{
							Name: "pod",
							Items: []kotsv1beta1.ConfigItem{
								{
									Name:  "podName",
									Value: multitype.BoolOrString{Type: 0, BoolVal: false, StrVal: "testing-123"},
								},
								{
									Name:  "specialCase",
									Value: multitype.BoolOrString{Type: 1, BoolVal: true, StrVal: ""},
								},
							},
						},
						{
							Name: "secrets",
							Items: []kotsv1beta1.ConfigItem{
								{
									Name:       "secretName",
									Repeatable: true,
									ValuesByGroup: kotsv1beta1.ValuesByGroup{
										"secrets": {
											"secretName-1": "123",
											"secretName-2": "456",
										},
									},
								},
							},
						},
					},
				},
			},
			updateValues: kotsv1beta1.ConfigValues{
				TypeMeta: metav1.TypeMeta{
					APIVersion: "kots.io/v1beta1",
					Kind:       "ConfigValues",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name: "test",
				},
				Spec: kotsv1beta1.ConfigValuesSpec{
					Values: map[string]kotsv1beta1.ConfigValue{
						"secretName-1": {
							Value:          "789",
							RepeatableItem: "secretName",
						},
						"podName": {
							Value: "test-pod",
						},
					},
				},
			},
			want: kotsv1beta1.Config{
				TypeMeta: metav1.TypeMeta{
					APIVersion: "kots.io/v1beta1",
					Kind:       "Config",
				},
				Spec: kotsv1beta1.ConfigSpec{
					Groups: []kotsv1beta1.ConfigGroup{
						{
							Name: "pod",
							Items: []kotsv1beta1.ConfigItem{
								{
									Name:          "podName",
									Value:         multitype.BoolOrString{Type: 0, BoolVal: false, StrVal: "test-pod"},
									ValuesByGroup: map[string]kotsv1beta1.GroupValues{},
								},
								{
									Name:          "specialCase",
									Value:         multitype.BoolOrString{Type: 1, BoolVal: false, StrVal: ""},
									Default:       multitype.BoolOrString{Type: 1, BoolVal: false, StrVal: ""},
									ValuesByGroup: map[string]kotsv1beta1.GroupValues{},
								},
							},
						},
						{
							Name: "secrets",
							Items: []kotsv1beta1.ConfigItem{
								{
									Name:       "secretName",
									Repeatable: true,
									ValuesByGroup: kotsv1beta1.ValuesByGroup{
										"secrets": {
											"secretName-1": "789",
										},
									},
								},
							},
						},
					},
				},
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			req := require.New(t)

			results, err := updateConfigObject(&test.config, &test.updateValues, false)
			req.NoError(err)

			req.Equal(test.want, *results)
		})
	}
}
