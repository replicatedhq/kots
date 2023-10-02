package util

import (
	"reflect"
	"testing"
)

func Test_filesToTGZAndTGZToFiles(t *testing.T) {
	tests := []struct {
		name  string
		files map[string]string
	}{
		{
			name: "SingleFile",
			files: map[string]string{
				"file.txt": "File content",
			},
		},
		{
			name: "MultipleFiles",
			files: map[string]string{
				"file1.txt": "File 1 content",
				"file2.txt": "File 2 content",
			},
		},
		{
			name:  "EmptyFiles",
			files: map[string]string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tgzBytes, err := FilesToTGZ(tt.files)
			if err != nil {
				t.Errorf("FilesToTGZ() error = %v", err)
				return
			}

			actualFiles, err := TGZToFiles(tgzBytes)
			if err != nil {
				t.Errorf("TGZToFiles() error = %v", err)
				return
			}

			if !reflect.DeepEqual(actualFiles, tt.files) {
				t.Errorf("filesToTGZAndTGZToFiles() = %v, want %v", actualFiles, tt.files)
			}
		})
	}
}
