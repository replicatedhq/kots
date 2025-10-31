package types

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"encoding/base64"
	"fmt"
	"io"
	"os"
	"path"
	"strings"

	"github.com/pkg/errors"
	"github.com/replicatedhq/kots/pkg/lint/util"
)

// SpecFiles is a collection of SpecFile
type SpecFiles []SpecFile

// SpecFile represents a single file in a KOTS application
type SpecFile struct {
	Name            string    `json:"name"`
	Path            string    `json:"path"`
	Content         string    `json:"content"`
	DocIndex        int       `json:"docIndex,omitempty"`
	AllowDuplicates bool      `json:"allowDuplicates"` // kotskinds can be duplicated if they are coming from secrets or configmaps
	Children        SpecFiles `json:"children"`
}

// GVKDoc represents a Kubernetes resource with basic GVK info
type GVKDoc struct {
	Kind       string      `yaml:"kind" json:"kind" validate:"required"`
	APIVersion string      `yaml:"apiVersion" json:"apiVersion"`
	Metadata   GVKMetadata `yaml:"metadata" json:"metadata"`
}

// GVKMetadata represents basic metadata for a Kubernetes resource
type GVKMetadata struct {
	Name      string `yaml:"name" json:"name"`
	Namespace string `yaml:"namespace" json:"namespace"`
}

// LintExpression represents a single lint finding
type LintExpression struct {
	Rule      string                       `json:"rule"`
	Type      string                       `json:"type"` // "error", "warn", "info"
	Message   string                       `json:"message"`
	Path      string                       `json:"path"`
	Positions []LintExpressionItemPosition `json:"positions"`
}

// LintExpressionsByRule implements sort.Interface for []LintExpression based on Rule field
type LintExpressionsByRule []LintExpression

func (a LintExpressionsByRule) Len() int           { return len(a) }
func (a LintExpressionsByRule) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a LintExpressionsByRule) Less(i, j int) bool { return a[i].Rule < a[j].Rule }

// OPALintExpression represents a lint expression from OPA policy evaluation
type OPALintExpression struct {
	Rule     string `json:"rule"`
	Type     string `json:"type"`
	Message  string `json:"message"`
	Path     string `json:"path"`
	DocIndex int    `json:"docIndex"`
	Field    string `json:"field"`
	Match    string `json:"match"`
}

// LintExpressionItemPosition represents the position of a lint finding in a file
type LintExpressionItemPosition struct {
	Start LintExpressionItemLinePosition `json:"start"`
}

// LintExpressionItemLinePosition represents a line position
type LintExpressionItemLinePosition struct {
	Line int `json:"line"`
}

// LintResult represents the complete result of linting
type LintResult struct {
	LintExpressions []LintExpression `json:"lintExpressions"`
	IsComplete      bool             `json:"isLintingComplete"`
}

// HasErrors returns true if the result contains any errors
func (r *LintResult) HasErrors() bool {
	for _, expr := range r.LintExpressions {
		if expr.Type == "error" {
			return true
		}
	}
	return false
}

// HasWarnings returns true if the result contains any warnings
func (r *LintResult) HasWarnings() bool {
	for _, expr := range r.LintExpressions {
		if expr.Type == "warn" {
			return true
		}
	}
	return false
}

// ErrorCount returns the number of errors
func (r *LintResult) ErrorCount() int {
	count := 0
	for _, expr := range r.LintExpressions {
		if expr.Type == "error" {
			count++
		}
	}
	return count
}

// WarningCount returns the number of warnings
func (r *LintResult) WarningCount() int {
	count := 0
	for _, expr := range r.LintExpressions {
		if expr.Type == "warn" {
			count++
		}
	}
	return count
}

// IsTarGz returns true if the file is a tar.gz or .tgz file
func (f SpecFile) IsTarGz() bool {
	return strings.HasSuffix(f.Path, ".tgz") || strings.HasSuffix(f.Path, ".tar.gz")
}

// IsYAML returns true if the file is a YAML file
func (f SpecFile) IsYAML() bool {
	return strings.HasSuffix(f.Path, ".yaml") || strings.HasSuffix(f.Path, ".yml")
}

// hasContent returns true if the file has non-empty content
func (f SpecFile) hasContent() bool {
	return util.CleanUpYaml(f.Content) != ""
}

