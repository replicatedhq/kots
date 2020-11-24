package policy

import (
	"bytes"
	"net/http"
	"text/template"

	"github.com/gorilla/mux"
	"github.com/pkg/errors"
	"github.com/replicatedhq/kots/kotsadm/pkg/handlers"
	"github.com/replicatedhq/kots/kotsadm/pkg/logger"
)

type Policy struct {
	resource         string
	resourceTemplate *template.Template
}

func NewPolicy(resource string) (policy *Policy, err error) {
	policy = &Policy{resource: resource}
	policy.resourceTemplate, err = template.New(resource).Parse(resource)
	return
}

func Must(p *Policy, err error) *Policy {
	if err != nil {
		panic(err)
	}
	return p
}

func (p *Policy) Enforce(handler http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		resource, err := p.execute(r)
		if err != nil {
			logger.Error(errors.Wrapf(err, "failed to execute policy template %q", p.resource))
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		if err := handlers.CheckAccessOrAbort(w, r, resource); err != nil {
			logger.Error(err)
			return
		}
		handler(w, r)
	}
}

func (p *Policy) execute(r *http.Request) (string, error) {
	var buf bytes.Buffer
	err := p.resourceTemplate.Execute(&buf, mux.Vars(r))
	return buf.String(), err
}
