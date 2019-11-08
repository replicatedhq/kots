package main

import "C"

import (
	"fmt"

	"github.com/replicatedhq/kots/pkg/config"
	"github.com/replicatedhq/kots/pkg/logger"
)

//export TemplateConfig
func TemplateConfig(configPath string, configData string, configValuesData string) *C.char {
	rendered, err := config.TemplateConfig(logger.NewLogger(), configPath, configData, configValuesData)
	if err != nil {
		fmt.Printf("failed to apply templates to config: %s\n", err.Error())
		return C.CString("")
	}
	return C.CString(rendered)
}
