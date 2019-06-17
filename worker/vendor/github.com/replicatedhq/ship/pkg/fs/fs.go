package fs

import (
	"github.com/spf13/afero"
)

// NewBaseFilesystem creates a new Afero OS filesystem
func NewBaseFilesystem() afero.Afero {
	return afero.Afero{Fs: afero.NewOsFs()}
}
