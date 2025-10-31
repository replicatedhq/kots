package schema

import (
	"path/filepath"
	"runtime"
)

const (
	// KubernetesLintVersion is the Kubernetes version to use for linting
	KubernetesLintVersion = "1.33.3"
)

var (
	// KubernetesJsonSchemaDir is the directory containing the Kubernetes JSON schemas
	KubernetesJsonSchemaDir string
)

func init() {
	// Get the directory of this source file
	_, filename, _, _ := runtime.Caller(0)
	dir := filepath.Dir(filename)

	// The schema files are in the same directory as this file
	KubernetesJsonSchemaDir = filepath.Join(dir, "schema")
}

// GetSchemaDir returns the path to the schema directory
func GetSchemaDir() string {
	return KubernetesJsonSchemaDir
}
