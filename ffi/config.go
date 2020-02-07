package main

import "C"

import (
	"fmt"

	"github.com/replicatedhq/kots/pkg/config"
	"github.com/replicatedhq/kots/pkg/logger"
)

//export TemplateConfig
func TemplateConfig(configSpecData string, configValuesData string) *C.char {
	rendered, err := config.TemplateConfig(logger.NewLogger(), configSpecData, configValuesData)
	if err != nil {
		fmt.Printf("failed to apply templates to config: %s\n", err.Error())
		return C.CString("")
	}
	return C.CString(rendered)
}
