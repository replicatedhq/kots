package template

import (
	"encoding/json"
	"fmt"
	"text/template"

	kurlv1beta1 "github.com/replicatedhq/kurl/kurlkinds/pkg/apis/cluster/v1beta1"
)

type installerCtx struct {
	Installer *kurlv1beta1.Installer
}

func newInstallerCtx(installer *kurlv1beta1.Installer) installerCtx {
	ctx := installerCtx{
		Installer: installer,
	}

	return ctx
}

// FuncMap represents the available functions in the installerCtx.
func (ctx installerCtx) FuncMap() template.FuncMap {
	return template.FuncMap{
		"ReleaseInstallerSpec": ctx.releaseInstallerSpec,
	}
}

func (ctx installerCtx) releaseInstallerSpec() string {
	// return "" for a nil installer
	if ctx.Installer == nil {
		return ""
	}

	fmt.Printf("marshalling installer spec:\n%+v\n", ctx.Installer.Spec)
	b, err := json.Marshal(ctx.Installer.Spec)
	if err != nil {
		fmt.Println("failed to marshal release installer spec")
		return ""
	}

	return string(b)
}
