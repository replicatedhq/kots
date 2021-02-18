package snapshot

import (
	"testing"
	"time"

	"github.com/replicatedhq/kots/pkg/api/snapshot/types"
	"github.com/stretchr/testify/require"
	velerov1 "github.com/vmware-tanzu/velero/pkg/apis/velero/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

func Test_parseOpenShiftWarning(t *testing.T) {
	tests := []struct {
		name           string
		restoreWarning types.SnapshotError
		wantKind       string
		wantName       string
	}{
		{
			"parse rolebinding warning",
			types.SnapshotError{
				Message:   `could not restore, rolebindings.rbac.authorization.k8s.io "my-role-binding" already exists. Warning: the in-cluster version is different than the backed-up version.`,
				Namespace: "somens",
			},
			"rolebindings",
			"my-role-binding",
		},
		{
			"parse role warning",
			types.SnapshotError{
				Message:   `could not restore, roles.rbac.authorization.k8s.io "my-role" already exists. Warning: the in-cluster version is different than the backed-up version.`,
				Namespace: "somens",
			},
			"roles",
			"my-role",
		},
		{
			"parse some other warning",
			types.SnapshotError{
				Message:   `could not restore, serverlessservices.networking.internal.knative.dev "some-name" already exists. Warning: the in-cluster version is different than the backed-up version.`,
				Namespace: "somens",
			},
			"serverlessservices",
			"some-name",
		},
		{
			"parse unsupported warning",
			types.SnapshotError{
				Message:   `error restoring validatingwebhookconfigurations.admissionregistration.k8s.io/monitoring-prometheus-oper-admission: ValidatingWebhookConfiguration.admissionregistration.k8s.io "monitoring-prometheus-oper-admission" is invalid: webhooks[0].sideEffects: Unsupported value: "Unknown": supported values: "None", "NoneOnDryRun"`,
				Namespace: "somens",
			},
			"",
			"",
		},
	}

	req := require.New(t)
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			gotKind, gotName := parseOpenShiftWarning(test.restoreWarning)
			req.Equal(gotKind, test.wantKind)
			req.Equal(gotName, test.wantName)
		})
	}
}

type mockGetter struct {
}

func (g *mockGetter) GetClientSet() (kubernetes.Interface, error) {
	return nil, nil
}

func (g *mockGetter) IsOpenShift(kubernetes.Interface) bool {
	return true
}

func (g *mockGetter) GetRole(namespace, roleName string, clientset kubernetes.Interface) (*rbacv1.Role, error) {
	role := &rbacv1.Role{
		ObjectMeta: metav1.ObjectMeta{
			// warning about newer objects will be removed
			CreationTimestamp: metav1.NewTime(time.Now().Add(time.Hour * 10)),
		},
	}
	return role, nil
}

func (g *mockGetter) GetRoleBinding(namespace, roleBindingName string, clientset kubernetes.Interface) (*rbacv1.RoleBinding, error) {
	binding := &rbacv1.RoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			// warning about older objects will be kept
			CreationTimestamp: metav1.NewTime(time.Now().Add(time.Hour * -10)),
		},
	}
	return binding, nil
}

func Test_filterWarnings(t *testing.T) {
	tests := []struct {
		name           string
		restoreWarning types.SnapshotError
		want           bool
	}{
		{
			"parse rolebinding warning",
			types.SnapshotError{
				Message:   `could not restore, rolebindings.rbac.authorization.k8s.io "my-role-binding" already exists. Warning: the in-cluster version is different than the backed-up version.`,
				Namespace: "somens",
			},
			true, // because timestamp hardcoded in getter is post-restore
		},
		{
			"parse role warning",
			types.SnapshotError{
				Message:   `could not restore, roles.rbac.authorization.k8s.io "my-role" already exists. Warning: the in-cluster version is different than the backed-up version.`,
				Namespace: "somens",
			},
			false, // because timestamp hardcoded in getter is pre-restore
		},
		{
			"parse some other warning",
			types.SnapshotError{
				Message:   `could not restore, serverlessservices.networking.internal.knative.dev "some-name" already exists. Warning: the in-cluster version is different than the backed-up version.`,
				Namespace: "somens",
			},
			true,
		},
		{
			"parse unsupported warning",
			types.SnapshotError{
				Message:   `error restoring validatingwebhookconfigurations.admissionregistration.k8s.io/monitoring-prometheus-oper-admission: ValidatingWebhookConfiguration.admissionregistration.k8s.io "monitoring-prometheus-oper-admission" is invalid: webhooks[0].sideEffects: Unsupported value: "Unknown": supported values: "None", "NoneOnDryRun"`,
				Namespace: "somens",
			},
			true,
		},
	}

	req := require.New(t)

	warnings := make([]types.SnapshotError, 0)
	for _, test := range tests {
		warnings = append(warnings, test.restoreWarning)
	}
	restore := &velerov1.Restore{
		ObjectMeta: metav1.ObjectMeta{
			CreationTimestamp: metav1.NewTime(time.Now()),
		},
	}

	t.Run("filter warnings", func(t *testing.T) {
		got, err := filterWarnings(restore, warnings, &mockGetter{})
		req.NoError(err)
		for _, test := range tests {
			if test.want {
				req.Contains(got, test.restoreWarning)
			} else {
				req.NotContains(got, test.restoreWarning)
			}
		}
	})
}
