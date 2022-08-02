package v1beta1

import (
	"fmt"
	"strings"
	"testing"

	"github.com/go-test/deep"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_HelmChartSpecRenderValues(t *testing.T) {
	tests := []struct {
		name   string
		values map[string]MappedChartValue
		expect map[string]interface{}
	}{
		{
			name: "simple",
			values: map[string]MappedChartValue{
				"a": MappedChartValue{
					strValue:  "b",
					valueType: "string",
				},
			},
			expect: map[string]interface{}{"a": "b"},
		},
		{
			name: "string with comma",
			values: map[string]MappedChartValue{
				"a": MappedChartValue{
					strValue:  "b,c,d",
					valueType: "string",
				},
			},
			expect: map[string]interface{}{"a": "b,c,d"},
		},
		{

			name: "with-child",
			values: map[string]MappedChartValue{
				"postgres": MappedChartValue{
					valueType: "children",
					children: map[string]*MappedChartValue{
						"enabled": &MappedChartValue{
							boolValue: true,
							valueType: "bool",
						},
					},
				},
			},
			expect: map[string]interface{}{"postgres": map[string]interface{}{"enabled": true}},
		},
		{
			name: "children with array",
			values: map[string]MappedChartValue{
				"worker": MappedChartValue{
					valueType: "children",
					children: map[string]*MappedChartValue{
						"queues": &MappedChartValue{
							valueType: "array",
							array: []*MappedChartValue{
								{
									valueType: "children",
									children: map[string]*MappedChartValue{
										"queue": &MappedChartValue{
											strValue:  "first",
											valueType: "string",
										},
										"replicas": &MappedChartValue{
											floatValue: float64(1),
											valueType:  "float",
										},
									},
								},
								{
									valueType: "children",
									children: map[string]*MappedChartValue{
										"queue": &MappedChartValue{
											strValue:  "second",
											valueType: "string",
										},
										"replicas": &MappedChartValue{
											floatValue: float64(2),
											valueType:  "float",
										},
									},
								},
							},
						},
					},
				},
			},
			expect: map[string]interface{}(
				map[string]interface{}{
					"worker": map[string]interface{}{
						"queues": []interface{}{
							map[string]interface{}{"queue": "first", "replicas": float64(1)},
							map[string]interface{}{"queue": "second", "replicas": float64(2)},
						},
					},
				},
			),
		},
		{
			name: "with-deep-children",
			values: map[string]MappedChartValue{
				"storage": MappedChartValue{
					valueType: "children",
					children: map[string]*MappedChartValue{
						"postgres": &MappedChartValue{
							valueType: "children",
							children: map[string]*MappedChartValue{
								"enabled": &MappedChartValue{
									boolValue: true,
									valueType: "bool",
								},
								"replacementtest": &MappedChartValue{
									strValue:  "something",
									valueType: `string`,
								},
							},
						},
					},
				},
			},
			expect: map[string]interface{}(
				map[string]interface{}{
					"storage": map[string]interface{}{
						"postgres": map[string]interface{}{
							"enabled":         true,
							"replacementtest": "something",
						},
					},
				},
			),
		},
		{
			name: "complex",
			values: map[string]MappedChartValue{
				"replicas": MappedChartValue{
					valueType:  "float",
					floatValue: float64(4),
				},
				"storage": MappedChartValue{
					valueType: "children",
					children: map[string]*MappedChartValue{
						"postgres": &MappedChartValue{
							valueType: "children",
							children: map[string]*MappedChartValue{
								"enabled": &MappedChartValue{
									boolValue: true,
									valueType: "bool",
								},
								"host": &MappedChartValue{
									strValue:  "amazonaws.com",
									valueType: "string",
								},
							},
						},
					},
				},
			},
			expect: map[string]interface{}(
				map[string]interface{}{
					"replicas": float64(4),
					"storage": map[string]interface{}{
						"postgres": map[string]interface{}{
							"enabled": true,
							"host":    "amazonaws.com",
						},
					},
				},
			),
		},
		{
			name: "with a map",
			values: map[string]MappedChartValue{
				"ingress": MappedChartValue{
					valueType: "children",
					children: map[string]*MappedChartValue{
						"enabled": &MappedChartValue{
							boolValue: true,
							valueType: "bool",
						},
						"annotations": &MappedChartValue{
							valueType: "children",
							children: map[string]*MappedChartValue{
								"kubernetes.io/ingress.class": &MappedChartValue{
									strValue:  "nginx",
									valueType: "string",
								},
								"anotherstring": &MappedChartValue{
									strValue:  "something",
									valueType: `string`,
								},
							},
						},
					},
				},
			},
			expect: map[string]interface{}(
				map[string]interface{}{
					"ingress": map[string]interface{}{
						"annotations": map[string]interface{}{
							"kubernetes.io/ingress.class": "nginx",
							"anotherstring":               "something",
						},
						"enabled": true,
					},
				},
			),
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			req := require.New(t)

			h := HelmChartSpec{
				Values: test.values,
			}
			actual, err := h.GetHelmValues(h.Values)
			req.NoError(err)

			assert.Equal(t, test.expect, actual)
		})
	}
}

