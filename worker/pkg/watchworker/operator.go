package watchworker

import (
	"fmt"

	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"github.com/pkg/errors"
	"github.com/replicatedhq/ship-cluster/worker/pkg/types"
	shipv1beta1 "github.com/replicatedhq/ship-operator/pkg/apis/ship/v1beta1"
	corev1 "k8s.io/api/core/v1"
	networkv1 "k8s.io/api/networking/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func namespaceForWatch(watch *types.Watch) *corev1.Namespace {
	labels := make(map[string]string)
	labels["ship-watch"] = watch.ID
	labels["shipcloud-role"] = "watch"

	namespace := corev1.Namespace{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "Namespace",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:   watch.Namespace(),
			Labels: labels,
		},
	}

	return &namespace
}

func roleSpec(watch *types.Watch) *rbacv1.Role {
	role := rbacv1.Role{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "Role",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "operator",
			Namespace: watch.Namespace(),
		},
		Rules: []rbacv1.PolicyRule{
			{
				APIGroups:     []string{""},
				Resources:     []string{"secrets"},
				ResourceNames: []string{watch.ID},
				Verbs:         metav1.Verbs{"get", "update"},
			},
		},
	}

	return &role
}

func roleBindingSpec(watch *types.Watch) *rbacv1.RoleBinding {
	roleBinding := rbacv1.RoleBinding{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "RoleBinding",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "operator",
			Namespace: watch.Namespace(),
		},
		Subjects: []rbacv1.Subject{
			{
				Kind:      "ServiceAccount",
				Name:      "default",
				Namespace: watch.Namespace(),
			},
		},
		RoleRef: rbacv1.RoleRef{
			APIGroup: "rbac.authorization.k8s.io",
			Kind:     "Role",
			Name:     "operator",
		},
	}

	return &roleBinding
}

func watchToCustomResource(shipImage string, shipTag string, shipPullPolicy string, watch *types.Watch, namespaceName string, githubToken string) (*corev1.Secret, *shipv1beta1.ShipWatch) {
	images := make([]shipv1beta1.ImageSpec, 0, 0)

	images = append(images, shipv1beta1.ImageSpec{
		Image:           shipImage,
		Tag:             shipTag,
		ImagePullPolicy: shipPullPolicy,
	}, shipv1beta1.ImageSpec{
		Image:           "replicated/ship-operator-tools",
		Tag:             "latest",
		ImagePullPolicy: "Always",
	})

	stateSecret := corev1.Secret{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "Secret",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      watch.ID,
			Namespace: namespaceName,
		},
		Data: map[string][]byte{
			"state.json": []byte(watch.StateJSON),
		},
	}

	actions := make([]shipv1beta1.ActionSpec, 0, 0)

	// create the "updated" action
	updatedAction := shipv1beta1.ActionSpec{
		Name: "updated",
		Webhook: &shipv1beta1.WebhookActionSpec{
			URI:     fmt.Sprintf("http://watch-server.default.svc.cluster.local:3000/v1/updated/%s", watch.ID),
			Payload: "{}",
			Secret:  "not-implemented",
		},
	}
	actions = append(actions, updatedAction)

	// Add the user-defined actions
	for _, notification := range watch.Notifications {
		if notification.Enabled {
			if notification.Webhook != nil {
				action := shipv1beta1.ActionSpec{
					Name: notification.ID,
					Webhook: &shipv1beta1.WebhookActionSpec{
						URI:     fmt.Sprintf("http://watch-server.default.svc.cluster.local:3000/v1/webhook/%s", notification.ID),
						Payload: "{}",
						Secret:  "not-implemented",
					},
				}

				actions = append(actions, action)
			} else if notification.Email != nil {
				action := shipv1beta1.ActionSpec{
					Name: notification.ID,
					Webhook: &shipv1beta1.WebhookActionSpec{
						URI:     fmt.Sprintf("http://watch-server.default.svc.cluster.local:3000/v1/email/%s", notification.ID),
						Payload: "{}",
						Secret:  "not-implemented",
					},
				}

				actions = append(actions, action)
			}
		}
	}

	shipWatch := shipv1beta1.ShipWatch{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "ShipWatch",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      watch.ID,
			Namespace: namespaceName,
		},
		Spec: shipv1beta1.ShipWatchSpec{
			State: shipv1beta1.StateSpec{
				ValueFrom: shipv1beta1.ShipWatchValueFromSpec{
					SecretKeyRef: shipv1beta1.SecretKeyRef{
						Name: watch.ID,
						Key:  "state.json",
					},
				},
			},
			Images:      images,
			Actions:     actions,
			Environment: []corev1.EnvVar{{Name: "GITHUB_TOKEN", Value: githubToken}},
		},
	}

	return &stateSecret, &shipWatch
}

