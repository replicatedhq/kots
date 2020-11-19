package identity

import (
	"bytes"
	"context"
	"fmt"
	"path"
	"text/template"

	dexstorage "github.com/dexidp/dex/storage"
	dexk8sstorage "github.com/dexidp/dex/storage/kubernetes"
	"github.com/ghodss/yaml"
	"github.com/pkg/errors"
	identitytypes "github.com/replicatedhq/kots/pkg/identity/types"
	dextypes "github.com/replicatedhq/kots/pkg/identity/types/dex"
	ingresstypes "github.com/replicatedhq/kots/pkg/ingress/types"
	kotsadmtypes "github.com/replicatedhq/kots/pkg/kotsadm/types"
	"github.com/replicatedhq/kots/pkg/logger"
	"github.com/segmentio/ksuid"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	extensionsv1beta1 "k8s.io/api/extensions/v1beta1"
	rbacv1 "k8s.io/api/rbac/v1"
	apiextensionsv1beta1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	apiextensionsclientset "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	kuberneteserrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/client-go/kubernetes"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
)

const (
	DexDeploymentName, DexServiceName, DexIngressName      = "kotsadm-dex", "kotsadm-dex", "kotsadm-dex"
	DexServiceAccountName, DexRoleName, DexRoleBindingName = "kotsadm-dex", "kotsadm-dex", "kotsadm-dex"
	DexSecretName                                          = "kotsadm-dex"
)

var (
	DexAdditionalLabels = map[string]string{
		KotsIdentityLabelKey: KotsIdentityLabelValue,
	}
)

func Deploy(ctx context.Context, logger *logger.Logger, clientset kubernetes.Interface, namespace string, identityConfig identitytypes.Config, kotsIngressConfig ingresstypes.Config) error {
	if err := DeployCRDs(ctx, logger, clientset); err != nil {
		// Dex will deploy this if it has permissions
		logger.Error(errors.Wrap(err, "failed to deploy crds"))
	}
	if err := DeployServiceAccount(ctx, logger, clientset, namespace); err != nil {
		return errors.Wrap(err, "failed to deploy service account")
	}
	if err := ensureSecret(ctx, clientset, namespace, identityConfig, kotsIngressConfig); err != nil {
		return errors.Wrap(err, "failed to ensure secret")
	}
	if err := ensureDeployment(ctx, clientset, namespace); err != nil {
		return errors.Wrap(err, "failed to ensure deployment")
	}
	if err := ensureService(ctx, clientset, namespace); err != nil {
		return errors.Wrap(err, "failed to ensure service")
	}
	if err := ensureIngress(ctx, clientset, namespace, identityConfig.IngressConfig); err != nil {
		return errors.Wrap(err, "failed to ensure ingress")
	}
	return nil
}

func DeployServiceAccount(ctx context.Context, logger *logger.Logger, clientset kubernetes.Interface, namespace string) error {
	if err := ensureServiceAccount(ctx, clientset, namespace); err != nil {
		return err
	}
	if err := ensureRole(ctx, clientset, namespace); err != nil {
		return err
	}
	if err := ensureRoleBinding(ctx, clientset, namespace); err != nil {
		return err
	}
	return nil
}

func DeployCRDs(ctx context.Context, logger *logger.Logger, clientset kubernetes.Interface) error {
	cfg, err := config.GetConfig()
	if err != nil {
		return errors.Wrap(err, "failed to get cluster config")
	}

	apiextensionsclientset, err := apiextensionsclientset.NewForConfig(cfg)
	if err != nil {
		return errors.Wrap(err, "failed to create apiextensions clientset")
	}

	for _, crd := range DexCustomResourceDefinitions {
		if err := ensureCRD(ctx, apiextensionsclientset, crd); err != nil {
			return err
		}
	}

	// Dex will automatically wait for crds
	return nil
}

