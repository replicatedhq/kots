package midstream

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_uniqueStrings(t *testing.T) {
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
			expected:     []string{"abc", "xyz"},
		},
		{
			existingList: []string{"abc", "xyz", "ghi"},
			newList:      []string{"abc", "def", "xyz"},
			expected:     []string{"abc", "xyz", "ghi", "def"},
		},
	}

	for _, test := range tests {
		uniq := uniqueStrings(test.existingList, test.newList)
		assert.Equal(t, test.expected, uniq)
	}
}
