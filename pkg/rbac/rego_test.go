package rbac

import (
	"context"
	"testing"

	"github.com/replicatedhq/kots/pkg/rbac/types"
)

func Test_regoEval(t *testing.T) {
	type args struct {
		action            string
		resource          string
		roles             []string
		allowRolePolicies map[string][]types.Policy
		denyRolePolicies  map[string][]types.Policy
	}
	tests := []struct {
		name    string
		args    args
		want    bool
		wantErr bool
	}{
		{
			name: "cluster-admin role",
			args: args{
				action:            "read",
				resource:          "app.my-app",
				roles:             []string{"cluster-admin"},
				allowRolePolicies: DefaultAllowRolePolicies,
			},
			want: true,
		},
		{
			name: "readonly read",
			args: args{
				action:   "read",
				resource: "app.my-app",
				roles:    []string{"readonly"},
				allowRolePolicies: map[string][]types.Policy{
					"readonly": {
						{Action: "read", Resource: "**"},
					},
					ClusterAdminRole.ID: ClusterAdminRole.Allow,
				},
			},
			want: true,
		},
		{
			name: "readonly write",
			args: args{
				action:   "write",
				resource: "app.my-app",
				roles:    []string{"readonly"},
				allowRolePolicies: map[string][]types.Policy{
					"readonly": {
						{Action: "read", Resource: "**"},
					},
					ClusterAdminRole.ID: ClusterAdminRole.Allow,
				},
			},
			want: false,
		},
		{
			name: "correct namespace",
			args: args{
				action:   "write",
				resource: "app.my-app.supportbundle.my-bundle",
				roles:    []string{"myapp.superadmin"},
				allowRolePolicies: map[string][]types.Policy{
					"myapp.superadmin": {
						{Action: "**", Resource: "app.my-app.**"},
					},
					ClusterAdminRole.ID: ClusterAdminRole.Allow,
				},
			},
			want: true,
		},
		{
			name: "wrong namespace",
			args: args{
				action:   "write",
				resource: "supportbundle.my-bundle",
				roles:    []string{"myapp.superadmin"},
				allowRolePolicies: map[string][]types.Policy{
					"myapp.superadmin": {
						{Action: "**", Resource: "app.my-app.**"},
					},
					ClusterAdminRole.ID: ClusterAdminRole.Allow,
				},
			},
			want: false,
		},
		{
			name: "only supportbundle",
			args: args{
				action:   "write",
				resource: "supportbundle.my-bundle",
				roles:    []string{"myapp.support"},
				allowRolePolicies: map[string][]types.Policy{
					"myapp.support": {
						{Action: "**", Resource: "supportbundle.*"},
						{Action: "**", Resource: "**.supportbundle.*"},
					},
					ClusterAdminRole.ID: ClusterAdminRole.Allow,
				},
			},
			want: true,
		},
		{
			name: "app supportbundle",
			args: args{
				action:   "write",
				resource: "app.myapp.supportbundle.my-bundle",
				roles:    []string{"myapp.support"},
				allowRolePolicies: map[string][]types.Policy{
					"myapp.support": {
						{Action: "**", Resource: "supportbundle.*"},
						{Action: "**", Resource: "**.supportbundle.*"},
					},
					ClusterAdminRole.ID: ClusterAdminRole.Allow,
				},
			},
			want: true,
		},
		{
			name: "not supportbundle",
			args: args{
				action:   "write",
				resource: "redactor.my-redactor",
				roles:    []string{"myapp.support"},
				allowRolePolicies: map[string][]types.Policy{
					"myapp.support": {
						{Action: "**", Resource: "supportbundle.*"},
						{Action: "**", Resource: "**.supportbundle.*"},
					},
					ClusterAdminRole.ID: ClusterAdminRole.Allow,
				},
			},
			want: false,
		},
		{
			name: "undefined role",
			args: args{
				action:            "read",
				resource:          "app.my-app",
				roles:             []string{"undefined"},
				allowRolePolicies: DefaultAllowRolePolicies,
			},
			want: false,
		},
		{
			name: "allow app filetree",
			args: args{
				action:   "read",
				resource: "app.my-app.",
				roles:    []string{"admin"},
				allowRolePolicies: map[string][]types.Policy{
					"admin": ClusterAdminRole.Allow,
				},
				denyRolePolicies: map[string][]types.Policy{
					"admin": {
						{Action: "**", Resource: "app.*.filetree."},
					},
				},
			},
			want: true,
		},
		{
			name: "deny app filetree",
			args: args{
				action:   "read",
				resource: "app.my-app.filetree.",
				roles:    []string{"admin"},
				allowRolePolicies: map[string][]types.Policy{
					"admin": ClusterAdminRole.Allow,
				},
				denyRolePolicies: map[string][]types.Policy{
					"admin": {
						{Action: "**", Resource: "app.*.filetree."},
					},
				},
			},
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			input := map[string]interface{}{
				"action":            tt.args.action,
				"resource":          tt.args.resource,
				"roles":             tt.args.roles,
				"allowRolePolicies": tt.args.allowRolePolicies,
				"denyRolePolicies":  tt.args.denyRolePolicies,
			}
			got, err := regoEval(context.Background(), input)
			if (err != nil) != tt.wantErr {
				t.Errorf("regoEval() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("regoEval() = %v, want %v", got, tt.want)
			}
		})
	}
}