func ensureSecret(ctx context.Context, clientset kubernetes.Interface, namespace string, identityConfig identitytypes.Config, kotsIngressConfig ingresstypes.Config) error {
	secret, err := secretResource(DexSecretName, identityConfig, kotsIngressConfig)
	if err != nil {
		return errors.Wrap(err, "failed to get secret resource")
	}

	existingSecret, err := clientset.CoreV1().Secrets(namespace).Get(ctx, DexSecretName, metav1.GetOptions{})
	if err != nil {
		if !kuberneteserrors.IsNotFound(err) {
			return errors.Wrap(err, "failed to get existing secret")
		}

		_, err = clientset.CoreV1().Secrets(namespace).Create(ctx, secret, metav1.CreateOptions{})
		if err != nil {
			return errors.Wrap(err, "failed to create secret")
		}

		return nil
	}

	existingSecret = updateSecret(existingSecret, secret)

	_, err = clientset.CoreV1().Secrets(namespace).Update(ctx, existingSecret, metav1.UpdateOptions{})
	if err != nil {
		return errors.Wrap(err, "failed to update secret")
	}

	return nil
}

func secretResource(secretName string, identityConfig identitytypes.Config, ingressConfig ingresstypes.Config) (*corev1.Secret, error) {
	config := dextypes.Config{
		Logger: dextypes.Logger{
			Level:  "debug",
			Format: "text",
		},
		Issuer: dexIssuerURL(identityConfig.IngressConfig),
		Storage: dextypes.Storage{
			Type: "kubernetes",
			Config: dexk8sstorage.Config{
				InCluster: true,
			},
		},
		Web: dextypes.Web{
			HTTP: fmt.Sprintf("http://%s", path.Join(identityConfig.IngressConfig.Host, identityConfig.IngressConfig.IngressPath(), "/dex")),
		},
		OAuth2: dextypes.OAuth2{
			SkipApprovalScreen: true,
		},
		StaticClients: []dexstorage.Client{
			{
				ID:     "kotsadm",
				Name:   "kotsadm",
				Secret: ksuid.New().String(),
				RedirectURIs: []string{
					fmt.Sprintf("http://%s", path.Join(ingressConfig.Host, ingressConfig.KotsadmPath(), "api/v1/oidc/login/callback")),
				},
			},
		},
		EnablePasswordDB: false,
	}

	if len(identityConfig.DexConnectors) > 0 {
		config.StaticConnectors = identityConfig.DexConnectors
	}

	if err := config.Validate(); err != nil {
		return nil, errors.Wrap(err, "invalid dex config")
	}

	marshalledConfig, err := yaml.Marshal(config)
	if err != nil {
		return nil, errors.Wrap(err, "failed to marshal dex config")
	}

	buf := bytes.NewBuffer(nil)
	t, err := template.New(secretName).Parse(string(marshalledConfig))
	if err != nil {
		return nil, errors.Wrap(err, "failed to parse dex config for templating")
	}
	t.Execute(buf, map[string]string{
		"OIDCIdentityRedirectURI": fmt.Sprintf("%s/callback", dexIssuerURL(identityConfig.IngressConfig)),
	})

	return &corev1.Secret{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "Secret",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:   secretName,
			Labels: kotsadmtypes.GetKotsadmLabels(DexAdditionalLabels),
		},
		Data: map[string][]byte{
			"dexConfig.yaml": []byte(marshalledConfig),
		},
	}, nil
}

func updateSecret(existingSecret, desiredSecret *corev1.Secret) *corev1.Secret {
	existingSecret.Data = desiredSecret.Data
	return existingSecret
}

func ensureDeployment(ctx context.Context, clientset kubernetes.Interface, namespace string) error {
	deployment := deploymentResource(DexDeploymentName, DexServiceAccountName)

	existingDeployment, err := clientset.AppsV1().Deployments(namespace).Get(ctx, DexDeploymentName, metav1.GetOptions{})
	if err != nil {
		if !kuberneteserrors.IsNotFound(err) {
			return errors.Wrap(err, "failed to get existing deployment")
		}

		_, err = clientset.AppsV1().Deployments(namespace).Create(ctx, deployment, metav1.CreateOptions{})
		if err != nil {
			return errors.Wrap(err, "failed to create deployment")
		}

		return nil
	}

	existingDeployment = updateDeployment(existingDeployment, deployment)

	_, err = clientset.AppsV1().Deployments(namespace).Update(ctx, existingDeployment, metav1.UpdateOptions{})
	if err != nil {
		return errors.Wrap(err, "failed to update deployment")
	}

	return nil
}

var (
	dexCPUResource    = resource.MustParse("100m")
	dexMemoryResource = resource.MustParse("50Mi")
)

