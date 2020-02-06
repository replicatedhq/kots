package util

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
	"go.undefinedlabs.com/scopeagent"
)

func TestMarshalIndent(t *testing.T) {
	lineString := "this is a very very long line that would normally wrap after 80 characters with the default yaml.v3 encoder, but should not wrap here."
	tests := []struct {
		name   string
		indent int
		in     interface{}
		want   string
	}{
		{
			name:   "long line",
			indent: 2,
			in: struct {
				Line string
			}{
				Line: lineString,
			},
			want: fmt.Sprintf("line: %s\n", lineString),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			scopetest := scopeagent.StartTest(t)
			defer scopetest.End()
			req := require.New(t)

			got, err := MarshalIndent(tt.indent, tt.in)
			req.NoError(err)
			req.Equal(tt.want, string(got))
		})
	}
}
