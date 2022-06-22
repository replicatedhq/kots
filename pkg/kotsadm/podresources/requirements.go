package podresources

import (
	"github.com/pkg/errors"
	kotsadmtypes "github.com/replicatedhq/kots/pkg/kotsadm/types"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
)

var (
	cpu50m      = resource.MustParse("50m")
	cpu100m     = resource.MustParse("100m")
	cpu1        = resource.MustParse("1")
	memory50Mi  = resource.MustParse("50Mi")
	memory100Mi = resource.MustParse("100Mi")
	memory2Gi   = resource.MustParse("2Gi")
)

var (
	KotsadmPodRequirements = PodRequirements{
		PodCpuRequest:    "100m",
		PodMemoryRequest: "100Mi",
		PodCpuLimit:      "1",
		PodMemoryLimit:   "2Gi",
	}
	MinioPodRequirements = PodRequirements{
		PodCpuRequest:    "50m",
		PodMemoryRequest: "100Mi",
		PodCpuLimit:      "100m",
		PodMemoryLimit:   "200Mi",
	}
	PostgresPodRequirements = PodRequirements{
		PodCpuRequest:    "100m",
		PodMemoryRequest: "100Mi",
		PodCpuLimit:      "200m",
		PodMemoryLimit:   "200Mi",
	}
	DexPodRequirements = PodRequirements{
		PodCpuRequest:    "100m",
		PodMemoryRequest: "50Mi",
		// PodCpuLimit:      "100m",
		// PodMemoryLimit:   "50Mi",
	}
)

type PodRequirements struct {
	PodCpuRequest    string
	PodMemoryRequest string
	PodCpuLimit      string
	PodMemoryLimit   string
}

func AllPodRequirementsFlags(flagset *pflag.FlagSet) {
	KotsadmPodRequirementsFlags(flagset)
	MinioPodRequirementsFlags(flagset)
	PostgresPodRequirementsFlags(flagset)
	DexPodRequirementsFlags(flagset)
}

func KotsadmPodRequirementsFlags(flagset *pflag.FlagSet) {
	PodRequirementsFlags(flagset, "kotsadm", KotsadmPodRequirements)
}

func MinioPodRequirementsFlags(flagset *pflag.FlagSet) {
	PodRequirementsFlags(flagset, "minio", MinioPodRequirements)
}

func PostgresPodRequirementsFlags(flagset *pflag.FlagSet) {
	PodRequirementsFlags(flagset, "postgres", PostgresPodRequirements)
}

func DexPodRequirementsFlags(flagset *pflag.FlagSet) {
	PodRequirementsFlags(flagset, "dex", DexPodRequirements)
}

func PodRequirementsFlags(flagset *pflag.FlagSet, key string, r PodRequirements) {
	flagset.String(key+"-pod-cpu-request", r.PodCpuRequest, key+" pod cpu request")
	flagset.String(key+"-pod-mem-request", r.PodMemoryRequest, key+" pod memory request")
	flagset.String(key+"-pod-cpu-limit", r.PodCpuLimit, key+" pod cpu limit")
	flagset.String(key+"-pod-mem-limit", r.PodMemoryLimit, key+" pod memory limit")
}

func ParseAllPodRequirementsFlags(v *viper.Viper) (*kotsadmtypes.AllResourceRequirements, error) {
	var err error
	r := &kotsadmtypes.AllResourceRequirements{}

	r.Kotsadm, err = ParseKotsadmPodRequirementsFlags(v)
	if err != nil {
		return nil, errors.Wrap(err, "kotsadm")
	}

	r.Minio, err = ParseMinioPodRequirementsFlags(v)
	if err != nil {
		return nil, errors.Wrap(err, "minio")
	}

	r.Postgres, err = ParsePostgresPodRequirementsFlags(v, "postgres")
	if err != nil {
		return nil, errors.Wrap(err, "postgres")
	}

	r.Dex, err = ParseDexPodRequirementsFlags(v)
	if err != nil {
		return nil, errors.Wrap(err, "dex")
	}

	return r, nil
}

