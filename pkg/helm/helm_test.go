package helm

import (
	"reflect"
	"testing"

	kotsv1beta1 "github.com/replicatedhq/kots/kotskinds/apis/kots/v1beta1"
	"github.com/replicatedhq/kots/kotskinds/multitype"
	"github.com/replicatedhq/kots/pkg/template"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func Test_GetMergedValues(t *testing.T) {
	tests := []struct {
		name           string
		releaseValues  map[string]interface{}
		renderedValues map[string]interface{}
		expectedValues map[string]interface{}
	}{
		{
			name:           "empty values",
			releaseValues:  map[string]interface{}{},
			renderedValues: map[string]interface{}{},
			expectedValues: map[string]interface{}{},
		},
		{
			name: "top level override",
			releaseValues: map[string]interface{}{
				"nameOverride": "",
				"imageVersion": "0.1.2",
			},
			renderedValues: map[string]interface{}{
				"nameOverride": "override",
			},
			expectedValues: map[string]interface{}{
				"nameOverride": "override",
				"imageVersion": "0.1.2",
			},
		},
		{
			name: "nested override",
			releaseValues: map[string]interface{}{
				"name": map[string]interface{}{
					"override": "",
				},
				"imageVersion": "0.1.2",
			},
			renderedValues: map[string]interface{}{
				"name": map[string]interface{}{
					"override": "override",
				},
			},
			expectedValues: map[string]interface{}{
				"name": map[string]interface{}{
					"override": "override",
				},
				"imageVersion": "0.1.2",
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			mergedValues, err := GetMergedValues(test.releaseValues, test.renderedValues)
			if !reflect.DeepEqual(mergedValues, test.expectedValues) {
				t.Errorf("GetMergedValues() = %v, want %v", mergedValues, test.expectedValues)
			}
			if err != nil {
				t.Errorf("GetMergedValues() threw an error = %v", err)
			}
		})
	}
}

func Test_RenderValuesFromConfig(t *testing.T) {
	tests := []struct {
		name           string
		app            string
		newConfigItems map[string]template.ItemValue
		config         *kotsv1beta1.Config
		chart          []byte
		expectedValues map[string]interface{}
		expectedConfig *kotsv1beta1.Config
	}{
		{
			name: "top level override",
			app:  "testapp",
			newConfigItems: map[string]template.ItemValue{
				"myhelmvalue": template.ItemValue{
					Value: "myValue",
				},
			},
			config: &kotsv1beta1.Config{
				TypeMeta: metav1.TypeMeta{
					Kind:       "Config",
					APIVersion: "kots.io/v1beta1",
				},
				ObjectMeta: metav1.ObjectMeta{},
				Spec: kotsv1beta1.ConfigSpec{
					Groups: []kotsv1beta1.ConfigGroup{
						{
							Name:        "myGroup",
							Title:       "groupTitle",
							Description: "testing group",
							When:        multitype.QuotedBool("false"),
							Items: []kotsv1beta1.ConfigItem{
								{
									Name: "myhelmvalue",
									Value: multitype.BoolOrString{
										Type:   multitype.String,
										StrVal: "myValue",
									},
								},
							},
						},
					},
				},
				Status: kotsv1beta1.ConfigStatus{},
			},
			chart: []byte(`apiVersion: kots.io/v1beta1
kind: HelmChart
metadata:
  name: test-chart
spec:
  chart:
  name: test-chart
  chartVersion: 0.3.17
  helmVersion: v3
  useHelmInstall: true
  values: 
    myHelmValue: repl{{ConfigOption "myhelmvalue"}}
builder: {}`),
			expectedValues: map[string]interface{}{
				"myHelmValue": "myValue",
			},
			expectedConfig: &kotsv1beta1.Config{
				TypeMeta: metav1.TypeMeta{
					Kind:       "Config",
					APIVersion: "kots.io/v1beta1",
				},
				ObjectMeta: metav1.ObjectMeta{},
				Spec: kotsv1beta1.ConfigSpec{
					Groups: []kotsv1beta1.ConfigGroup{
						{
							Name:        "myGroup",
							Title:       "groupTitle",
							Description: "testing group",
							When:        multitype.QuotedBool("false"),
							Items: []kotsv1beta1.ConfigItem{
								{
									Name: "myhelmvalue",
									Value: multitype.BoolOrString{
										Type:   multitype.String,
										StrVal: "myValue",
									},
								},
							},
						},
					},
				},
				Status: kotsv1beta1.ConfigStatus{},
			},
		},
		{
			name: "nested override",
			app:  "testapp",
			newConfigItems: map[string]template.ItemValue{
				"myhelmvalue": template.ItemValue{
					Value: "myValue",
				},
			},
			config: &kotsv1beta1.Config{
				TypeMeta: metav1.TypeMeta{
					Kind:       "Config",
					APIVersion: "kots.io/v1beta1",
				},
				ObjectMeta: metav1.ObjectMeta{},
				Spec: kotsv1beta1.ConfigSpec{
					Groups: []kotsv1beta1.ConfigGroup{
						{
							Name:        "myGroup",
							Title:       "groupTitle",
							Description: "testing group",
							When:        multitype.QuotedBool("false"),
							Items: []kotsv1beta1.ConfigItem{
								{
									Name: "myhelmvalue",
									Value: multitype.BoolOrString{
										Type:   multitype.String,
										StrVal: "myValue",
									},
								},
							},
						},
					},
				},
				Status: kotsv1beta1.ConfigStatus{},
			},
			chart: []byte(`apiVersion: kots.io/v1beta1
kind: HelmChart
metadata:
  name: test-chart
spec:
  chart:
  name: test-chart
  chartVersion: 0.3.17
  helmVersion: v3
  useHelmInstall: true
  values: 
    myHelmValue: 
      toOverride: repl{{ConfigOption "myhelmvalue"}}
builder: {}`),
			expectedValues: map[string]interface{}{
				"myHelmValue": map[string]interface{}{
					"toOverride": "myValue",
				},
			},
			expectedConfig: &kotsv1beta1.Config{
				TypeMeta: metav1.TypeMeta{
					Kind:       "Config",
					APIVersion: "kots.io/v1beta1",
				},
				ObjectMeta: metav1.ObjectMeta{},
				Spec: kotsv1beta1.ConfigSpec{
					Groups: []kotsv1beta1.ConfigGroup{
						{
							Name:        "myGroup",
							Title:       "groupTitle",
							Description: "testing group",
							When:        multitype.QuotedBool("false"),
							Items: []kotsv1beta1.ConfigItem{
								{
									Name: "myhelmvalue",
									Value: multitype.BoolOrString{
										Type:   multitype.String,
										StrVal: "myValue",
									},
								},
							},
						},
					},
				},
				Status: kotsv1beta1.ConfigStatus{},
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			renderedValues, renderedConfig, err := RenderValuesFromConfig(test.app, test.newConfigItems, test.config, test.chart)
			if !reflect.DeepEqual(renderedValues, test.expectedValues) {
				t.Errorf("RenderValuesFromConfig() = %v, want %v", renderedValues, test.expectedValues)
			}
			if !reflect.DeepEqual(renderedConfig, test.expectedConfig) {
				t.Errorf("RenderValuesFromConfig() = %v, want %v", renderedConfig, test.expectedConfig)
			}
			if err != nil {
				t.Errorf("RenderValuesFromConfig() threw an error = %v", err)
			}
		})
	}
}
