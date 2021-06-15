package base

import (
	"reflect"
	"testing"
)

func Test_mergeBaseFiles(t *testing.T) {
	type args struct {
		baseFiles []BaseFile
	}
	tests := []struct {
		name string
		args args
		want []BaseFile
	}{
		{
			name: "basic",
			args: args{
				baseFiles: []BaseFile{
					{Path: "a", Content: []byte("aa")},
					{Path: "b", Content: []byte("ba")},
					{Path: "a", Content: []byte("ab")},
					{Path: "c", Content: []byte("ca")},
					{Path: "f", Content: []byte("fa")},
					{Path: "e", Content: []byte("ea")},
					{Path: "f", Content: []byte("fb")},
					{Path: "d", Content: []byte("da")},
				},
			},
			want: []BaseFile{
				{Path: "a", Content: []byte("aa\n---\nab")},
				{Path: "b", Content: []byte("ba")},
				{Path: "c", Content: []byte("ca")},
				{Path: "f", Content: []byte("fa\n---\nfb")},
				{Path: "e", Content: []byte("ea")},
				{Path: "d", Content: []byte("da")},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := mergeBaseFiles(tt.args.baseFiles); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("mergeBaseFiles() = %v, want %v", got, tt.want)
			}
		})
	}
}
