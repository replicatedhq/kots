package main

import "C"

import (
	"encoding/json"
	"fmt"

	"github.com/replicatedhq/kots/pkg/config"
	"github.com/replicatedhq/kots/pkg/logger"
	"github.com/replicatedhq/kots/pkg/template"
)

//export TemplateConfig
func TemplateConfig(configSpecData string, configValuesData string, licenseYaml string, registryJson string) *C.char {
	registryInfo := struct {
		Host      string `json:"registryHostname"`
		Username  string `json:"registryUsername"`
		Password  string `json:"registryPassword"`
		Namespace string `json:"namespace"`
	}{}
	if err := json.Unmarshal([]byte(registryJson), &registryInfo); err != nil {
		fmt.Printf("failed to unmarshal registry info: %s\n", err.Error())
		return C.CString("")
	}

	localRegistry := template.LocalRegistry{
		Host:      registryInfo.Host,
		Namespace: registryInfo.Namespace,
		Username:  registryInfo.Username,
		Password:  registryInfo.Password,
	}

	rendered, err := config.TemplateConfig(logger.NewLogger(), configSpecData, configValuesData, licenseYaml, localRegistry)
	if err != nil {
		fmt.Printf("failed to apply templates to config: %s\n", err.Error())
		return C.CString("")
	}
	return C.CString(rendered)
}
