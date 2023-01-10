package types

import (
	"testing"

	"gopkg.in/go-playground/assert.v1"
)

func Test_getTroubleshootLabels(t *testing.T) {
	tests := []struct {
		name           string
		additionLabels map[string]string
		expectLabels   map[string]string
	}{
		{
			name:           "pass case with default troubleshoot labels",
			additionLabels: nil,
			expectLabels: map[string]string{
				"kots.io/kotsadm":      "true",
				"kots.io/backup":       "velero",
				"troubleshoot.io/kind": "support-bundle",
			},
		},
		{
			name: "pass case with extra troubleshoot labels",
			additionLabels: map[string]string{
				"foo": "bar",
			},
			expectLabels: map[string]string{
				"foo":                  "bar",
				"kots.io/kotsadm":      "true",
				"kots.io/backup":       "velero",
				"troubleshoot.io/kind": "support-bundle",
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			labels := GetTroubleshootLabels(test.additionLabels)
			assert.Equal(t, test.expectLabels, labels)
		})
	}
}
