package midstream

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"go.undefinedlabs.com/scopeagent"
)

func Test_findNewStrings(t *testing.T) {
	tests := []struct {
		existingList []string
		newList      []string
		expected     []string
	}{
		{
			existingList: []string{},
			newList:      []string{"abc", "xyz"},
			expected:     []string{"abc", "xyz"},
		},
		{
			existingList: []string{"abc", "xyz"},
			newList:      []string{},
			expected:     []string{},
		},
		{
			existingList: []string{"abc", "xyz", "ghi"},
			newList:      []string{"abc", "def", "xyz"},
			expected:     []string{"def"},
		},
	}

	for _, test := range tests {
		scopetest := scopeagent.StartTest(t)
		defer scopetest.End()
		diff := findNewStrings(test.newList, test.existingList)
		assert.Equal(t, test.expected, diff)
	}
}