func ParseKotsadmPodRequirementsFlags(v *viper.Viper) (*kotsadmtypes.ResourceRequirements, error) {
	return ParsePodRequirementsFlags(v, "kotsadm")
}

func ParseMinioPodRequirementsFlags(v *viper.Viper) (*kotsadmtypes.ResourceRequirements, error) {
	return ParsePodRequirementsFlags(v, "minio")
}

func ParsePostgresPodRequirementsFlags(v *viper.Viper, key string) (*kotsadmtypes.ResourceRequirements, error) {
	return ParsePodRequirementsFlags(v, "postgres")
}

func ParseDexPodRequirementsFlags(v *viper.Viper) (*kotsadmtypes.ResourceRequirements, error) {
	return ParsePodRequirementsFlags(v, "dex")
}

func ParsePodRequirementsFlags(v *viper.Viper, key string) (*kotsadmtypes.ResourceRequirements, error) {
	var err error
	r := &kotsadmtypes.ResourceRequirements{}
	r.PodCpuRequest, err = resource.ParseQuantity(emptyToZero(v.GetString(key + "-pod-cpu-request")))
	if err != nil {
		return nil, errors.Wrap(err, key+"-pod-cpu-request")
	}
	if v.IsSet(key + "-pod-cpu-request") {
		r.PodCpuRequestIsSet = true
	}
	r.PodMemoryRequest, err = resource.ParseQuantity(emptyToZero(v.GetString(key + "-pod-mem-request")))
	if err != nil {
		return nil, errors.Wrap(err, key+"-pod-mem-request")
	}
	if v.IsSet(key + "-pod-mem-request") {
		r.PodMemoryRequestIsSet = true
	}
	r.PodCpuLimit, err = resource.ParseQuantity(emptyToZero(v.GetString(key + "-pod-cpu-limit")))
	if err != nil {
		return nil, errors.Wrap(err, key+"-pod-cpu-limit")
	}
	if v.IsSet(key + "-pod-cpu-limit") {
		r.PodCpuLimitIsSet = true
	}
	r.PodMemoryLimit, err = resource.ParseQuantity(emptyToZero(v.GetString(key + "-pod-mem-limit")))
	if err != nil {
		return nil, errors.Wrap(err, key+"-pod-mem-limit")
	}
	if v.IsSet(key + "-pod-mem-limit") {
		r.PodMemoryLimitIsSet = true
	}
	return r, nil
}

func GetKotsadmInitRequirements(podName string) corev1.ResourceRequirements {
	switch podName {
	case "restore-db", "restore-s3", "restore-data", "migrate-s3":
		return corev1.ResourceRequirements{
			Limits: corev1.ResourceList{
				"cpu":    cpu1,
				"memory": memory2Gi,
			},
			Requests: corev1.ResourceList{
				"cpu":    cpu100m,
				"memory": memory100Mi,
			},
		}
	case "schemahero-plan", "schemahero-apply":
		fallthrough
	default:
		return corev1.ResourceRequirements{
			Limits: corev1.ResourceList{
				"cpu":    cpu100m,
				"memory": memory100Mi,
			},
			Requests: corev1.ResourceList{
				"cpu":    cpu50m,
				"memory": memory50Mi,
			},
		}
	}
}

func GetS3OpsRequirements() corev1.ResourceRequirements {
	return corev1.ResourceRequirements{
		Limits: corev1.ResourceList{
			"cpu":    cpu100m,
			"memory": memory100Mi,
		},
		Requests: corev1.ResourceList{
			"cpu":    cpu50m,
			"memory": memory50Mi,
		},
	}
}

func GetMinioOpsRequirements() corev1.ResourceRequirements {
	return corev1.ResourceRequirements{
		Limits: corev1.ResourceList{
			"cpu":    cpu100m,
			"memory": memory100Mi,
		},
		Requests: corev1.ResourceList{
			"cpu":    cpu50m,
			"memory": memory50Mi,
		},
	}
}

func emptyToZero(q string) string {
	if len(q) == 0 {
		return "0"
	}
	return q
}
