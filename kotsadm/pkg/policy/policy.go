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

type VarsGetter func(vars map[string]string) (map[string]string, error)

type Policy struct {
	action           string
	resource         string
	resourceTemplate *template.Template
	varsGetterFns    []VarsGetter
}

func NewPolicy(action, resource string, fns ...VarsGetter) (policy *Policy, err error) {
	policy = &Policy{action: action, resource: resource, varsGetterFns: fns}
	policy.resourceTemplate, err = template.New(resource).Option("missingkey=error").Parse(resource)
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
		action, resource, appSlug, err := p.execute(r)
		if err != nil {
			logger.Error(errors.Wrapf(err, "failed to execute policy template %q", p.resource))
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		if err := handlers.CheckAccessOrAbort(w, r, action, resource, appSlug); err != nil {
			logger.Error(err)
			return
		}
		handler(w, r)
	}
}

func (p *Policy) execute(r *http.Request) (action, resource, appSlug string, err error) {
	vars := mux.Vars(r)
	for _, fn := range p.varsGetterFns {
		additionalVars, err := fn(vars)
		if err != nil {
			return action, resource, "", err
		}
		for key, val := range additionalVars {
			vars[key] = val
		}
	}
	var buf bytes.Buffer
	err = p.resourceTemplate.Execute(&buf, vars)
	appSlug, _ = vars["appSlug"]
	return p.action, buf.String(), appSlug, err
}
