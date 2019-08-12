package ship

import (
	"context"
	"encoding/base64"
	"fmt"
	"strconv"

	"github.com/replicatedhq/kotsadm/worker/pkg/types"
	corev1 "k8s.io/api/core/v1"
	networkv1 "k8s.io/api/networking/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

func GetNamespace(ctx context.Context, session types.Session) *corev1.Namespace {
	labels := make(map[string]string)
	labels[session.GetType()] = session.GetID()
	labels["shipcloud-role"] = session.GetRole()

	namespace := corev1.Namespace{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "Namespace",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:   session.GetName(),
			Labels: labels,
		},
	}

	return &namespace
}

func GetServiceAccountSpec(ctx context.Context, session types.Session) *corev1.ServiceAccount {
	labels := make(map[string]string)
	labels[session.GetType()] = session.GetID()
	labels["shipcloud-role"] = session.GetRole()

	serviceAccount := corev1.ServiceAccount{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "ServiceAccount",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      session.GetName(),
			Namespace: session.GetName(),
			Labels:    labels,
		},
		Secrets: []corev1.ObjectReference{
			{
				APIVersion: "v1",
				Kind:       "Secret",
				Name:       session.GetName(),
				Namespace:  session.GetName(),
			},
		},
	}

	return &serviceAccount
}

func GetPodSpec(ctx context.Context, logLevel string, shipImage string, shipTag string, shipPullPolicy string, s3State *S3State, serviceAccountName string, session types.Session, githubToken string) *corev1.Pod {
	labels := make(map[string]string)
	labels[session.GetType()] = session.GetID()
	labels["shipcloud-role"] = session.GetRole()
	labels["s3-filepath"] = base64.RawStdEncoding.EncodeToString([]byte(session.GetS3Filepath()))
	labels["state-id"] = s3State.ID

	if session.GetType() == "ship-update" {
		labels["update-sequence"] = strconv.Itoa(session.GetUploadSequence())
	}
	if session.GetType() == "ship-edit" {
		labels["edit-sequence"] = strconv.Itoa(session.GetUploadSequence())
	}
	if session.GetParentWatchID() != nil {
		labels["parent-watch-id"] = *session.GetParentWatchID()
	}
	if session.GetParentSequence() != nil {
		labels["parent-sequence"] = strconv.Itoa(*session.GetParentSequence())
	}
	if shipImage == "" {
		shipImage = "replicated/ship"
	}
	if shipTag == "" {
		shipTag = "alpha"
	}
	if shipPullPolicy == "" {
		shipPullPolicy = string(corev1.PullAlways)
	}

	nodeSelector := make(map[string]string)
	if session.GetNodeSelector() != "" {
		nodeSelector["replicated/node-pool"] = session.GetNodeSelector()
	}

	limits := corev1.ResourceList{}
	limits[corev1.ResourceCPU] = resource.MustParse(session.GetCPULimit())
	limits[corev1.ResourceMemory] = resource.MustParse(session.GetMemoryLimit())
	requests := corev1.ResourceList{}
	requests[corev1.ResourceCPU] = resource.MustParse(session.GetCPURequest())
	requests[corev1.ResourceMemory] = resource.MustParse(session.GetMemoryRequest())

	var activeDeadlineSecondsRef int64 = 60 * 60

	pod := corev1.Pod{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "Pod",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      session.GetName(),
			Namespace: session.GetName(),
			Labels:    labels,
		},
		Spec: corev1.PodSpec{
			ServiceAccountName:    serviceAccountName,
			NodeSelector:          nodeSelector,
			ActiveDeadlineSeconds: &activeDeadlineSecondsRef,
			Containers: []corev1.Container{
				{
					Image:           fmt.Sprintf("%s:%s", shipImage, shipTag),
					ImagePullPolicy: corev1.PullPolicy(shipPullPolicy),
					Name:            session.GetID(),
					Resources: corev1.ResourceRequirements{
						Limits:   limits,
						Requests: requests,
					},
					Args: append(
						session.GetShipArgs(),
						[]string{
							"--log-level",
							logLevel,
							"--prefer-git",
							"--files-in-state",
							"--state-from",
							"url",
							"--state-put-url",
							s3State.PutURL,
							"--state-get-url",
							s3State.GetURL,
							"--upload-assets-to",
							session.GetUploadURL(),
							"--no-outro",
						}...,
					),
					Ports: []corev1.ContainerPort{{
						Name:          session.GetType(),
						ContainerPort: 8800,
					}},
					Env: []corev1.EnvVar{{Name: "GITHUB_TOKEN", Value: githubToken}},
				},
			},
			RestartPolicy: corev1.RestartPolicyNever,
		},
	}

	return &pod
}

func GetServiceSpec(ctx context.Context, session types.Session) *corev1.Service {
	labels := make(map[string]string)
	labels[session.GetType()] = session.GetID()
	labels["shipcloud-role"] = session.GetRole()

	service := corev1.Service{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "Service",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      session.GetName(),
			Namespace: session.GetName(),
			Labels:    labels,
		},
		Spec: corev1.ServiceSpec{
			Ports: []corev1.ServicePort{{
				Name:       session.GetType(),
				Port:       8800,
				TargetPort: intstr.FromInt(8800),
			}},
			Type:     corev1.ServiceTypeClusterIP,
			Selector: labels,
		},
	}

	return &service
}

func GetRoleSpec(ctx context.Context, session types.Session) *rbacv1.Role {
	labels := make(map[string]string)
	labels[session.GetType()] = session.GetID()
	labels["shipcloud-role"] = session.GetRole()

	role := rbacv1.Role{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "Role",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      session.GetName(),
			Namespace: session.GetName(),
			Labels:    labels,
		},
		Rules: []rbacv1.PolicyRule{
			{
				APIGroups:     []string{""},
				Resources:     []string{"secrets"},
				ResourceNames: []string{session.GetName()},
				Verbs:         metav1.Verbs{"get", "update"},
			},
		},
	}

	return &role
}

func GetRoleBindingSpec(ctx context.Context, session types.Session) *rbacv1.RoleBinding {
	labels := make(map[string]string)
	labels[session.GetType()] = session.GetID()
	labels["shipcloud-role"] = session.GetRole()

	role := rbacv1.RoleBinding{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "Role",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      session.GetName(),
			Namespace: session.GetName(),
			Labels:    labels,
		},
		Subjects: []rbacv1.Subject{
			{
				Kind:      "ServiceAccount",
				Name:      session.GetName(),
				Namespace: session.GetName(),
			},
		},
		RoleRef: rbacv1.RoleRef{
			APIGroup: "rbac.authorization.k8s.io",
			Kind:     "Role",
			Name:     session.GetName(),
		},
	}

	return &role
}

func GetNetworkPolicySpec(ctx context.Context, session types.Session) *networkv1.NetworkPolicy {
	labels := make(map[string]string)
	labels[session.GetType()] = session.GetID()
	labels["shipcloud-role"] = session.GetRole()

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
			Name:      session.GetName(),
			Namespace: session.GetName(),
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
