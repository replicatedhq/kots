package snapshot

import (
	"context"
	"testing"

	"github.com/replicatedhq/kots/pkg/kotsadm/types"
	kotsadmtypes "github.com/replicatedhq/kots/pkg/kotsadm/types"
	kotssnapshottypes "github.com/replicatedhq/kots/pkg/snapshot/types"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/fake"
)

func Test_ensureFileSystemMinioDeployment(t *testing.T) {
	hostPathString := "/test/hostpath"
	type args struct {
		clientSet        kubernetes.Interface
		deployOptions    FileSystemDeployOptions
		registryConfig   types.RegistryConfig
		marshalledSecret []byte
	}
	tests := []struct {
		name     string
		args     args
		validate func(t *testing.T, clientset kubernetes.Interface)
	}{
		{
			name: "new hostpath deployment",
			args: args{
				clientSet: fake.NewClientset(
					&corev1.ConfigMap{
						ObjectMeta: metav1.ObjectMeta{
							Name:      kotsadmtypes.KotsadmConfigMap,
							Namespace: "test-namespace",
						},
						Data: map[string]string{
							"additional-annotations": "abc/xyz=test-annotation1,test.annotation/two=test.value/two/test",
							"additional-labels":      "xyz=label2,abc=123",
						},
					},
				),
				deployOptions: FileSystemDeployOptions{
					Namespace:   "test-namespace",
					IsOpenShift: false,
					ForceReset:  false,
					FileSystemConfig: kotssnapshottypes.FileSystemConfig{
						HostPath: &hostPathString,
					},
				},
			},
			validate: func(t *testing.T, clientset kubernetes.Interface) {
				createdDeployment, err := clientset.AppsV1().Deployments("test-namespace").Get(context.Background(), "kotsadm-fs-minio", metav1.GetOptions{})
				require.NoError(t, err)
				require.NotNil(t, createdDeployment)

				// check annotations
				require.Equal(t, map[string]string{"abc/xyz": "test-annotation1", "test.annotation/two": "test.value/two/test"}, createdDeployment.Annotations)
				// check labels
				require.Equal(t, map[string]string{"xyz": "label2", "abc": "123"}, createdDeployment.Labels)

				// check pod template annotations
				require.Equal(t, map[string]string{"abc/xyz": "test-annotation1", "test.annotation/two": "test.value/two/test", "kots.io/fs-minio-creds-secret-checksum": "d41d8cd98f00b204e9800998ecf8427e"}, createdDeployment.Spec.Template.Annotations)
				// check pod template labels
				require.Equal(t, map[string]string{"xyz": "label2", "abc": "123", "app": "kotsadm-fs-minio"}, createdDeployment.Spec.Template.Labels)
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			req := require.New(t)

			err := ensureFileSystemMinioDeployment(ctx, tt.args.clientSet, tt.args.deployOptions, tt.args.registryConfig, tt.args.marshalledSecret)
			req.NoError(err)
			tt.validate(t, tt.args.clientSet)
		})
	}
}
