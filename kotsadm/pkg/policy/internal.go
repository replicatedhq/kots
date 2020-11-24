package policy

import (
	"bytes"
	"net/http"
	"text/template"

	"github.com/gorilla/mux"
)

type Func func(r *http.Request) (string, error)

func executeRBACTemplate(r *http.Request, resource *template.Template) (string, error) {
	var buf bytes.Buffer
	err := resource.Execute(&buf, mux.Vars(r))
	return buf.String(), err
}
