package main

import "C"

import (
	"fmt"

	"github.com/replicatedhq/kots/pkg/config"
)

//export TemplateConfig
func TemplateConfig(configPath string, configData string, configValuesData string) *C.char {
	rendered, err := config.TemplateConfig(configPath, configData, configValuesData)
	if err != nil {
		fmt.Printf("failed to decode config data: %s\n", err.Error())
		return C.CString("")
	}
	return C.CString(rendered)
}
