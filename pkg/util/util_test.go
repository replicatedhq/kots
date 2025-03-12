package util

import (
	"reflect"
	"testing"

	kotsv1beta1 "github.com/replicatedhq/kotskinds/apis/kots/v1beta1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_CommonSlicePrefix(t *testing.T) {
	tests := []struct {
		name     string
		first    []string
		second   []string
		expected []string
	}{
		{
			name:     "no common",
			first:    []string{"a", "b"},
			second:   []string{"1", "2"},
			expected: []string{},
		},
		{
			name:     "partial",
			first:    []string{"1", "2", "3"},
			second:   []string{"1", "a", "b"},
			expected: []string{"1"},
		},
		{
			name:     "exact",
			first:    []string{"l", "m", "n", "o"},
			second:   []string{"l", "m", "n", "o"},
			expected: []string{"l", "m", "n", "o"},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			common := CommonSlicePrefix(test.first, test.second)
			assert.Equal(t, test.expected, common)
		})
	}
}

func Test_SplitStringOnLen(t *testing.T) {
	tests := []struct {
		name     string
		in       string
		max      int
		expected []string
	}{
		{
			name:     "single part",
			in:       "this is a test",
			max:      1000,
			expected: []string{"this is a test"},
		},
		{
			name:     "even parts",
			in:       "fourfivenine",
			max:      4,
			expected: []string{"four", "five", "nine"},
		},
		{
			name:     "too big",
			in:       "one two six",
			max:      7,
			expected: []string{"one two", " six"},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			req := require.New(t)

			parts, err := SplitStringOnLen(test.in, test.max)
			req.NoError(err)

			assert.Equal(t, test.expected, parts)
		})
	}
}

func TestIntPointer(t *testing.T) {
	tests := []struct {
		name string
		x    int
		want int64
	}{
		{
			name: "zero",
			x:    0,
			want: int64(0),
		},
		{
			name: "positive",
			x:    100,
			want: int64(100),
		},
		{
			name: "negative",
			x:    -128,
			want: int64(-128),
		},
		{
			name: "int max",
			x:    1<<31 - 1,
			want: int64(1<<31 - 1),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := require.New(t)
			got := IntPointer(tt.x)
			req.Equal(tt.want, *got)
		})
	}
}

func TestGenPassword(t *testing.T) {
	tests := []struct {
		name   string
		length int
	}{
		{
			name:   "8",
			length: 8,
		},
		{
			name:   "32",
			length: 32,
		},
		{
			name:   "0",
			length: 0,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := require.New(t)
			got := GenPassword(tt.length)
			req.Len(got, tt.length)
		})
	}
}

func TestCompareStringArrays(t *testing.T) {
	tests := []struct {
		name string
		arr1 []string
		arr2 []string
		want bool
	}{
		{
			name: "empty arrays",
			arr1: []string{},
			arr2: []string{},
			want: true,
		},
		{
			name: "one empty array",
			arr1: []string{},
			arr2: []string{"element"},
			want: false,
		},
		{
			name: "superset",
			arr1: []string{"different element", "element"},
			arr2: []string{"element"},
			want: false,
		},
		{
			name: "duplicates",
			arr1: []string{"different element", "element"},
			arr2: []string{"element", "element", "different element"},
			want: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := require.New(t)
			req.Equal(CompareStringArrays(tt.arr1, tt.arr2), tt.want)
		})
	}
}

func TestGetValueFromMapPath(t *testing.T) {
	tests := []struct {
		name   string
		object interface{}
		path   []string
		want   interface{}
	}{
		{
			name: "empty path",
			object: map[string]interface{}{
				"key": "val",
			},
			path: []string{},
			want: nil,
		},
		{
			name:   "not a map",
			object: 5,
			path:   []string{"key1", "key2"},
			want:   nil,
		},
		{
			name: "valid path",
			object: map[string]interface{}{
				"key1": map[interface{}]interface{}{
					"key2": map[string]interface{}{
						"key3": "abc",
					},
				},
			},
			path: []string{"key1", "key2", "key3"},
			want: "abc",
		},
		{
			name: "invalid path",
			object: map[string]interface{}{
				"key1": map[interface{}]interface{}{
					"key2": map[string]interface{}{
						"key3": "abc",
					},
				},
			},
			path: []string{"key1", "key2", "key4"},
			want: nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := require.New(t)
			got := GetValueFromMapPath(tt.object, tt.path)
			req.Equal(tt.want, got)
		})
	}
}

