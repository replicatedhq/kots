package base

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

var (
	TestServiceA = `apiVersion: v1
kind: Service
metadata:
  name: service-a`

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
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			actualResources, actualPatches := deduplicateOnContent(test.files, test.excludeKotsKinds)
			assert.ElementsMatch(t, test.expectedResources, actualResources)
			assert.ElementsMatch(t, test.expectedPatches, actualPatches)
		})
	}

}
