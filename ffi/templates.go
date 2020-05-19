package main

import "C"

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/mholt/archiver"
	"github.com/pkg/errors"
	kotsv1beta1 "github.com/replicatedhq/kots/kotskinds/apis/kots/v1beta1"
	kotsconfig "github.com/replicatedhq/kots/pkg/config"
	"github.com/replicatedhq/kots/pkg/crypto"
	"github.com/replicatedhq/kots/pkg/template"
	"k8s.io/client-go/kubernetes/scheme"
)

//export RenderFile
func RenderFile(socket string, filePath string, archivePath string, registryJson string) {
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

		registryInfo := struct {
			Host      string `json:"registryHostname"`
			Username  string `json:"registryUsername"`
			Password  string `json:"registryPassword"`
			Namespace string `json:"namespace"`
		}{}
		if err := json.Unmarshal([]byte(registryJson), &registryInfo); err != nil {
			fmt.Printf("failed to unmarshal registry info: %s\n", err.Error())
			ffiResult = NewFFIResult(-1).WithError(err)
			return
		}

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

		// look for config
		config, values, license, installation, err := findConfig(tmpRoot)
		if err != nil {
			ffiResult = NewFFIResult(-1).WithError(err)
			return
		}

		var cipher *crypto.AESCipher
		if installation != nil {
			c, err := crypto.AESCipherFromString(installation.Spec.EncryptionKey)
			if err != nil {
				ffiResult = NewFFIResult(-1).WithError(err)
				return
			}
			cipher = c
		}

		templateContextValues := make(map[string]template.ItemValue)
		if values != nil {
			for k, v := range values.Spec.Values {
				templateContextValues[k] = template.ItemValue{
					Value:   v.Value,
					Default: v.Default,
				}
			}
		}

		configGroups := []kotsv1beta1.ConfigGroup{}
		if config != nil {
			configGroups = config.Spec.Groups
		}

		localRegistry := template.LocalRegistry{
			Host:      registryInfo.Host,
			Namespace: registryInfo.Namespace,
			Username:  registryInfo.Username,
			Password:  registryInfo.Password,
		}

		builder, configVals, err := template.NewBuilder(configGroups, templateContextValues, localRegistry, cipher, license, nil)
		if err != nil {
			fmt.Printf("failed to create config context %s\n", err.Error())
			ffiResult = NewFFIResult(1).WithError(err)
			return
		}

		if config != nil {
			kotsconfig.ApplyValuesToConfig(config, configVals)
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

func findConfig(archivePath string) (*kotsv1beta1.Config, *kotsv1beta1.ConfigValues, *kotsv1beta1.License, *kotsv1beta1.Installation, error) {
	var config *kotsv1beta1.Config
	var values *kotsv1beta1.ConfigValues
	var license *kotsv1beta1.License
	var installation *kotsv1beta1.Installation

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
			} else if gvk.Group == "kots.io" && gvk.Version == "v1beta1" && gvk.Kind == "Installation" {
				installation = obj.(*kotsv1beta1.Installation)
			}

			return nil
		})

	if err != nil {
		return nil, nil, nil, nil, errors.Wrap(err, "failed to walk archive dir")
	}

	return config, values, license, installation, nil
}
