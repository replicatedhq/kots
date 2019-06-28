package troubleshoot

import (
	"context"
	"fmt"

	"github.com/replicatedhq/ship-cluster/worker/pkg/types"
	corev1 "k8s.io/api/core/v1"
	networkv1 "k8s.io/api/networking/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func name(id string) string {
	return fmt.Sprintf("analyze-%s", id)
}

func GetNamespace(ctx context.Context, supportBundle *types.SupportBundle) *corev1.Namespace {
	labels := make(map[string]string)
	labels["supportbundle-id"] = supportBundle.ID
	labels["shipcloud-role"] = "analyze"

	namespace := corev1.Namespace{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "Namespace",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:   name(supportBundle.ID),
			Labels: labels,
		},
	}

	return &namespace
}

func GetServiceAccountSpec(ctx context.Context, supportBundle *types.SupportBundle) *corev1.ServiceAccount {
	labels := make(map[string]string)
	labels["supportbundle-id"] = supportBundle.ID
	labels["shipcloud-role"] = "analyze"

	serviceAccount := corev1.ServiceAccount{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "ServiceAccount",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name(supportBundle.ID),
			Namespace: name(supportBundle.ID),
			Labels:    labels,
		},
		Secrets: []corev1.ObjectReference{
			{
				APIVersion: "v1",
				Kind:       "Secret",
				Name:       name(supportBundle.ID),
				Namespace:  name(supportBundle.ID),
			},
		},
	}

	return &serviceAccount
}

func GetConfigMapSpec(ctx context.Context, supportBundle *types.SupportBundle, analyzeSpec string) *corev1.ConfigMap {
	labels := make(map[string]string)
	labels["supportbundle-id"] = supportBundle.ID
	labels["shipcloud-role"] = "analyze"

	configMap := corev1.ConfigMap{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "ConfigMap",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name(supportBundle.ID),
			Namespace: name(supportBundle.ID),
			Labels:    labels,
		},
		Data: map[string]string{
			"analyze.yaml": analyzeSpec,
		},
	}

	return &configMap
}

func GetPodSpec(ctx context.Context, logLevel string, analyzeImage string, analyzeTag string, shipPullPolicy string, serviceAccountName string, supportBundle *types.SupportBundle, bundleGetURI string, desiredNodeSelector string) *corev1.Pod {
	labels := make(map[string]string)
	labels["supportbundle-id"] = supportBundle.ID
	labels["shipcloud-role"] = "analyze"

	if analyzeImage == "" {
		analyzeImage = "replicated/analyze"
	}
	if analyzeTag == "" {
		analyzeTag = "latest"
	}
	if shipPullPolicy == "" {
		shipPullPolicy = string(corev1.PullAlways)
	}

	nodeSelector := make(map[string]string)
	if desiredNodeSelector != "" {
		nodeSelector["replicated/node-pool"] = desiredNodeSelector
	}

	limits := corev1.ResourceList{}
	limits[corev1.ResourceCPU] = resource.MustParse("500m")
	limits[corev1.ResourceMemory] = resource.MustParse("500Mi")
	requests := corev1.ResourceList{}
	requests[corev1.ResourceCPU] = resource.MustParse("100m")
	requests[corev1.ResourceMemory] = resource.MustParse("100Mi")

	var activeDeadlineSecondsRef int64 = 60 * 60

	pod := corev1.Pod{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "Pod",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name(supportBundle.ID),
			Namespace: name(supportBundle.ID),
			Labels:    labels,
		},
		Spec: corev1.PodSpec{
			ServiceAccountName:    serviceAccountName,
			NodeSelector:          nodeSelector,
			ActiveDeadlineSeconds: &activeDeadlineSecondsRef,
			Containers: []corev1.Container{
				{
					Image:           fmt.Sprintf("%s:%s", analyzeImage, analyzeTag),
					ImagePullPolicy: corev1.PullPolicy(shipPullPolicy),
					Name:            name(supportBundle.ID),
					Resources: corev1.ResourceRequirements{
						Limits:   limits,
						Requests: requests,
					},
					Args: []string{
						"run",
						bundleGetURI,
						"--output",
						"json",
						"-f",
						"/specs/analyze.yaml",
					},
					VolumeMounts: []corev1.VolumeMount{
						{
							Name:      "specs",
							MountPath: "/specs",
						},
					},
				},
			},
			Volumes: []corev1.Volume{
				{
					Name: "specs",
					VolumeSource: corev1.VolumeSource{
						ConfigMap: &corev1.ConfigMapVolumeSource{
							LocalObjectReference: corev1.LocalObjectReference{
								Name: name(supportBundle.ID),
							},
						},
					},
				},
			},

			RestartPolicy: corev1.RestartPolicyNever,
		},
	}

	return &pod
}

func GetRoleSpec(ctx context.Context, supportBundle *types.SupportBundle) *rbacv1.Role {
	labels := make(map[string]string)
	labels["supportbundle-id"] = supportBundle.ID
	labels["shipcloud-role"] = "analyze"

	role := rbacv1.Role{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "Role",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name(supportBundle.ID),
			Namespace: name(supportBundle.ID),
			Labels:    labels,
		},
		Rules: []rbacv1.PolicyRule{
			{
				APIGroups:     []string{""},
				Resources:     []string{""},
				ResourceNames: []string{name(supportBundle.ID)},
				Verbs:         metav1.Verbs{""},
			},
		},
	}

	return &role
}

func GetRoleBindingSpec(ctx context.Context, supportBundle *types.SupportBundle) *rbacv1.RoleBinding {
	labels := make(map[string]string)
	labels["supportbundle-id"] = supportBundle.ID
	labels["shipcloud-role"] = "analyze"

	role := rbacv1.RoleBinding{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "Role",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name(supportBundle.ID),
			Namespace: name(supportBundle.ID),
			Labels:    labels,
		},
		Subjects: []rbacv1.Subject{
			{
				Kind:      "ServiceAccount",
				Name:      name(supportBundle.ID),
				Namespace: name(supportBundle.ID),
			},
		},
		RoleRef: rbacv1.RoleRef{
			APIGroup: "rbac.authorization.k8s.io",
			Kind:     "Role",
			Name:     name(supportBundle.ID),
		},
	}

	return &role
}

func GetNetworkPolicySpec(ctx context.Context, supportBundle *types.SupportBundle) *networkv1.NetworkPolicy {
	labels := make(map[string]string)
	labels["supportbundle-id"] = supportBundle.ID
	labels["shipcloud-role"] = "analyze"

	ipBlock := networkv1.IPBlock{
		CIDR: "0.0.0.0/0",
		Except: []string{
			"169.254.169.254/32",
		},
	}

	network := networkv1.NetworkPolicy{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "networking.k8s.io/v1",
			Kind:       "NetworkPolicy",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name(supportBundle.ID),
			Namespace: name(supportBundle.ID),
			Labels:    labels,
		},
		Spec: networkv1.NetworkPolicySpec{
			PolicyTypes: []networkv1.PolicyType{
				networkv1.PolicyTypeEgress,
			},
			Egress: []networkv1.NetworkPolicyEgressRule{
				{
					To: []networkv1.NetworkPolicyPeer{
						{
							IPBlock: &ipBlock,
						},
					},
				},
			},
		},
	}

	return &network
}
