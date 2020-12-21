package rbac

import (
	"context"

	"github.com/open-policy-agent/opa/ast"
	"github.com/open-policy-agent/opa/rego"
	"github.com/pkg/errors"
	"github.com/replicatedhq/kots/pkg/rbac/types"
)

/*
The RBAC module defined in this rego document is deny by default.
Deny will always take precedence over allow.
Policies are defined with glob matching patterns with dots as separators.
Multiple roles can be used as input.

Logic is as follows:
1. if deny, stop and deny
2. if allow, stop and allow
3. deny
*/

var regoModule string = `
package rbac

default allow = false
default deny = false

allow {
    not deny

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
		"rbac": regoModule,
	})
	if err != nil {
		panic(errors.Wrap(err, "failed to compile rego module"))
	}
}

func CheckAccess(ctx context.Context, roles []types.Role, action, resource string, sessionRoles []string) (bool, error) {
	for _, role := range roles {
		i := map[string]interface{}{
			"action":            action,
			"resource":          resource,
			"roles":             sessionRoles,
			"allowRolePolicies": roleToAllowRolePolicies(role),
			"denyRolePolicies":  roleToDenyRolePolicies(role),
		}
		allow, err := regoEval(ctx, i)
		if err != nil {
			return false, errors.Wrapf(err, "failed to evaluate role %s", role.ID)
		} else if allow {
			return true, nil
		}
	}
	return false, nil
}

func roleToAllowRolePolicies(role types.Role) map[string][]types.Policy {
	return map[string][]types.Policy{
		role.ID: role.Allow,
	}
}

func roleToDenyRolePolicies(role types.Role) map[string][]types.Policy {
	return map[string][]types.Policy{
		role.ID: role.Deny,
	}
}

func regoEval(ctx context.Context, input map[string]interface{}) (bool, error) {
	query := rego.New(
		rego.Query("data.rbac.allow"),
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
