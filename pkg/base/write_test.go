package base

import (
	"encoding/json"
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var (
	TestServiceA = `apiVersion: v1
kind: Service
metadata:
  name: service-a`

	TestServiceAnsB = `apiVersion: v1
kind: Service
metadata:
  name: service-a
  namespace: b`

	TestServiceAnsC = `apiVersion: v1
kind: Service
metadata:
  name: service-a
  namespace: c`

	TestServiceAnsTest = `apiVersion: v1
kind: Service
metadata:
  name: service-a
  namespace: test`

	TestServiceB = `apiVersion: v1
kind: Service
metadata:
  name: service-b`

	TestPodA = `apiVersion: v1
kind: Pod
metadata:
  name: pod-a`

	TestPodNamedServiceA = `apiVersion: v1
kind: Pod
metadata:
  name: service-a`
)

func Test_DeduplicateOnContent(t *testing.T) {
	tests := []struct {
		name              string
		files             []BaseFile
		excludeKotsKinds  bool
		expectedResources []BaseFile
		expectedPatches   []BaseFile
	}{
		{
			name: "all unique",
			files: []BaseFile{
				{
					Path:    "service-a",
					Content: []byte(TestServiceA),
				},
				{
					Path:    "service-b",
					Content: []byte(TestServiceB),
				},
			},
			excludeKotsKinds: true,
			expectedResources: []BaseFile{
				{
					Path:    "service-a",
					Content: []byte(TestServiceA),
				},
				{
					Path:    "service-b",
					Content: []byte(TestServiceB),
				},
			},
			expectedPatches: []BaseFile{},
		},
		{
			name: "duplicated service",
			files: []BaseFile{
				{
					Path:    "service-a",
					Content: []byte(TestServiceA),
				},
				{
					Path:    "service-b",
					Content: []byte(TestServiceB),
				},
				{
					Path:    "service-b",
					Content: []byte(TestServiceB),
				},
			},
			excludeKotsKinds: true,
			expectedResources: []BaseFile{
				{
					Path:    "service-a",
					Content: []byte(TestServiceA),
				},
				{
					Path:    "service-b",
					Content: []byte(TestServiceB),
				},
			},
			expectedPatches: []BaseFile{
				{
					Path:    "service-b",
					Content: []byte(TestServiceB),
				},
			},
		},
		{
			name: "same-name-different-gvk",
			files: []BaseFile{
				{
					Path:    "service-a",
					Content: []byte(TestServiceA),
				},
				{
					Path:    "service-a",
					Content: []byte(TestServiceB),
				},
				{
					Path:    "service-a",
					Content: []byte(TestPodNamedServiceA),
				},
			},
			excludeKotsKinds: true,
			expectedResources: []BaseFile{
				{
					Path:    "service-a",
					Content: []byte(TestServiceA),
				},
				{
					Path:    "service-a",
					Content: []byte(TestServiceB),
				},
				{
					Path:    "service-a",
					Content: []byte(TestPodNamedServiceA),
				},
			},
			expectedPatches: []BaseFile{},
		},
		{
			name: "same-name-specified-ns",
			files: []BaseFile{
				{
					Path:    "service-a",
					Content: []byte(TestServiceA),
				},
				{
					Path:    "service-a-ns-b",
					Content: []byte(TestServiceAnsB),
				},
				{
					Path:    "service-a-ns-c",
					Content: []byte(TestServiceAnsC),
				},
				{
					Path:    "service-a-ns-b-patch",
					Content: []byte(TestServiceAnsB),
				},
			},
			excludeKotsKinds: true,
			expectedResources: []BaseFile{
				{
					Path:    "service-a",
					Content: []byte(TestServiceA),
				},
				{
					Path:    "service-a-ns-b",
					Content: []byte(TestServiceAnsB),
				},
				{
					Path:    "service-a-ns-c",
					Content: []byte(TestServiceAnsC),
				},
			},
			expectedPatches: []BaseFile{
				{
					Path:    "service-a-ns-b-patch",
					Content: []byte(TestServiceAnsB),
				},
			},
		},
		{
			name: "base-ns",
			files: []BaseFile{
				{
					Path:    "service-a",
					Content: []byte(TestServiceA),
				},
				{
					Path:    "service-a-ns-test-patch",
					Content: []byte(TestServiceAnsTest),
				},
			},
			excludeKotsKinds: true,
			expectedResources: []BaseFile{
				{
					Path:    "service-a",
					Content: []byte(TestServiceA),
				},
			},
			expectedPatches: []BaseFile{
				{
					Path:    "service-a-ns-test-patch",
					Content: []byte(TestServiceAnsTest),
				},
			},
		},
		{
			name: "not yaml",
			files: []BaseFile{
				{
					Path:    "service-a",
					Content: []byte(TestServiceA),
				},
				{
					Path:    "not-yaml",
					Content: []byte("not yaml"),
				},
			},
			excludeKotsKinds: true,
			expectedResources: []BaseFile{
				{
					Path:    "service-a",
					Content: []byte(TestServiceA),
				},
			},
			expectedPatches: []BaseFile{},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			req := require.New(t)

			actualResources, actualPatches, err := deduplicateOnContent(test.files, test.excludeKotsKinds, "test")
			req.NoError(err)

			assert.ElementsMatch(t, test.expectedResources, actualResources)
			assert.ElementsMatch(t, test.expectedPatches, actualPatches)
		})
	}

}

func Test_convertToSingleDocBaseFiles(t *testing.T) {
	tests := []struct {
		name  string
		files []BaseFile
		want  []BaseFile
	}{
		{
			name: "basic",
			files: []BaseFile{
				{
					Path:    "/dir/multi.yaml",
					Content: []byte("---\na: b\n---\nc: d"),
				},
				{
					Path:    "/dir/single.yaml",
					Content: []byte("e: f"),
				},
				{
					Path:    "/dir/empty.yaml",
					Content: []byte(""),
				},
			},
			want: []BaseFile{
				{
					Path:    "/dir/multi.yaml",
					Content: []byte("---\na: b"),
				},
				{
					Path:    "/dir/multi-2.yaml",
					Content: []byte("c: d"),
				},
				{
					Path:    "/dir/single.yaml",
					Content: []byte("e: f"),
				},
				{
					Path:    "/dir/empty.yaml",
					Content: []byte(""),
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := convertToSingleDocBaseFiles(tt.files); !reflect.DeepEqual(got, tt.want) {
				gotB, _ := json.MarshalIndent(got, "", "  ")
				wantB, _ := json.MarshalIndent(tt.want, "", "  ")
				t.Errorf("convertToSingleDocBaseFiles() = %s, want %s", gotB, wantB)
			}
		})
	}
}
