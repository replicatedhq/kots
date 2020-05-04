package base

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.undefinedlabs.com/scopeagent"
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
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			scopetest := scopeagent.StartTest(t)
			defer scopetest.End()
			req := require.New(t)

			actualResources, actualPatches, err := deduplicateOnContent(test.files, test.excludeKotsKinds)
			req.NoError(err)

			assert.ElementsMatch(t, test.expectedResources, actualResources)
			assert.ElementsMatch(t, test.expectedPatches, actualPatches)
		})
	}

}