func TestBase64DecodeInterface(t *testing.T) {
	tests := []struct {
		name    string
		encoded interface{}
		want    []byte
		isError bool
	}{
		{
			name:    "test string",
			encoded: "YWJj", // "abc"
			want:    []byte("abc"),
			isError: false,
		},
		{
			name:    "test bytes",
			encoded: []byte("eHl6"), // "xyz"
			want:    []byte("xyz"),
			isError: false,
		},
		{
			name:    "test invalid",
			encoded: 5,
			want:    nil,
			isError: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := require.New(t)
			got, err := Base64DecodeInterface(tt.encoded)
			if tt.isError {
				req.Error(err)
			} else {
				req.NoError(err)
			}
			req.Equal(tt.want, got)
		})
	}
}

func TestConvertToSingleDocs(t *testing.T) {
	tests := []struct {
		name string
		doc  []byte
		want [][]byte
	}{
		{
			name: "empty doc",
			doc:  []byte(""),
			want: [][]byte{},
		},
		{
			name: "single doc",
			doc:  []byte("abc"),
			want: [][]byte{[]byte("abc")},
		},
		{
			name: "multiple docs",
			doc:  []byte("abc\n---\ndef"),
			want: [][]byte{[]byte("abc"), []byte("def")},
		},
		{
			name: "multiple docs with empty",
			doc:  []byte("abc\n---\n\n---\ndef"),
			want: [][]byte{[]byte("abc"), []byte("def")},
		},
		{
			name: "multiple docs with windows line endings",
			doc:  []byte("abc\r\n---\r\ndef"),
			want: [][]byte{[]byte("abc"), []byte("def")},
		},
		{
			name: "multiple docs with empty and windows line endings",
			doc:  []byte("abc\r\n---\r\n\r\n---\r\ndef"),
			want: [][]byte{[]byte("abc"), []byte("def")},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ConvertToSingleDocs(tt.doc); !reflect.DeepEqual(got, tt.want) {
				for i, doc := range got {
					t.Logf("got[%d]:\n%s", i, string(doc))
				}
				for i, doc := range tt.want {
					t.Logf("want[%d]:\n%s", i, string(doc))
				}
				t.Fatal("ConvertToSingleDocs() mismatch")
			}
		})
	}
}

func Test_ReplicatedAPIEndpoint(t *testing.T) {
	tests := []struct {
		name        string
		license     *kotsv1beta1.License
		isEmbedded  bool
		envEndpoint string
		want        string
		wantError   bool
	}{
		{
			name: "license with endpoint",
			license: &kotsv1beta1.License{
				Spec: kotsv1beta1.LicenseSpec{
					Endpoint: "https://replicated.app",
				},
			},
			isEmbedded: false,
			want:       "https://replicated.app",
		},
		{
			name: "license with endpoint including port",
			license: &kotsv1beta1.License{
				Spec: kotsv1beta1.LicenseSpec{
					Endpoint: "https://replicated.app:8443",
				},
			},
			isEmbedded: false,
			want:       "https://replicated.app:8443",
		},
		{
			name:       "no license uses default endpoint",
			license:    nil,
			isEmbedded: false,
			want:       "https://replicated.app",
		},
		{
			name:        "embedded cluster with env endpoint",
			license:     nil,
			isEmbedded:  true,
			envEndpoint: "https://replicated.app",
			want:        "https://replicated.app",
		},
		{
			name:       "embedded cluster without env endpoint",
			license:    nil,
			isEmbedded: true,
			wantError:  true,
		},
		{
			name: "embedded cluster without env endpoint but with license should error",
			license: &kotsv1beta1.License{
				Spec: kotsv1beta1.LicenseSpec{
					Endpoint: "https://replicated.app:8443",
				},
			},
			isEmbedded: true,
			wantError:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := require.New(t)

			// Setup environment
			if tt.isEmbedded {
				t.Setenv("EMBEDDED_CLUSTER_ID", "123")

				if tt.envEndpoint != "" {
					t.Setenv("REPLICATED_APP_ENDPOINT", tt.envEndpoint)
				}
			}

			result, err := ReplicatedAPIEndpoint(tt.license)

			if tt.wantError {
				req.Error(err)
				return
			}

			req.NoError(err)
			assert.Equal(t, tt.want, result)
		})
	}
}
