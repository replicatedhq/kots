package podresources

import (
	"testing"

	kotsadmtypes "github.com/replicatedhq/kots/pkg/kotsadm/types"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"k8s.io/apimachinery/pkg/api/resource"
)

func TestParseAllPodRequirementsFlags(t *testing.T) {
	tests := []struct {
		name    string
		flags   []string
		want    *kotsadmtypes.AllResourceRequirements
		wantErr bool
	}{
		{
			name: "defaults",
			want: &kotsadmtypes.AllResourceRequirements{
				Kotsadm: &kotsadmtypes.ResourceRequirements{
					PodCpuRequest:    resource.MustParse("100m"),
					PodMemoryRequest: resource.MustParse("100Mi"),
					PodCpuLimit:      resource.MustParse("1"),
					PodMemoryLimit:   resource.MustParse("2Gi"),
				},
				Minio: &kotsadmtypes.ResourceRequirements{
					PodCpuRequest:    resource.MustParse("50m"),
					PodMemoryRequest: resource.MustParse("100Mi"),
					PodCpuLimit:      resource.MustParse("100m"),
					PodMemoryLimit:   resource.MustParse("200Mi"),
				},
				Postgres: &kotsadmtypes.ResourceRequirements{
					PodCpuRequest:    resource.MustParse("100m"),
					PodMemoryRequest: resource.MustParse("100Mi"),
					PodCpuLimit:      resource.MustParse("200m"),
					PodMemoryLimit:   resource.MustParse("200Mi"),
				},
				Dex: &kotsadmtypes.ResourceRequirements{
					PodCpuRequest:    resource.MustParse("100m"),
					PodMemoryRequest: resource.MustParse("50Mi"),
					PodCpuLimit:      resource.MustParse("0"), // resource.MustParse("100m"),
					PodMemoryLimit:   resource.MustParse("0"), // resource.MustParse("50Mi"),
				},
			},
		},
		{
			name: "override",
			flags: []string{
				"--kotsadm-pod-cpu-request=1m",
				"--kotsadm-pod-mem-request=2Mi",
				"--kotsadm-pod-cpu-limit=3",
				"--kotsadm-pod-mem-limit=4Gi",
				"--minio-pod-cpu-request=5m",
				"--minio-pod-mem-request=6Mi",
				"--minio-pod-cpu-limit=7m",
				"--minio-pod-mem-limit=8Mi",
				"--postgres-pod-cpu-request=9m",
				"--postgres-pod-mem-request=10Mi",
				"--postgres-pod-cpu-limit=11m",
				"--postgres-pod-mem-limit=12Mi",
				"--dex-pod-cpu-request=13m",
				"--dex-pod-mem-request=14Mi",
				"--dex-pod-cpu-limit=15m",
				"--dex-pod-mem-limit=16Mi",
			},
			want: &kotsadmtypes.AllResourceRequirements{
				Kotsadm: &kotsadmtypes.ResourceRequirements{
					PodCpuRequest:         resource.MustParse("1m"),
					PodCpuRequestIsSet:    true,
					PodMemoryRequest:      resource.MustParse("2Mi"),
					PodMemoryRequestIsSet: true,
					PodCpuLimit:           resource.MustParse("3"),
					PodCpuLimitIsSet:      true,
					PodMemoryLimit:        resource.MustParse("4Gi"),
					PodMemoryLimitIsSet:   true,
				},
				Minio: &kotsadmtypes.ResourceRequirements{
					PodCpuRequest:         resource.MustParse("5m"),
					PodCpuRequestIsSet:    true,
					PodMemoryRequest:      resource.MustParse("6Mi"),
					PodMemoryRequestIsSet: true,
					PodCpuLimit:           resource.MustParse("7m"),
					PodCpuLimitIsSet:      true,
					PodMemoryLimit:        resource.MustParse("8Mi"),
					PodMemoryLimitIsSet:   true,
				},
				Postgres: &kotsadmtypes.ResourceRequirements{
					PodCpuRequest:         resource.MustParse("9m"),
					PodCpuRequestIsSet:    true,
					PodMemoryRequest:      resource.MustParse("10Mi"),
					PodMemoryRequestIsSet: true,
					PodCpuLimit:           resource.MustParse("11m"),
					PodCpuLimitIsSet:      true,
					PodMemoryLimit:        resource.MustParse("12Mi"),
					PodMemoryLimitIsSet:   true,
				},
				Dex: &kotsadmtypes.ResourceRequirements{
					PodCpuRequest:         resource.MustParse("13m"),
					PodCpuRequestIsSet:    true,
					PodMemoryRequest:      resource.MustParse("14Mi"),
					PodMemoryRequestIsSet: true,
					PodCpuLimit:           resource.MustParse("15m"),
					PodCpuLimitIsSet:      true,
					PodMemoryLimit:        resource.MustParse("16Mi"),
					PodMemoryLimitIsSet:   true,
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v := viper.New()
			flagSet := pflag.NewFlagSet("test", pflag.ContinueOnError)
			AllPodRequirementsFlags(flagSet)
			err := flagSet.Parse(tt.flags)
			if err != nil {
				t.Errorf("pflag.FlagSet.Parse() error = %v", err)
				return
			}
			err = v.BindPFlags(flagSet)
			if err != nil {
				t.Errorf("viper.BindPFlags() error = %v", err)
				return
			}
			got, err := ParseAllPodRequirementsFlags(v)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseAllPodRequirementsFlags() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			assert.Equal(t, tt.want.Kotsadm, got.Kotsadm)
			assert.Equal(t, tt.want.Minio, got.Minio)
			assert.Equal(t, tt.want.Postgres, got.Postgres)
			assert.Equal(t, tt.want.Dex, got.Dex)
		})
	}
}