func Test_MappedChartValueGetValue(t *testing.T) {
	tests := []struct {
		name             string
		mappedChartValue MappedChartValue
		expected         interface{}
	}{
		{
			name: "string",
			mappedChartValue: MappedChartValue{
				strValue:  "abc",
				valueType: "string",
			},
			expected: "abc",
		},
		{
			name: "string with comma",
			mappedChartValue: MappedChartValue{
				strValue:  "abc,def,ghi",
				valueType: "string",
			},
			expected: `abc,def,ghi`,
		},
		{
			name: "bool",
			mappedChartValue: MappedChartValue{
				boolValue: true,
				valueType: "bool",
			},
			expected: true,
		},
		{
			name: "float",
			mappedChartValue: MappedChartValue{
				floatValue: 42,
				valueType:  "float",
			},
			expected: float64(42),
		},
		{
			name: "children",
			mappedChartValue: MappedChartValue{
				valueType: "children",
				children: map[string]*MappedChartValue{
					"child": &MappedChartValue{
						strValue:  "val",
						valueType: "string",
					},
				},
			},
			expected: map[string]interface{}{
				"child": "val",
			},
		},
		{
			name: "array",
			mappedChartValue: MappedChartValue{
				valueType: "array",
				array: []*MappedChartValue{
					&MappedChartValue{
						strValue:  "val1",
						valueType: "string",
					},
					&MappedChartValue{
						strValue:  "val2",
						valueType: "string",
					},
				},
			},
			expected: []interface{}{
				"val1",
				"val2",
			},
		},
		{
			name: "children with array",
			mappedChartValue: MappedChartValue{
				valueType: "children",
				children: map[string]*MappedChartValue{
					"child": &MappedChartValue{
						array: []*MappedChartValue{
							&MappedChartValue{
								strValue:  "val1",
								valueType: "string",
							},
							&MappedChartValue{
								strValue:  "val2",
								valueType: "string",
							},
						},
						valueType: "array",
					},
				},
			},
			expected: map[string]interface{}{
				"child": []interface{}{
					"val1",
					"val2",
				},
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			req := require.New(t)

			actual, err := test.mappedChartValue.getBuiltValue()
			req.NoError(err)

			assert.Equal(t, test.expected, actual)
		})
	}
}