func networkPolicySpec(watch *types.Watch) *networkv1.NetworkPolicy {
	labels := make(map[string]string)
	labels["shipcloud-role"] = "watch"

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
			Name:      watch.Namespace(),
			Namespace: watch.Namespace(),
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

// returns whether a namespace was created and any error that may have occurred
func (w *Worker) ensureNamespace(namespace *corev1.Namespace) (bool, error) {
	_, err := w.K8sClient.CoreV1().Namespaces().Get(namespace.Name, metav1.GetOptions{})
	if apierrors.IsNotFound(err) {
		if _, err := w.K8sClient.CoreV1().Namespaces().Create(namespace); err != nil {
			return true, errors.Wrap(err, "create namespace")
		}
		return true, nil
	} else if err != nil {
		level.Error(w.Logger).Log("event", "ensureNamespace", "get namespace", namespace.Name, "err", err)
		return false, errors.Wrap(err, "get namespace")
	}

	return false, nil
}

func (w *Worker) ensureNetworkPolicy(networkPolicy *networkv1.NetworkPolicy) error {
	debug := level.Debug(log.With(w.Logger, "method", "updateworker.Worker.ensureNamespace"))

	_, err := w.K8sClient.NetworkingV1().NetworkPolicies(networkPolicy.Namespace).Get(networkPolicy.Name, metav1.GetOptions{})
	if apierrors.IsNotFound(err) {
		debug.Log("event", "create networkPolicy", "networkPolicy", networkPolicy.Name)
		if _, err := w.K8sClient.NetworkingV1().NetworkPolicies(networkPolicy.Namespace).Create(networkPolicy); err != nil {
			return errors.Wrap(err, "create networkPolicy")
		}
	}

	return nil
}

func (w *Worker) ensureSecret(secret *corev1.Secret) error {
	_, err := w.K8sClient.CoreV1().Secrets(secret.Namespace).Get(secret.Name, metav1.GetOptions{})
	if apierrors.IsNotFound(err) {
		if _, err := w.K8sClient.CoreV1().Secrets(secret.Namespace).Create(secret); err != nil {
			return errors.Wrap(err, "create secret")
		}
	}

	return nil
}

func (w *Worker) ensureShipwatch(shipwatch *shipv1beta1.ShipWatch) error {
	currentWatch, err := w.ShipK8sClient.ShipV1beta1().ShipWatches(shipwatch.Namespace).Get(shipwatch.Name, metav1.GetOptions{})
	if apierrors.IsNotFound(err) {
		if _, err := w.ShipK8sClient.ShipV1beta1().ShipWatches(shipwatch.Namespace).Create(shipwatch); err != nil {
			return errors.Wrap(err, "create shipwatch")
		}
	} else if err != nil {
		level.Error(w.Logger).Log("event", "ensureShipwatch", "get shipwatch", shipwatch.Name, "err", err)
		return errors.Wrap(err, "get shipwatch")
	} else {
		shipwatch.ObjectMeta = currentWatch.ObjectMeta

		_, err = w.ShipK8sClient.ShipV1beta1().ShipWatches(shipwatch.Namespace).Update(shipwatch)
		if err != nil {
			level.Error(w.Logger).Log("event", "ensureShipwatch", "update shipwatch", shipwatch.Name, "err", err)
			return errors.Wrap(err, "update shipwatch")
		}
	}

	return nil
}

func (w *Worker) ensureRole(role *rbacv1.Role) error {
	_, err := w.K8sClient.RbacV1().Roles(role.Namespace).Get(role.Name, metav1.GetOptions{})
	if apierrors.IsNotFound(err) {
		if _, err := w.K8sClient.RbacV1().Roles(role.Namespace).Create(role); err != nil {
			return errors.Wrap(err, "create role")
		}
	}

	return nil
}

func (w *Worker) ensureRoleBinding(roleBinding *rbacv1.RoleBinding) error {
	_, err := w.K8sClient.RbacV1().RoleBindings(roleBinding.Namespace).Get(roleBinding.Name, metav1.GetOptions{})
	if apierrors.IsNotFound(err) {
		if _, err := w.K8sClient.RbacV1().RoleBindings(roleBinding.Namespace).Create(roleBinding); err != nil {
			return errors.Wrap(err, "create rolebinding")
		}
	}

	return nil
}
