package main

import "C"

import (
	"fmt"

	"github.com/replicatedhq/kots/pkg/config"
	"github.com/replicatedhq/kots/pkg/logger"
	"github.com/replicatedhq/kots/pkg/template"
)

//export TemplateConfig
func TemplateConfig(configSpecData string, configValuesData string, licenseYaml string, registryHost string, registryNamespace string, registryUsername string, registryPassword string) *C.char {
	localRegistry := template.LocalRegistry{
		Host:      registryHost,
		Namespace: registryNamespace,
		Username:  registryUsername,
		Password:  registryPassword,
	}

	rendered, err := config.TemplateConfig(logger.NewLogger(), configSpecData, configValuesData, licenseYaml, localRegistry)
	if err != nil {
		fmt.Printf("failed to apply templates to config: %s\n", err.Error())
		return C.CString("")
	}
	return C.CString(rendered)
}
