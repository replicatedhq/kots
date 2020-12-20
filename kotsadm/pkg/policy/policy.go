package policy

import (
	"bytes"
	"net/http"
	"text/template"

	"github.com/gorilla/mux"
	"github.com/replicatedhq/kots/kotsadm/pkg/store"
)

type VarsGetter func(kotsStore store.KOTSStore, vars map[string]string) (map[string]string, error)

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

func (p *Policy) execute(r *http.Request, kotsStore store.KOTSStore) (action, resource string, err error) {
	vars := mux.Vars(r)
	for _, fn := range p.varsGetterFns {
		additionalVars, err := fn(kotsStore, vars)
		if err != nil {
			return action, resource, err
		}
		for key, val := range additionalVars {
			vars[key] = val
		}
	}
	var buf bytes.Buffer
	err = p.resourceTemplate.Execute(&buf, vars)
	return p.action, buf.String(), err
}