// Unnest flattens nested files (extracts children from archives)
func (fs SpecFiles) Unnest() SpecFiles {
	unnestedFiles := SpecFiles{}
	for _, file := range fs {
		if len(file.Children) > 0 {
			unnestedFiles = append(unnestedFiles, file.Children.Unnest()...)
		} else {
			unnestedFiles = append(unnestedFiles, file)
		}
	}
	return unnestedFiles
}

// GetFile returns a file by path
func (fs SpecFiles) GetFile(path string) (*SpecFile, error) {
	for _, file := range fs {
		if file.Path == path {
			return &file, nil
		}
	}
	return nil, fmt.Errorf("spec file not found for path %s", path)
}

// Separate splits multi-document YAML files into individual documents
func (fs SpecFiles) Separate() (SpecFiles, error) {
	separatedSpecFiles := SpecFiles{}

	for _, file := range fs {
		if !file.IsYAML() {
			separatedSpecFiles = append(separatedSpecFiles, file)
			continue
		}

		cleanedContent := util.CleanUpYaml(file.Content)
		docs := strings.Split(cleanedContent, "\n---\n")

		for index, doc := range docs {
			doc = strings.TrimPrefix(doc, "---")
			doc = strings.TrimLeft(doc, "\n")

			if len(doc) == 0 {
				continue
			}

			separatedSpecFile := SpecFile{
				Name:     file.Name,
				Path:     file.Path, // keep original path to be able to link it back
				Content:  doc,
				DocIndex: index,
				// Split files inherit the original file's AllowDuplicates.
				// This will only be set for KotsKinds extracted from Secrets and ConfigMaps, so this works.
				// But also there is no good way to allow split docs to have their own flag.
				AllowDuplicates: file.AllowDuplicates,
			}

			separatedSpecFiles = append(separatedSpecFiles, separatedSpecFile)
		}
	}

	return separatedSpecFiles, nil
}

// SpecFilesFromTar loads spec files from a tar archive
func SpecFilesFromTar(reader io.Reader) (SpecFiles, error) {
	specFiles := SpecFiles{}

	tr := tar.NewReader(reader)

	for {
		header, err := tr.Next()
		if err != nil {
			if err == io.EOF {
				break
			}
			return nil, err
		}

		if header.Typeflag != tar.TypeReg {
			continue
		}

		var data bytes.Buffer
		_, err = io.Copy(&data, tr)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to get data for %s", header.Name)
		}

		specFile := SpecFile{
			Name:    header.FileInfo().Name(),
			Path:    header.Name,
			Content: data.String(),
		}

		specFiles = append(specFiles, specFile)
	}

	return specFiles, nil
}

// SpecFilesFromTarGz loads spec files from a .tgz archive
func SpecFilesFromTarGz(tarGz SpecFile) (SpecFiles, error) {
	content, err := base64.StdEncoding.DecodeString(tarGz.Content)
	if err != nil {
		// tarGz content is not base64 encoded, read as bytes
		content = []byte(tarGz.Content)
	}

	gzf, err := gzip.NewReader(bytes.NewReader(content))
	if err != nil {
		return nil, errors.Wrap(err, "failed to create gzip reader")
	}

	files, err := SpecFilesFromTar(gzf)
	if err != nil {
		return nil, errors.Wrap(err, "failed to read chart archive")
	}

	// remove any common prefix from all files
	if len(files) > 0 {
		firstFileDir, _ := path.Split(files[0].Path)
		commonPrefix := strings.Split(firstFileDir, string(os.PathSeparator))

		for _, file := range files {
			d, _ := path.Split(file.Path)
			dirs := strings.Split(d, string(os.PathSeparator))
			commonPrefix = util.CommonSlicePrefix(commonPrefix, dirs)
		}

		cleanedFiles := SpecFiles{}
		for _, file := range files {
			d, f := path.Split(file.Path)
			d2 := strings.Split(d, string(os.PathSeparator))

			cleanedFile := file
			d2 = d2[len(commonPrefix):]
			cleanedFile.Path = path.Join(path.Join(d2...), f)

			cleanedFiles = append(cleanedFiles, cleanedFile)
		}

		files = cleanedFiles
	}

	return files, nil
}