func deploymentResource(deploymentName, serviceAccountName string) *appsv1.Deployment {
	replicas := int32(2)
	volume := configSecretVolume()
	return &appsv1.Deployment{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "apps/v1",
			Kind:       "Deployment",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:   deploymentName,
			Labels: kotsadmtypes.GetKotsadmLabels(DexAdditionalLabels),
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: &replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"app": "dex",
				},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"app": "dex",
					},
				},
				Spec: corev1.PodSpec{
					ServiceAccountName: serviceAccountName,
					Containers: []corev1.Container{
						{
							// TODO: airgap registry
							// fmt.Sprintf("%s/kotsadm:%s", kotsadmRegistry(deployOptions.KotsadmOptions), kotsadmTag(deployOptions.KotsadmOptions)),
							Image:           "quay.io/dexidp/dex:v2.26.0",
							ImagePullPolicy: corev1.PullAlways,
							Name:            "dex",
							Command:         []string{"/usr/local/bin/dex", "serve", "/etc/dex/cfg/dexConfig.yaml"},
							Ports: []corev1.ContainerPort{
								{Name: "http", ContainerPort: 5556},
							},
							VolumeMounts: []corev1.VolumeMount{
								{Name: volume.Name, MountPath: "/etc/dex/cfg"},
							},
							Resources: corev1.ResourceRequirements{
								Limits: corev1.ResourceList{
									"cpu":    dexCPUResource,
									"memory": dexMemoryResource,
								},
								Requests: corev1.ResourceList{
									"cpu":    dexCPUResource,
									"memory": dexMemoryResource,
								},
							},
						},
					},
					Volumes: []corev1.Volume{
						volume,
					},
				},
			},
		},
	}
}

func configSecretVolume() corev1.Volume {
	return corev1.Volume{
		Name: "config",
		VolumeSource: corev1.VolumeSource{
			Secret: &corev1.SecretVolumeSource{
				SecretName: DexSecretName,
			},
		},
	}
}

func updateDeployment(existingDeployment, desiredDeployment *appsv1.Deployment) *appsv1.Deployment {
	if len(existingDeployment.Spec.Template.Spec.Containers) == 0 {
		// wtf
		return desiredDeployment
	}

	existingDeployment.Spec.Template.Spec.Containers[0].Image = desiredDeployment.Spec.Template.Spec.Containers[0].Image

	existingDeployment = updateDeploymentConfigSecretVolume(existingDeployment, desiredDeployment)

	return existingDeployment
}

func updateDeploymentConfigSecretVolume(existingDeployment *appsv1.Deployment, desiredDeployment *appsv1.Deployment) *appsv1.Deployment {
	if len(existingDeployment.Spec.Template.Spec.Containers) == 0 {
		return desiredDeployment
	}

	newConfigSecretVolume := configSecretVolume()
	newConfigSecretVolumeMount := corev1.VolumeMount{Name: newConfigSecretVolume.Name, MountPath: "/etc/dex/cfg"}

	var existingSecretVolumeName string
	for i, volumeMount := range existingDeployment.Spec.Template.Spec.Containers[0].VolumeMounts {
		if volumeMount.MountPath == "/etc/dex/cfg" {
			existingSecretVolumeName = volumeMount.Name
			existingDeployment.Spec.Template.Spec.Containers[0].VolumeMounts[i] = newConfigSecretVolumeMount
			break
		}
	}
	if existingSecretVolumeName != "" {
		for i, volume := range existingDeployment.Spec.Template.Spec.Volumes {
			if volume.Name == existingSecretVolumeName {
				existingDeployment.Spec.Template.Spec.Volumes[i] = newConfigSecretVolume
			}
		}
		return existingDeployment
	}

	existingDeployment.Spec.Template.Spec.Containers[0].VolumeMounts =
		append(existingDeployment.Spec.Template.Spec.Containers[0].VolumeMounts, newConfigSecretVolumeMount)
	existingDeployment.Spec.Template.Spec.Volumes =
		append(existingDeployment.Spec.Template.Spec.Volumes, newConfigSecretVolume)

	return existingDeployment
}

