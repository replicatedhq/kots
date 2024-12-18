package types

import (
	"slices"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRollupStatus(t *testing.T) {
	tests := []struct {
		backupStatuses []BackupStatus
		expected       BackupStatus
	}{
		{
			backupStatuses: []BackupStatus{
				BackupStatusInProgress,
				BackupStatusInProgress,
			},
			expected: BackupStatusInProgress,
		},
		{
			backupStatuses: []BackupStatus{
				BackupStatusInProgress,
				BackupStatusFailed,
			},
			expected: BackupStatusInProgress,
		},
		{
			backupStatuses: []BackupStatus{
				BackupStatusInProgress,
				BackupStatusCompleted,
			},
			expected: BackupStatusInProgress,
		},
		{
			backupStatuses: []BackupStatus{
				BackupStatusInProgress,
				BackupStatusDeleting,
			},
			expected: BackupStatusInProgress,
		},
		{
			backupStatuses: []BackupStatus{
				BackupStatusDeleting,
				BackupStatusDeleting,
			},
			expected: BackupStatusDeleting,
		},
		{
			backupStatuses: []BackupStatus{
				BackupStatusDeleting,
				BackupStatusFailed,
			},
			expected: BackupStatusDeleting,
		},
		{
			backupStatuses: []BackupStatus{
				BackupStatusDeleting,
				BackupStatusCompleted,
			},
			expected: BackupStatusDeleting,
		},
		{
			backupStatuses: []BackupStatus{
				BackupStatusFailed,
				BackupStatusFailed,
			},
			expected: BackupStatusFailed,
		},
		{
			backupStatuses: []BackupStatus{
				BackupStatusFailed,
				BackupStatusCompleted,
			},
			expected: BackupStatusFailed,
		},
		{
			backupStatuses: []BackupStatus{
				BackupStatusCompleted,
				BackupStatusCompleted,
			},
			expected: BackupStatusCompleted,
		},
	}

	for _, test := range tests {
		name := ""
		for _, status := range test.backupStatuses {
			name += string(status) + "-"
		}
		name = strings.TrimSuffix(name, "-")
		t.Run(name, func(t *testing.T) {
			assert.Equal(t, test.expected, RollupStatus(test.backupStatuses))
			// Reverse the order of the statuses and check if the result is the same
			slices.Reverse(test.backupStatuses)
			assert.Equal(t, test.expected, RollupStatus(test.backupStatuses))
		})
	}
}
