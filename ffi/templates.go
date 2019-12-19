package main

import "C"

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/mholt/archiver"
	"github.com/pkg/errors"
	kotsv1beta1 "github.com/replicatedhq/kots/kotskinds/apis/kots/v1beta1"
	kotsconfig "github.com/replicatedhq/kots/pkg/config"
	"github.com/replicatedhq/kots/pkg/template"
	"k8s.io/client-go/kubernetes/scheme"
)

//export RenderFile
func RenderFile(socket string, filePath string, archivePath string) {
	go func() {
		var ffiResult *FFIResult

		statusClient, err := connectToStatusServer(socket)
		if err != nil {
			fmt.Printf("failed to connect to status server: %s\n", err)
			return
		}
		defer func() {
			statusClient.end(ffiResult)
		}()

		tmpRoot, err := ioutil.TempDir("", "kots")
		if err != nil {
			fmt.Printf("failed to create temp root path: %s\n", err.Error())
			ffiResult = NewFFIResult(1).WithError(err)
			return
		}
		defer os.RemoveAll(tmpRoot)

		tarGz := archiver.TarGz{
			Tar: &archiver.Tar{
				ImplicitTopLevelFolder: false,
			},
		}
		if err := tarGz.Unarchive(archivePath, tmpRoot); err != nil {
			fmt.Printf("failed to unarchive %s\n", err.Error())
			ffiResult = NewFFIResult(1).WithError(err)
			return
		}

		builder := template.Builder{}
		builder.AddCtx(template.StaticCtx{})

		// look for config
		config, values, license, err := findConfig(tmpRoot)
		if err != nil {
			ffiResult = NewFFIResult(-1).WithError(err)
			return
		}
		if config != nil {
			templateContextValues := make(map[string]template.ItemValue)

			if values != nil {
				for k, v := range values.Spec.Values {
					templateContextValues[k] = template.ItemValue{
						Value:   v.Value,
						Default: v.Default,
					}
				}
			}

			configCtx, err := builder.NewConfigContext(config.Spec.Groups, templateContextValues)
			if err != nil {
				fmt.Printf("failed to create config context %s\n", err.Error())
				ffiResult = NewFFIResult(1).WithError(err)
				return
			}

			builder.AddCtx(configCtx)

			kotsconfig.ApplyValuesToConfig(config, configCtx.ItemValues)
		}

		if license != nil {
			licenseCtx := template.LicenseCtx{
				License: license,
			}
			builder.AddCtx(licenseCtx)
		}

		inputContent, err := ioutil.ReadFile(filePath)
		if err != nil {
			fmt.Printf("failed to read file %s\n", err.Error())
			ffiResult = NewFFIResult(1).WithError(err)
			return
		}
		rendered, err := builder.RenderTemplate(filePath, string(inputContent))
		if err != nil {
			fmt.Printf("failed to render template %s\n", err.Error())
			ffiResult = NewFFIResult(1).WithError(err)
			return
		}

		ioutil.WriteFile(filePath, []byte(rendered), 0644)
		ffiResult = NewFFIResult(0)
	}()
}

func findConfig(archivePath string) (*kotsv1beta1.Config, *kotsv1beta1.ConfigValues, *kotsv1beta1.License, error) {
	var config *kotsv1beta1.Config
	var values *kotsv1beta1.ConfigValues
	var license *kotsv1beta1.License

	err := filepath.Walk(archivePath,
		func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}

			if info.IsDir() {
				return nil
			}

			content, err := ioutil.ReadFile(path)
			if err != nil {
				return err
			}

			decode := scheme.Codecs.UniversalDeserializer().Decode
			obj, gvk, err := decode(content, nil, nil)
			if err != nil {
				return nil
			}

			if gvk.Group == "kots.io" && gvk.Version == "v1beta1" && gvk.Kind == "Config" {
				config = obj.(*kotsv1beta1.Config)
			} else if gvk.Group == "kots.io" && gvk.Version == "v1beta1" && gvk.Kind == "ConfigValues" {
				values = obj.(*kotsv1beta1.ConfigValues)
			} else if gvk.Group == "kots.io" && gvk.Version == "v1beta1" && gvk.Kind == "License" {
				license = obj.(*kotsv1beta1.License)
			}

			return nil
		})

	if err != nil {
		return nil, nil, nil, errors.Wrap(err, "failed to walk archive dir")
	}

	return config, values, license, nil
}
