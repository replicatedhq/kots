package main

import (
	"bufio"
	"bytes"
	"fmt"
	"go/format"
	"io/ioutil"
	"log"
	"os"
	"strings"
	"text/template"
)

const (
	minioReference                    = "minio"
	dexReference                      = "dex"
	postgresAlpineReference           = "postgres-alpine"
	postgresDebianReference           = "postgres-debian"
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

type generationContext struct {
	inputFilename          string
	outputConstantFilename string
	outputEnvFilename      string
	tagFinderFn            tagFinderFn
}

func main() {
	log.Println("started tagged image file generator")
	ctx := generationContext{
		inputFilename:          inputFilename,
		outputConstantFilename: outputConstantFilename,
		outputEnvFilename:      outputEnvFilename,
		tagFinderFn:            getTagFinder(),
	}
	if err := generateTaggedImageFiles(ctx); err != nil {
		log.Fatalf("generator failed: %s", err)
	}
	log.Printf("successfully generated constant file %q", outputConstantFilename)
	log.Printf("successfully generated dot env file %q", outputEnvFilename)
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

	return nil
}

type templatePostProcessorFn func(buff []byte)([]byte, error)

func goFmt(buff []byte)([]byte, error){
	return format.Source(buff)
}

func noopPostProcessor(buff []byte)([]byte, error) {
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

	if err := ioutil.WriteFile(filename, buff, 0644); err != nil {
		return err
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
	return strings.ToUpper(strings.ReplaceAll(s, "-", "_"))
}
