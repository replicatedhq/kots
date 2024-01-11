package main

import (
	"bufio"
	"bytes"
	"fmt"
	"go/format"
	"log"
	"os"
	"regexp"
	"strings"
	"text/template"

	"github.com/pkg/errors"
)

const (
	schemaheroReference               = "schemahero"
	minioReference                    = "minio"
	dexReference                      = "dex"
	rqliteReference                   = "rqlite"
	lvpReference                      = "lvp"
	inputFilename                     = "cmd/imagedeps/image-spec"
	outputConstantFilename            = "pkg/image/constants.go"
	outputEnvFilename                 = ".image.env"
	dockerRegistryUrl                 = "https://index.docker.io"
	githubPageSize                    = 100
	githubAuthTokenEnvironmentVarName = "GITHUB_AUTH_TOKEN"
	constantFileTemplate              = `package image

// Generated file, do not modify.  This file is generated from a text file containing a list of images. The
// most recent tag is interpolated from the source repository and used to generate a fully qualified
// image name.

const (
{{- range .}}
{{.GetDeclarationLine}}
{{- end}}
)`
	environmentFileTemplate = `# Generated file, do not modify.  This file is generated from a text file containing a list of images. The
# most recent tag is interpolated from the source repository and used to generate a fully qualified image
# name.
{{- range .}}
{{.GetEnvironmentLine}}
{{- end}}`
)

type replacer struct {
	path    string
	regexFn func(ir *ImageRef) string
	valueFn func(ir *ImageRef) string
}

var (
	replacers = []*replacer{
		getMakefileReplacer("Makefile"),
		getMakefileReplacer("migrations/Makefile"),
	}
)

type generationContext struct {
	inputFilename          string
	outputConstantFilename string
	outputEnvFilename      string
	tagFinderFn            tagFinderFn
	replacers              []*replacer
}

func main() {
	log.Println("started tagged image file generator")
	ctx := generationContext{
		inputFilename:          inputFilename,
		outputConstantFilename: outputConstantFilename,
		outputEnvFilename:      outputEnvFilename,
		tagFinderFn:            getTagFinder(),
		replacers:              replacers,
	}
	if err := generateTaggedImageFiles(ctx); err != nil {
		log.Fatalf("generator failed: %s", err)
	}
	log.Printf("successfully generated constant file %q", outputConstantFilename)
	log.Printf("successfully generated dot env file %q", outputEnvFilename)
	for _, r := range ctx.replacers {
		log.Printf("successfully updated %q", r.path)
	}
}

func generateTaggedImageFiles(ctx generationContext) error {
	f, err := os.Open(ctx.inputFilename)
	if err != nil {
		return fmt.Errorf("could not read image file %q %w", ctx.inputFilename, err)
	}
	defer f.Close()

	var references []*ImageRef
	scan := bufio.NewScanner(f)
	for scan.Scan() {
		line := scan.Text()
		ref, err := ctx.tagFinderFn(line)
		if err != nil {
			return fmt.Errorf("could not process image line %q %w", line, err)
		}
		references = append(references, ref)
	}

	if len(references) == 0 {
		return fmt.Errorf("no references to images found")
	}

	if err := generateOutput(ctx.outputConstantFilename, constantFileTemplate, references, goFmt); err != nil {
		return fmt.Errorf("failed to generate output file %q %w", ctx.outputConstantFilename, err)
	}

	if err := generateOutput(ctx.outputEnvFilename, environmentFileTemplate, references, noopPostProcessor); err != nil {
		return fmt.Errorf("failed to generate file %q %w", ctx.outputEnvFilename, err)
	}

	for _, r := range ctx.replacers {
		if err := r.replace(references); err != nil {
			return fmt.Errorf("failed to replace in %q %w", r.path, err)
		}
	}

	return nil
}

type templatePostProcessorFn func(buff []byte) ([]byte, error)

func goFmt(buff []byte) ([]byte, error) {
	return format.Source(buff)
}

func noopPostProcessor(buff []byte) ([]byte, error) {
	return buff, nil
}

func generateOutput(filename, fileTemplate string, refs []*ImageRef, fn templatePostProcessorFn) error {
	var out bytes.Buffer
	if err := template.Must(template.New("constants").Parse(fileTemplate)).Execute(&out, refs); err != nil {
		return err
	}

	buff, err := fn(out.Bytes())
	if err != nil {
		return err
	}

	if err := os.WriteFile(filename, buff, 0644); err != nil {
		return err
	}

	return nil
}

func (r *replacer) replace(refs []*ImageRef) error {
	b, err := os.ReadFile(r.path)
	if err != nil {
		return errors.Wrap(err, "failed to read file")
	}
	content := string(b)

	for _, ref := range refs {
		reg, err := regexp.Compile(r.regexFn(ref))
		if err != nil {
			return errors.Wrap(err, "failed to compile regex")
		}
		content = reg.ReplaceAllString(content, r.valueFn(ref))
	}

	if err := os.WriteFile(r.path, []byte(content), 0644); err != nil {
		return errors.Wrap(err, "failed to write file")
	}

	return nil
}

// converts a name from the input file into a package public constant name
// for example: foo-bar-baz -> FooBarBaz
func getConstantName(s string) string {
	parts := strings.Split(s, "-")
	result := ""
	for _, part := range parts {
		result += strings.Title(part)
	}
	return result
}

// converts a name from the input file into an environment variable name
// for example: foo_bar_baz -> FOO_BAR_BAZ
func getEnvironmentName(s string) string {
	return strings.ToUpper(strings.ReplaceAll(s, "-", "_")) + "_TAG"
}

// converts a name from the input string into an a makefile variable name
// for example: foo_bar_baz -> FOO_BAR_BAZ
func getMakefileVarName(s string) string {
	return strings.ToUpper(strings.ReplaceAll(s, "-", "_")) + "_TAG"
}

// converts a name from the input string into an a dockerfile variable name
// for example: foo_bar_baz -> FOO_BAR_BAZ
func getDockerfileVarName(s string) string {
	return strings.ToUpper(strings.ReplaceAll(s, "-", "_")) + "_TAG"
}

func getMakefileReplacer(path string) *replacer {
	return &replacer{
		path: path,
		regexFn: func(ir *ImageRef) string {
			return fmt.Sprintf("%s [\\?:]= .*", getMakefileVarName(ir.name))
		},
		valueFn: func(ir *ImageRef) string {
			return ir.GetMakefileLine()
		},
	}
}

func getDockerfileReplacer(path string) *replacer {
	return &replacer{
		path: path,
		regexFn: func(ir *ImageRef) string {
			return fmt.Sprintf("ARG %s=.*", getDockerfileVarName(ir.name))
		},
		valueFn: func(ir *ImageRef) string {
			return ir.GetDockerfileLine()
		},
	}
}
