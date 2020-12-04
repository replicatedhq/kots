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
				allowRolePolicies: roleToAllowRolePolicies(ClusterAdminRole),
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
				allowRolePolicies: roleToAllowRolePolicies(ClusterAdminRole),
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
						{Action: "**", Resource: "app.*.downstream.filetree."},
					},
				},
			},
			want: true,
		},
		{
			name: "deny app filetree",
			args: args{
				action:   "read",
				resource: "app.my-app.downstream.filetree.",
				roles:    []string{"admin"},
				allowRolePolicies: map[string][]types.Policy{
					"admin": ClusterAdminRole.Allow,
				},
				denyRolePolicies: map[string][]types.Policy{
					"admin": {
						{Action: "**", Resource: "app.*.downstream.filetree."},
					},
				},
			},
			want: false,
		},
		{
			name: "multiple roles allow",
			args: args{
				action:   "read",
				resource: "app.my-app.downstream.logs.",
				roles:    []string{SupportRole.ID, "yeslogs"},
				allowRolePolicies: map[string][]types.Policy{
					SupportRole.ID: SupportRole.Allow,
					"yeslogs": {
						{Action: "**", Resource: "app.*.downstream.logs."},
					},
				},
				denyRolePolicies: map[string][]types.Policy{
					SupportRole.ID: SupportRole.Deny,
				},
			},
			want: true,
		},
		{
			name: "multiple roles deny",
			args: args{
				action:   "read",
				resource: "app.my-app.downstream.logs.",
				roles:    []string{ClusterAdminRole.ID, "nologs"},
				allowRolePolicies: map[string][]types.Policy{
					ClusterAdminRole.ID: ClusterAdminRole.Allow,
				},
				denyRolePolicies: map[string][]types.Policy{
					ClusterAdminRole.ID: ClusterAdminRole.Deny,
					"nologs": {
						{Action: "**", Resource: "app.*.downstream.logs."},
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

func TestCheckAccess(t *testing.T) {
	type args struct {
		action       string
		resource     string
		sessionRoles []string
		appSlugs     []string
	}
	tests := []struct {
		name    string
		args    args
		want    bool
		wantErr bool
	}{
		{
			name: "single role allow",
			args: args{
				action:       "read",
				resource:     "app.my-app.downstream.filetree.",
				sessionRoles: []string{ClusterAdminRole.ID},
			},
			want: true,
		},
		{
			name: "single role deny",
			args: args{
				action:       "read",
				resource:     "app.my-app.downstream.filetree.",
				sessionRoles: []string{SupportRole.ID},
			},
			want: false,
		},
		{
			name: "multiple roles allow",
			args: args{
				action:       "read",
				resource:     "app.my-app.downstream.filetree.",
				sessionRoles: []string{SupportRole.ID, ClusterAdminRole.ID},
			},
			want: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := CheckAccess(context.Background(), tt.args.action, tt.args.resource, tt.args.sessionRoles, tt.args.appSlugs)
			if (err != nil) != tt.wantErr {
				t.Errorf("CheckAccess() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("CheckAccess() = %v, want %v", got, tt.want)
			}
		})
	}
}
