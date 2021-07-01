package handlers

import (
	"reflect"
	"testing"

	kotsv1beta1 "github.com/replicatedhq/kots/kotskinds/apis/kots/v1beta1"
	"github.com/replicatedhq/kots/kotskinds/multitype"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func Test_updateAppConfigValues(t *testing.T) {
	tests := []struct {
		values       map[string]kotsv1beta1.ConfigValue
		configGroups []kotsv1beta1.ConfigGroup
		want         map[string]kotsv1beta1.ConfigValue
	}{
		{
			values: map[string]kotsv1beta1.ConfigValue{
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
					Value:          "111",
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
		updatedValues, err := updateAppConfigValues(test.values, test.configGroups, "")
		if err != nil {
			t.Errorf("updateAppConfigValues() test failed with err %v", err)
		}
		if !reflect.DeepEqual(updatedValues, test.want) {
			t.Errorf("updateAppConfigValues() failed: want: \n%+v\n got: \n%+v", test.want, updatedValues)
		}
	}
}

func Test_mergeConfigValues(t *testing.T) {
	tests := []struct {
		config         kotsv1beta1.Config
		existingValues kotsv1beta1.ConfigValues
		newValues      kotsv1beta1.ConfigValues
		want           kotsv1beta1.ConfigValuesSpec
		wantErr        bool
	}{
		{
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
						"secretName-2": {
							Value:          "4567",
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
						Value:          "4567",
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
		result, err := mergeConfigValues(&test.config, &test.existingValues, &test.newValues)
		if !test.wantErr && err != nil {
			t.Errorf("mergeConfigValues() test failed with err: %v", err)
			return
		}
		if !reflect.DeepEqual(test.want, result.Spec) {
			t.Errorf("mergeConfigValues() failed: \nwant:\n%+v\ngot:\n%+v", test.want, result.Spec)
		}
	}
}

func Test_updateConfigObject(t *testing.T) {
	tests := []struct {
		config       kotsv1beta1.Config
		updateValues kotsv1beta1.ConfigValues
		want         kotsv1beta1.Config
	}{
		{
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
									Value: multitype.BoolOrString{0, false, "testing-123"},
								},
								{
									Name:  "specialCase",
									Value: multitype.BoolOrString{1, true, ""},
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
									Value:         multitype.BoolOrString{0, false, "test-pod"},
									ValuesByGroup: map[string]kotsv1beta1.GroupValues{},
								},
								{
									Name:          "specialCase",
									Value:         multitype.BoolOrString{1, false, ""},
									Default:       multitype.BoolOrString{1, false, ""},
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
		results, err := updateConfigObject(&test.config, &test.updateValues, false)
		if err != nil {
			t.Errorf("updateConfigObject() failed with err: %v", err)
			return
		}
		if !reflect.DeepEqual(&test.want.Spec.Groups, results.Spec.Groups) {
			t.Errorf("updateConfigObject() failed:\nwant:\n%+v\ngot:\n%+v", &test.want, results)
		}
	}
}
