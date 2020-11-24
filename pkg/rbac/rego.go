package rbac

import (
	"context"

	"github.com/open-policy-agent/opa/ast"
	"github.com/open-policy-agent/opa/rego"
	"github.com/pkg/errors"
)

var regoAllowModule string = `
package rbac.allow

default allow = false

allow {
    # for each role in that list
    r := input.roles[_]
    # lookup the policies list for role r
    policies := input.allowRolePolicies[r]
    # for each policy
    p := policies[_]
    # check if the permission granted to r matches the roles's request
    glob.match(p.action, [], input.action)
    glob.match(p.resource, [], input.resource)
}
`

var regoDenyModule string = `
package rbac.deny

default deny = false

deny {
    # for each role in that list
    r := input.roles[_]
    # lookup the policies list for role r
    policies := input.denyRolePolicies[r]
    # for each policy
    p := policies[_]
    # check if the permission granted to r matches the roles's request
    glob.match(p.action, [], input.action)
    glob.match(p.resource, [], input.resource)
}
`

var compiler *ast.Compiler

func init() {
	var err error
	compiler, err = ast.CompileModules(map[string]string{
		"rbac.rego.allow": regoAllowModule,
		"rbac.rego.deny":  regoDenyModule,
	})
	if err != nil {
		panic(errors.Wrap(err, "failed to compile rego module"))
	}
}

func CheckAccess(ctx context.Context, action, resource string, roles []string) (bool, error) {
	i := map[string]interface{}{
		"action":            action,
		"resource":          resource,
		"roles":             roles,
		"allowRolePolicies": DefaultAllowRolePolicies,
		"denyRolePolicies":  DefaultDenyRolePolicies,
	}
	return regoEval(ctx, i)
}

func regoEval(ctx context.Context, input map[string]interface{}) (bool, error) {
	deny, err := regoEvalDeny(ctx, input)
	if err != nil {
		return false, errors.Wrap(err, "failed to evaluate deny")
	} else if deny {
		return false, nil
	}

	allow, err := regoEvalAllow(ctx, input)
	return allow, errors.Wrap(err, "failed to evaluate allow")
}

func regoEvalAllow(ctx context.Context, input map[string]interface{}) (bool, error) {
	query := rego.New(
		rego.Query("data.rbac.allow.allow"),
		rego.Compiler(compiler),
		rego.Input(input),
	)
	results, err := query.Eval(ctx)
	if err != nil {
		return false, errors.Wrap(err, "failed to evaluate query")
	} else if len(results) == 0 {
		return false, errors.New("empty result set")
	} else if len(results[0].Expressions) == 0 {
		return false, errors.New("empty expressions")
	}

	allow, ok := results[0].Expressions[0].Value.(bool)
	if !ok {
		return false, errors.New("unexpected result type")
	}
	return allow, nil
}

func regoEvalDeny(ctx context.Context, input map[string]interface{}) (bool, error) {
	query := rego.New(
		rego.Query("data.rbac.deny.deny"),
		rego.Compiler(compiler),
		rego.Input(input),
	)
	results, err := query.Eval(ctx)
	if err != nil {
		return false, errors.Wrap(err, "failed to evaluate query")
	} else if len(results) == 0 {
		return false, errors.New("empty result set")
	} else if len(results[0].Expressions) == 0 {
		return false, errors.New("empty expressions")
	}

	deny, ok := results[0].Expressions[0].Value.(bool)
	if !ok {
		return false, errors.New("unexpected result type")
	}
	return deny, nil
}
