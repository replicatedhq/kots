package snapshot

import (
	"testing"

	"github.com/stretchr/testify/require"
	velerov1 "github.com/vmware-tanzu/velero/pkg/apis/velero/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func Test_validateInstanceBackupForRestore(t *testing.T) {
	instanceAnnotations := map[string]string{
		"kots.io/instance":                 "true",
		"kots.io/kotsadm-image":            "kotsadm/kotsadm:v1.0.0",
		"kots.io/kotsadm-deploy-namespace": "kotsadm",
	}

	withoutAnnotation := func(key string) map[string]string {
		annotations := map[string]string{}
		for k, v := range instanceAnnotations {
			if k != key {
				annotations[k] = v
			}
		}
		return annotations
	}

	tests := []struct {
		name          string
		annotations   map[string]string
		phase         velerov1.BackupPhase
		wantImage     string
		wantNamespace string
		wantErr       string
	}{
		{
			name:          "completed instance backup",
			annotations:   instanceAnnotations,
			phase:         velerov1.BackupPhaseCompleted,
			wantImage:     "kotsadm/kotsadm:v1.0.0",
			wantNamespace: "kotsadm",
		},
		{
			name:        "not an instance backup",
			annotations: map[string]string{},
			phase:       velerov1.BackupPhaseCompleted,
			wantErr:     "backup provided is not an instance backup",
		},
		{
			name:        "missing kotsadm image annotation",
			annotations: withoutAnnotation("kots.io/kotsadm-image"),
			phase:       velerov1.BackupPhaseCompleted,
			wantErr:     "failed to find kotsadm image annotation",
		},
		{
			name:        "missing kotsadm deploy namespace annotation",
			annotations: withoutAnnotation("kots.io/kotsadm-deploy-namespace"),
			phase:       velerov1.BackupPhaseCompleted,
			wantErr:     "failed to find kotsadm deploy namespace annotation",
		},
		{
			name:        "failed backup",
			annotations: instanceAnnotations,
			phase:       velerov1.BackupPhaseFailed,
			wantErr:     "cannot restore a failed backup",
		},
		{
			name:        "partially failed backup",
			annotations: instanceAnnotations,
			phase:       velerov1.BackupPhasePartiallyFailed,
			wantErr:     "cannot restore a failed backup",
		},
		{
			name:        "in progress backup",
			annotations: instanceAnnotations,
			phase:       velerov1.BackupPhaseInProgress,
			wantErr:     "backup is still in progress",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			backup := velerov1.Backup{
				ObjectMeta: metav1.ObjectMeta{
					Name:        "instance-abcd",
					Annotations: tt.annotations,
				},
				Status: velerov1.BackupStatus{
					Phase: tt.phase,
				},
			}

			image, namespace, err := validateInstanceBackupForRestore(backup)
			if tt.wantErr != "" {
				require.EqualError(t, err, tt.wantErr)
				return
			}
			require.NoError(t, err)
			require.Equal(t, tt.wantImage, image)
			require.Equal(t, tt.wantNamespace, namespace)
		})
	}
}