func ensureService(ctx context.Context, clientset kubernetes.Interface, namespace string) error {
	_, err := clientset.CoreV1().Services(namespace).Get(ctx, DexServiceName, metav1.GetOptions{})
	if err != nil {
		if !kuberneteserrors.IsNotFound(err) {
			return errors.Wrap(err, "failed to get existing service")
		}

		_, err = clientset.CoreV1().Services(namespace).Create(ctx, serviceResource(DexServiceName), metav1.CreateOptions{})
		if err != nil {
			return errors.Wrap(err, "failed to create service")
		}

		return nil
	}

	// no patch necessary

	return nil
}

func serviceResource(serviceName string) *corev1.Service {
	return &corev1.Service{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "Service",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:   serviceName,
			Labels: kotsadmtypes.GetKotsadmLabels(DexAdditionalLabels),
		},
		Spec: corev1.ServiceSpec{
			Type: corev1.ServiceTypeClusterIP,
			Selector: map[string]string{
				"app": "dex",
			},
			Ports: []corev1.ServicePort{
				{Name: "http", Port: 5556},
			},
		},
	}
}

func ensureServiceAccount(ctx context.Context, clientset kubernetes.Interface, namespace string) error {
	_, err := clientset.CoreV1().ServiceAccounts(namespace).Get(ctx, DexServiceAccountName, metav1.GetOptions{})
	if err != nil {
		if !kuberneteserrors.IsNotFound(err) {
			return errors.Wrap(err, "failed to get existing service account")
		}

		_, err = clientset.CoreV1().ServiceAccounts(namespace).Create(ctx, serviceAccountResource(DexServiceAccountName), metav1.CreateOptions{})
		if err != nil {
			return errors.Wrap(err, "failed to create service account")
		}

		return nil
	}

	// no patch necessary

	return nil
}

func serviceAccountResource(serviceAccountName string) *corev1.ServiceAccount {
	return &corev1.ServiceAccount{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "ServiceAccount",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:   serviceAccountName,
			Labels: kotsadmtypes.GetKotsadmLabels(DexAdditionalLabels),
		},
	}
}

func ensureRole(ctx context.Context, clientset kubernetes.Interface, namespace string) error {
	_, err := clientset.RbacV1().Roles(namespace).Get(ctx, DexRoleName, metav1.GetOptions{})
	if err != nil {
		if !kuberneteserrors.IsNotFound(err) {
			return errors.Wrap(err, "failed to get existing role")
		}

		_, err = clientset.RbacV1().Roles(namespace).Create(ctx, roleResource(DexRoleName), metav1.CreateOptions{})
		if err != nil {
			return errors.Wrap(err, "failed to create role")
		}

		return nil
	}

	// no patch necessary

	return nil
}

func roleResource(roleName string) *rbacv1.Role {
	return &rbacv1.Role{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "rbac.authorization.k8s.io/v1",
			Kind:       "Role",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:   roleName,
			Labels: kotsadmtypes.GetKotsadmLabels(DexAdditionalLabels),
		},
		Rules: []rbacv1.PolicyRule{
			{
				APIGroups: []string{"dex.coreos.com"}, // API group created by dex
				Resources: []string{"*"},
				Verbs:     []string{"*"},
			},
			{
				// This will no longer be needed if kots deploys dex crds
				APIGroups: []string{"apiextensions.k8s.io"},
				Resources: []string{"customresourcedefinitions"},
				Verbs:     []string{"create"}, // To manage its own resources, dex must be able to create customresourcedefinitions
			},
		},
	}
}

func ensureRoleBinding(ctx context.Context, clientset kubernetes.Interface, namespace string) error {
	_, err := clientset.RbacV1().RoleBindings(namespace).Get(ctx, DexRoleBindingName, metav1.GetOptions{})
	if err != nil {
		if !kuberneteserrors.IsNotFound(err) {
			return errors.Wrap(err, "failed to get existing role binding")
		}

		roleBinding := roleBindingResource(DexRoleBindingName, DexRoleName, DexServiceAccountName, namespace)
		_, err = clientset.RbacV1().RoleBindings(namespace).Create(ctx, roleBinding, metav1.CreateOptions{})
		if err != nil {
			return errors.Wrap(err, "failed to create role binding")
		}

		return nil
	}

	// no patch necessary

	return nil
}

