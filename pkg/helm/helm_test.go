package helm

import (
	"reflect"
	"testing"

	apptypes "github.com/replicatedhq/kots/pkg/app/types"
	"github.com/replicatedhq/kots/pkg/kotsutil"
	kotsv1beta1 "github.com/replicatedhq/kotskinds/apis/kots/v1beta1"
	helmrelease "helm.sh/helm/v3/pkg/release"
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
	app := &apptypes.HelmApp{
		Version: 1,
		Release: helmrelease.Release{
			Name: "testapp",
		},
	}

	license := &kotsv1beta1.License{
		Spec: kotsv1beta1.LicenseSpec{},
	}

	tests := []struct {
		name           string
		app            *apptypes.HelmApp
		newConfigItems *kotsv1beta1.ConfigValues
		chart          []byte
		expectedValues map[string]interface{}
	}{
		{
			name: "top level override",
			app:  app,
			newConfigItems: &kotsv1beta1.ConfigValues{
				TypeMeta: metav1.TypeMeta{
					Kind:       "ConfigValues",
					APIVersion: "kots.io/v1beta1",
				},
				Spec: kotsv1beta1.ConfigValuesSpec{
					Values: map[string]kotsv1beta1.ConfigValue{
						"myhelmvalue": {
							Value: "myValue",
						},
					},
				},
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
		},
		{
			name: "nested override",
			app:  app,
			newConfigItems: &kotsv1beta1.ConfigValues{
				TypeMeta: metav1.TypeMeta{
					Kind:       "ConfigValues",
					APIVersion: "kots.io/v1beta1",
				},
				Spec: kotsv1beta1.ConfigValuesSpec{
					Values: map[string]kotsv1beta1.ConfigValue{
						"myhelmvalue": {
							Value: "myValue",
						},
					},
				},
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
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			kotsKinds := kotsutil.EmptyKotsKinds()
			kotsKinds.License = license
			kotsKinds.ConfigValues = test.newConfigItems

			renderedValues, err := RenderValuesFromConfig(test.app, &kotsKinds, test.chart)
			if !reflect.DeepEqual(renderedValues, test.expectedValues) {
				t.Errorf("RenderValuesFromConfig() = %v, want %v", renderedValues, test.expectedValues)
			}
			if err != nil {
				t.Errorf("RenderValuesFromConfig() threw an error = %v", err)
			}
		})
	}
}