func Test_MergeHelmChartValues(t *testing.T) {
	tests := []struct {
		name          string
		baseValues    map[string]MappedChartValue
		overlayValues map[string]MappedChartValue
		expect        map[string]MappedChartValue
	}{
		{
			name: "with-child",
			baseValues: map[string]MappedChartValue{
				"postgres": MappedChartValue{
					valueType: "children",
					children: map[string]*MappedChartValue{
						"enabled": &MappedChartValue{
							boolValue: false,
							valueType: "bool",
						},
					},
				},
			},
			overlayValues: map[string]MappedChartValue{
				"postgres": MappedChartValue{
					valueType: "children",
					children: map[string]*MappedChartValue{
						"enabled": &MappedChartValue{
							boolValue: true,
							valueType: "bool",
						},
					},
				},
			},
			expect: map[string]MappedChartValue{
				"postgres": MappedChartValue{
					valueType: "children",
					children: map[string]*MappedChartValue{
						"enabled": &MappedChartValue{
							boolValue: true,
							valueType: "bool",
						},
					},
				},
			},
		},
		{
			name: "base-only-no-overlay",
			baseValues: map[string]MappedChartValue{
				"postgres": MappedChartValue{
					valueType: "children",
					children: map[string]*MappedChartValue{
						"enabled": &MappedChartValue{
							boolValue: false,
							valueType: "bool",
						},
					},
				},
			},
			overlayValues: map[string]MappedChartValue{},
			expect: map[string]MappedChartValue{
				"postgres": MappedChartValue{
					valueType: "children",
					children: map[string]*MappedChartValue{
						"enabled": &MappedChartValue{
							boolValue: false,
							valueType: "bool",
						},
					},
				},
			},
		},
		{
			name:       "no-base-only-overlay",
			baseValues: map[string]MappedChartValue{},
			overlayValues: map[string]MappedChartValue{
				"postgres": MappedChartValue{
					valueType: "children",
					children: map[string]*MappedChartValue{
						"enabled": &MappedChartValue{
							boolValue: false,
							valueType: "bool",
						},
					},
				},
			},
			expect: map[string]MappedChartValue{
				"postgres": MappedChartValue{
					valueType: "children",
					children: map[string]*MappedChartValue{
						"enabled": &MappedChartValue{
							boolValue: false,
							valueType: "bool",
						},
					},
				},
			},
		},
		{
			name: "with-deep-children",
			baseValues: map[string]MappedChartValue{
				"storage": MappedChartValue{
					valueType: "children",
					children: map[string]*MappedChartValue{
						"postgres": &MappedChartValue{
							valueType: "children",
							children: map[string]*MappedChartValue{
								"enabled": &MappedChartValue{
									boolValue: true,
									valueType: "bool",
								},
								"replacementtest": &MappedChartValue{
									strValue:  "somethinghello",
									valueType: `string`,
								},
							},
						},
					},
				},
			},
			overlayValues: map[string]MappedChartValue{
				"storage": MappedChartValue{
					valueType: "children",
					children: map[string]*MappedChartValue{
						"postgres": &MappedChartValue{
							valueType: "children",
							children: map[string]*MappedChartValue{
								"enabled": &MappedChartValue{
									boolValue: true,
									valueType: "bool",
								},
								"replacementtest": &MappedChartValue{
									strValue:  "somethingOverwritten",
									valueType: `string`,
								},
							},
						},
					},
				},
			},
			expect: map[string]MappedChartValue{
				"storage": MappedChartValue{
					valueType: "children",
					children: map[string]*MappedChartValue{
						"postgres": &MappedChartValue{
							valueType: "children",
							children: map[string]*MappedChartValue{
								"enabled": &MappedChartValue{
									boolValue: true,
									valueType: "bool",
								},
								"replacementtest": &MappedChartValue{
									strValue:  "somethingOverwritten",
									valueType: `string`,
								},
							},
						},
					},
				},
			},
		},
		{
			name: "with-deep-children-missing-overlay",
			baseValues: map[string]MappedChartValue{
				"storage": MappedChartValue{
					valueType: "children",
					children: map[string]*MappedChartValue{
						"postgres": &MappedChartValue{
							valueType: "children",
							children: map[string]*MappedChartValue{
								"enabled": &MappedChartValue{
									boolValue: true,
									valueType: "bool",
								},
								"replacementtest": &MappedChartValue{
									strValue:  "somethinghello",
									valueType: `string`,
								},
							},
						},
					},
				},
			},
			overlayValues: map[string]MappedChartValue{
				"storage": MappedChartValue{
					valueType: "children",
					children: map[string]*MappedChartValue{
						"postgres": &MappedChartValue{
							valueType: "children",
							children: map[string]*MappedChartValue{
								"enabled": &MappedChartValue{
									boolValue: true,
									valueType: "bool",
								},
							},
						},
					},
				},
			},
			expect: map[string]MappedChartValue{
				"storage": MappedChartValue{
					valueType: "children",
					children: map[string]*MappedChartValue{
						"postgres": &MappedChartValue{
							valueType: "children",
							children: map[string]*MappedChartValue{
								"enabled": &MappedChartValue{
									boolValue: true,
									valueType: "bool",
								},
								"replacementtest": &MappedChartValue{
									strValue:  "somethinghello",
									valueType: `string`,
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
			actual := MergeHelmChartValues(test.baseValues, test.overlayValues)
			deep.CompareUnexportedFields = true
			diff := deep.Equal(&actual, &test.expect)
			if len(diff) != 0 {
				fmt.Printf("Failed diff compare with %s", strings.Join(diff, "\n"))
				assert.NotEqual(t, test.expect, actual)
			}
		})
	}
}

func Test_Intersect(t *testing.T) {
	tests := []struct {
		name   string
		v      interface{}
		k      string
		values map[string]interface{}
		expect interface{}
	}{
		{
			name: "string value",
			v:    "repl{{ConfigOption stringValue}}",
			k:    "stringKey",
			values: map[string]interface{}{
				"stringKey": "valueIWant",
			},
			expect: "valueIWant",
		}, {
			name: "int value",
			v:    2,
			k:    "intKey",
			values: map[string]interface{}{
				"intKey": 1,
			},
			expect: 1,
		}, {
			name: "nested value",
			v:    2,
			k:    "intKey",
			values: map[string]interface{}{
				"myKeys": map[string]interface{}{
					"intKey": 1,
				},
			},
			expect: 1,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			actual := intersect(test.v, test.k, test.values)
			diff := deep.Equal(&actual, &test.expect)
			if len(diff) != 0 {
				fmt.Printf("Failed diff compare with %s", strings.Join(diff, "\n"))
				assert.NotEqual(t, test.expect, actual)
			}
		})
	}
}

func Test_GetMapIntersect(t *testing.T) {
	tests := []struct {
		name   string
		m1     map[string]interface{}
		m2     map[string]interface{}
		expect map[string]interface{}
	}{
		{
			name: "root level",
			m1: map[string]interface{}{
				"key":  "repl{{ConfigOption myconfigkey}}",
				"key2": "should not be in result",
			},
			m2: map[string]interface{}{
				"key":  "myrenderedvalue",
				"key3": "should not be in result",
			},
			expect: map[string]interface{}{
				"key": "myrenderedvalue",
			},
		}, {
			name: "nested key",
			m1: map[string]interface{}{
				"key": map[string]interface{}{
					"value": "repl{{ConfigOption myconfigkey}}",
				},
			},
			m2: map[string]interface{}{
				"key": map[string]interface{}{
					"value":  "myvalue",
					"value2": "should not be present",
				},
			},
			expect: map[string]interface{}{
				"key": map[string]interface{}{
					"value": "myvalue",
				},
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			req := require.New(t)
			actual, err := GetMapIntersect(test.m1, test.m2)
			req.NoError(err)
			diff := deep.Equal(&actual, &test.expect)
			if len(diff) != 0 {
				fmt.Printf("Failed diff compare with %s", strings.Join(diff, "\n"))
				assert.NotEqual(t, test.expect, actual)
			}
		})
	}
}

func Test_GetReplTmplValues(t *testing.T) {
	tests := []struct {
		name   string
		spec   *HelmChartSpec
		values map[string]MappedChartValue
		expect map[string]interface{}
	}{
		{
			name: "root level",
			values: map[string]MappedChartValue{
				"key": MappedChartValue{
					strValue: "{{repl ConfigOption 'test'}}",
				},
				"key2": MappedChartValue{
					strValue: "repl{{ ConfigOption 'test'}}",
				},
				"key3": MappedChartValue{
					strValue: "should not be returned",
				},
			},
			expect: map[string]interface{}{
				"key":  "{{repl ConfigOption 'test'}}",
				"key2": "repl{{ ConfigOption 'test'}}",
			},
		}, {
			name: "nested",
			values: map[string]MappedChartValue{
				"key": MappedChartValue{
					children: map[string]*MappedChartValue{
						"childKey": &MappedChartValue{
							strValue: "repl{{ConfigOption 'test'}}",
						},
						"childKey2": &MappedChartValue{
							strValue: "should not be returned",
						},
					},
				},
				"key2": MappedChartValue{
					strValue: "repl{{ ConfigOption 'test'}}",
				},
				"key3": MappedChartValue{
					strValue: "should not be returned",
				},
			},
			expect: map[string]interface{}{
				"key": MappedChartValue{
					children: map[string]*MappedChartValue{
						"childKey": &MappedChartValue{
							strValue: "repl{{ConfigOption 'test'}}",
						},
					},
				},
				"key2": "repl{{ ConfigOption 'test'}}",
			},
		}, {
			name: "complex nested",
			values: map[string]MappedChartValue{
				"rootKey": MappedChartValue{
					children: map[string]*MappedChartValue{
						"level2Key": &MappedChartValue{
							children: map[string]*MappedChartValue{
								"level3Key": &MappedChartValue{
									strValue: "should not be returned",
								},
							},
						},
						"level2Key2": &MappedChartValue{
							children: map[string]*MappedChartValue{
								"level3Key2": &MappedChartValue{
									strValue: "repl{{ConfigOption 'test'}}",
								},
							},
						},
					},
				},
				"key2": MappedChartValue{
					strValue: "repl{{ ConfigOption 'test'}}",
				},
				"key3": MappedChartValue{
					strValue: "should not be returned",
				},
			},
			expect: map[string]interface{}{
				"rootKey": MappedChartValue{
					children: map[string]*MappedChartValue{
						"level2Key2": &MappedChartValue{
							children: map[string]*MappedChartValue{
								"level3Key2": &MappedChartValue{
									strValue: "repl{{ConfigOption 'test'}}",
								},
							},
						},
					},
				},
				"key2": "repl{{ ConfigOption 'test'}}",
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			req := require.New(t)
			actual, err := test.spec.GetReplTmplValues(test.values)
			req.NoError(err)
			diff := deep.Equal(&actual, &test.expect)
			if len(diff) != 0 {
				fmt.Printf("Failed diff compare with %s", strings.Join(diff, "\n"))
				assert.NotEqual(t, test.expect, actual)
			}
		})
	}
}