func roleBindingResource(roleBindingName, roleName, serviceAccountName, serviceAccountNamespace string) *rbacv1.RoleBinding {
	return &rbacv1.RoleBinding{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "rbac.authorization.k8s.io/v1",
			Kind:       "RoleBinding",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:   roleBindingName,
			Labels: kotsadmtypes.GetKotsadmLabels(DexAdditionalLabels),
		},
		RoleRef: rbacv1.RoleRef{
			APIGroup: "rbac.authorization.k8s.io",
			Kind:     "Role",
			Name:     roleName,
		},
		Subjects: []rbacv1.Subject{
			{
				Kind:      "ServiceAccount",
				Name:      serviceAccountName,
				Namespace: serviceAccountNamespace,
			},
		},
	}
}

func ensureCRD(ctx context.Context, clientset apiextensionsclientset.Interface, crd apiextensionsv1beta1.CustomResourceDefinition) error {
	_, err := clientset.ApiextensionsV1beta1().CustomResourceDefinitions().Get(ctx, crd.Name, metav1.GetOptions{})
	if err != nil {
		if !kuberneteserrors.IsNotFound(err) {
			return errors.Wrapf(err, "failed to get existing crd %s", crd.Spec.Names.Plural)
		}

		_, err = clientset.ApiextensionsV1beta1().CustomResourceDefinitions().Create(ctx, &crd, metav1.CreateOptions{})
		if err != nil {
			return errors.Wrapf(err, "failed to create crd %s", crd.Spec.Names.Plural)
		}

		return nil
	}

	// no patch necessary

	return nil
}

func ensureIngress(ctx context.Context, clientset kubernetes.Interface, namespace string, config identitytypes.IngressConfig) error {
	existingIngress, err := clientset.ExtensionsV1beta1().Ingresses(namespace).Get(ctx, DexIngressName, metav1.GetOptions{})
	if err != nil {
		if !kuberneteserrors.IsNotFound(err) {
			return errors.Wrap(err, "failed to get existing dex ingress")
		}

		_, err = clientset.ExtensionsV1beta1().Ingresses(namespace).Create(ctx, ingressResource(namespace, config), metav1.CreateOptions{})
		if err != nil {
			return errors.Wrap(err, "failed to create dex ingress")
		}

		return nil
	}

	existingIngress = updateIngress(existingIngress, namespace, config)

	_, err = clientset.ExtensionsV1beta1().Ingresses(namespace).Update(ctx, existingIngress, metav1.UpdateOptions{})
	if err != nil {
		return errors.Wrap(err, "failed to update dex ingress")
	}

	return nil
}

func ingressResource(namespace string, config identitytypes.IngressConfig) *extensionsv1beta1.Ingress {
	ingress := &extensionsv1beta1.Ingress{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1beta1",
			Kind:       "Ingress",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      DexIngressName,
			Namespace: namespace,
			Labels:    kotsadmtypes.GetKotsadmLabels(DexAdditionalLabels),
		},
		Spec: extensionsv1beta1.IngressSpec{
			Rules: []extensionsv1beta1.IngressRule{
				{
					IngressRuleValue: extensionsv1beta1.IngressRuleValue{
						HTTP: &extensionsv1beta1.HTTPIngressRuleValue{
							Paths: []extensionsv1beta1.HTTPIngressPath{
								{
									Path: config.IngressPath(),
									Backend: extensionsv1beta1.IngressBackend{
										ServiceName: DexServiceName,
										ServicePort: intstr.IntOrString{
											IntVal: 5556,
										},
									},
								},
							},
						},
					},
				},
			},
		},
	}

	if config.Annotations != nil {
		ingress.ObjectMeta.Annotations = config.Annotations
	}

	if config.Host != "" {
		ingress.Spec.Rules[0].Host = config.Host
	}

	if len(config.TLS) > 0 {
		ingress.Spec.TLS = config.TLS
	}

	return ingress
}

func updateIngress(existingIngress *extensionsv1beta1.Ingress, namespace string, config identitytypes.IngressConfig) *extensionsv1beta1.Ingress {
	desiredIngress := ingressResource(namespace, config)
	existingIngress.Spec.Rules = desiredIngress.Spec.Rules
	return existingIngress
}

func dexIssuerURL(ingressConfig identitytypes.IngressConfig) string {
	return fmt.Sprintf("http://%s", path.Join(ingressConfig.Host, ingressConfig.IngressPath(), "dex"))
}
