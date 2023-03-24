package types

import (
	"testing"

	"gopkg.in/go-playground/assert.v1"
)

func Test_getKotsadmLabels(t *testing.T) {
	tests := []struct {
		name         string
		labels       []map[string]string
		expectLabels map[string]string
	}{
		{
			name: "pass case with additional labels",
			labels: []map[string]string{
				{
					"foo": "foo",
				},
			},
			expectLabels: map[string]string{
				"kots.io/kotsadm": "true",
				"kots.io/backup":  "velero",
				"foo":             "foo",
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			labels := GetKotsadmLabels(test.labels...)
			assert.Equal(t, test.expectLabels, labels)
		})
	}
}

func Test_getTroubleshootLabels(t *testing.T) {
	tests := []struct {
		name         string
		labels       []map[string]string
		expectLabels map[string]string
	}{
		{
			name: "pass case with additional labels",
			labels: []map[string]string{
				{
					"foo": "foo",
				},
			},
			expectLabels: map[string]string{
				"troubleshoot.sh/kind": "support-bundle",
				"foo":                  "foo",
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			labels := GetTroubleshootLabels(test.labels...)
			assert.Equal(t, test.expectLabels, labels)
		})
	}
}

func Test_mergeLabels(t *testing.T) {
	tests := []struct {
		name         string
		labels       []map[string]string
		expectLabels map[string]string
	}{
		{
			name: "pass case with merge labels",
			labels: []map[string]string{
				{
					"foo": "foo",
				},
				{
					"bar": "bar",
				},
				{
					"baz": "baz",
				},
			},
			expectLabels: map[string]string{
				"foo": "foo",
				"bar": "bar",
				"baz": "baz",
			},
		},
		{
			name: "pass case with merge troubleshoot and kotadm labels",
			labels: []map[string]string{
				GetKotsadmLabels(),
				GetTroubleshootLabels(),
			},
			expectLabels: map[string]string{
				"kots.io/kotsadm":      "true",
				"kots.io/backup":       "velero",
				"troubleshoot.sh/kind": "support-bundle",
			},
		},
		{
			name: "pass case with merge troubleshoot and kotadm with additional labels",
			labels: []map[string]string{
				GetKotsadmLabels(map[string]string{"foo": "foo"}),
				GetTroubleshootLabels(map[string]string{"bar": "bar"}),
			},
			expectLabels: map[string]string{
				"foo":                  "foo",
				"bar":                  "bar",
				"kots.io/kotsadm":      "true",
				"kots.io/backup":       "velero",
				"troubleshoot.sh/kind": "support-bundle",
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			labels := MergeLabels(test.labels...)
			assert.Equal(t, test.expectLabels, labels)
		})
	}
}
