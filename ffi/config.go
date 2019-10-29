package main

import "C"

import (
	"fmt"

	kotsv1beta1 "github.com/replicatedhq/kots/kotskinds/apis/kots/v1beta1"
	kotsscheme "github.com/replicatedhq/kots/kotskinds/client/kotsclientset/scheme"
	"github.com/replicatedhq/kots/pkg/base"
	"github.com/replicatedhq/kots/pkg/template"
	"k8s.io/client-go/kubernetes/scheme"
)

//export TemplateConfig
func TemplateConfig(configPath string, configData string, configValuesData string) *C.char {
	kotsscheme.AddToScheme(scheme.Scheme)
	decode := scheme.Codecs.UniversalDeserializer().Decode
	obj, _, err := decode([]byte(configData), nil, nil)
	if err != nil {
		fmt.Printf("failed to decode config data: %s\n", err.Error())
		return C.CString("")
	}
	config := obj.(*kotsv1beta1.Config)

	builder := template.Builder{}
	builder.AddCtx(template.StaticCtx{})

	// get template context from config values
	var templateContext map[string]interface{}
	ctx, err := base.UnmarshalConfigValuesContent([]byte(configValuesData))
	if err != nil {
		fmt.Printf("failed to unmarshal config values content: %#v\n", err)
		return C.CString("")
	}
	templateContext = ctx

	// add config context
	configCtx, err := builder.NewConfigContext(config.Spec.Groups, templateContext)
	if err != nil {
		fmt.Printf("failed to create config context: %#v\n", err)
		return C.CString("")
	}
	builder.AddCtx(configCtx)

	rendered, err := builder.RenderTemplate(configPath, configData)
	if err != nil {
		fmt.Printf("failed to render template: %#v\n", err)
		return C.CString("")
	}

	return C.CString(rendered)
}
