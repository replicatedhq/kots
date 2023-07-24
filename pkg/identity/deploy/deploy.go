package deploy

import (
	"context"
	"crypto/md5"
	"fmt"
	"net/url"
	"path"

	"github.com/pkg/errors"
	"github.com/replicatedhq/kots/pkg/identity/types"
	"github.com/replicatedhq/kots/pkg/image"
	"github.com/replicatedhq/kots/pkg/ingress"
	kotsadmtypes "github.com/replicatedhq/kots/pkg/kotsadm/types"
	kotsadmversion "github.com/replicatedhq/kots/pkg/kotsadm/version"
	"github.com/replicatedhq/kots/pkg/template"
	kotsv1beta1 "github.com/replicatedhq/kotskinds/apis/kots/v1beta1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	kuberneteserrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8stypes "k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/client-go/kubernetes"
	"k8s.io/utils/pointer"
)

var (
	KotsIdentityLabelKey = "kots.io/identity"
)

type Options struct {
	NamePrefix         string
	Namespace          string
	IdentitySpec       kotsv1beta1.IdentitySpec
	IdentityConfigSpec kotsv1beta1.IdentityConfigSpec
	IsOpenShift        bool
	ImageRewriteFn     kotsadmversion.ImageRewriteFunc
	ProxyEnv           map[string]string
	AdditionalLabels   map[string]string
	Builder            *template.Builder
}

func Deploy(ctx context.Context, clientset kubernetes.Interface, options Options) error {
	issuerURL, err := dexIssuerURL(options.IdentitySpec, options.Builder)
	if err != nil {
		return errors.Wrap(err, "failed to get dex issuer url")
	}
	dexConfig, err := getDexConfig(ctx, issuerURL, options)
	if err != nil {
		return errors.Wrap(err, "failed to get dex config")
	}
	// TODO (salah): make this work with minimal rbac
	if err := ensureNamespace(ctx, clientset, options); err != nil {
		return errors.Wrap(err, "failed to ensure namespace")
	}
	if err := ensureClusterRole(ctx, clientset, options); err != nil {
		return errors.Wrap(err, "failed to ensure cluster role")
	}
	if err := ensureServiceAccount(ctx, clientset, options); err != nil {
		return errors.Wrap(err, "failed to ensure service account")
	}
	if err := ensureClusterRoleBinding(ctx, clientset, options); err != nil {
		return errors.Wrap(err, "failed to ensure cluster role binding")
	}
	if err := ensureClientSecret(ctx, clientset, options); err != nil {
		return errors.Wrap(err, "failed to ensure client secret")
	}
	if err := ensureSecret(ctx, clientset, dexConfig, options); err != nil {
		return errors.Wrap(err, "failed to ensure secret")
	}
	if err := ensureDexThemeConfigMap(ctx, clientset, options); err != nil {
		return errors.Wrap(err, "failed to ensure dex theme config map")
	}
	if err := ensureDeployment(ctx, clientset, issuerURL, dexConfig, options); err != nil {
		return errors.Wrap(err, "failed to ensure deployment")
	}
	if err := ensureService(ctx, clientset, options); err != nil {
		return errors.Wrap(err, "failed to ensure service")
	}
	if err := ensureIngress(ctx, clientset, options); err != nil {
		return errors.Wrap(err, "failed to ensure ingress")
	}
	return nil
}

func Configure(ctx context.Context, clientset kubernetes.Interface, options Options) error {
	issuerURL, err := dexIssuerURL(options.IdentitySpec, options.Builder)
	if err != nil {
		return errors.Wrap(err, "failed to get dex issuer url")
	}
	dexConfig, err := getDexConfig(ctx, issuerURL, options)
	if err != nil {
		return errors.Wrap(err, "failed to get dex config")
	}
	if err := ensureSecret(ctx, clientset, dexConfig, options); err != nil {
		return errors.Wrap(err, "failed to ensure secret")
	}
	if err := patchDeploymentSecret(ctx, clientset, issuerURL, dexConfig, options); err != nil {
		return errors.Wrap(err, "failed to patch deployment secret")
	}
	return nil
}

func AdditionalLabels(namePrefix string, additionalLabels map[string]string) map[string]string {
	next := map[string]string{
		KotsIdentityLabelKey: namePrefix,
	}
	for key, value := range additionalLabels {
		next[key] = value
	}
	return next
}

func ensureNamespace(ctx context.Context, clientset kubernetes.Interface, options Options) error {
	ns := namespaceResource(options)

	_, err := clientset.CoreV1().Namespaces().Get(ctx, ns.Name, metav1.GetOptions{})
	if err != nil {
		if !kuberneteserrors.IsNotFound(err) {
			return errors.Wrap(err, "failed to get namespace")
		}

		_, err = clientset.CoreV1().Namespaces().Create(ctx, ns, metav1.CreateOptions{})
		if err != nil {
			return errors.Wrap(err, "failed to create namespace")
		}
	}

	return nil
}

func namespaceResource(options Options) *corev1.Namespace {
	return &corev1.Namespace{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "Namespace",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:   options.Namespace,
			Labels: kotsadmtypes.GetKotsadmLabels(AdditionalLabels(options.NamePrefix, options.AdditionalLabels)),
		},
	}
}

func ensureClusterRole(ctx context.Context, clientset kubernetes.Interface, options Options) error {
	clusterRole := clusterRoleResource(options)

	_, err := clientset.RbacV1().ClusterRoles().Get(ctx, clusterRole.Name, metav1.GetOptions{})
	if err != nil {
		if !kuberneteserrors.IsNotFound(err) {
			return errors.Wrap(err, "failed to get existing cluster role")
		}

		_, err = clientset.RbacV1().ClusterRoles().Create(ctx, clusterRole, metav1.CreateOptions{})
		if err != nil {
			return errors.Wrap(err, "failed to create cluster role")
		}
	}

	return nil
}

func clusterRoleResource(options Options) *rbacv1.ClusterRole {
	return &rbacv1.ClusterRole{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "rbac.authorization.k8s.io/v1",
			Kind:       "ClusterRole",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:   types.ClusterRoleName(options.NamePrefix),
			Labels: kotsadmtypes.GetKotsadmLabels(AdditionalLabels(options.NamePrefix, options.AdditionalLabels)),
		},
		Rules: []rbacv1.PolicyRule{
			{
				APIGroups: []string{"dex.coreos.com"},
				Resources: []string{"*"},
				Verbs:     metav1.Verbs{"*"},
			},
			{
				APIGroups: []string{"apiextensions.k8s.io"},
				Resources: []string{"customresourcedefinitions"},
				Verbs:     metav1.Verbs{"create"},
			},
		},
	}
}

func ensureServiceAccount(ctx context.Context, clientset kubernetes.Interface, options Options) error {
	sa := serviceAccountResource(options)

	_, err := clientset.CoreV1().ServiceAccounts(options.Namespace).Get(ctx, sa.Name, metav1.GetOptions{})
	if err != nil {
		if !kuberneteserrors.IsNotFound(err) {
			return errors.Wrap(err, "failed to get existing service account")
		}

		_, err = clientset.CoreV1().ServiceAccounts(options.Namespace).Create(ctx, sa, metav1.CreateOptions{})
		if err != nil {
			return errors.Wrap(err, "failed to create service account")
		}
	}

	return nil
}

func serviceAccountResource(options Options) *corev1.ServiceAccount {
	return &corev1.ServiceAccount{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "ServiceAccount",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      types.ServiceAccountName(options.NamePrefix),
			Namespace: options.Namespace,
			Labels:    kotsadmtypes.GetKotsadmLabels(AdditionalLabels(options.NamePrefix, options.AdditionalLabels)),
		},
	}
}

func ensureClusterRoleBinding(ctx context.Context, clientset kubernetes.Interface, options Options) error {
	clusterRoleBinding := clusterRoleBindingResource(options)

	_, err := clientset.RbacV1().ClusterRoleBindings().Get(ctx, clusterRoleBinding.Name, metav1.GetOptions{})
	if err != nil {
		if !kuberneteserrors.IsNotFound(err) {
			return errors.Wrap(err, "failed to get existing cluster role binding")
		}

		_, err = clientset.RbacV1().ClusterRoleBindings().Create(ctx, clusterRoleBinding, metav1.CreateOptions{})
		if err != nil {
			return errors.Wrap(err, "failed to create cluster role binding")
		}
	}

	return nil
}

func clusterRoleBindingResource(options Options) *rbacv1.ClusterRoleBinding {
	return &rbacv1.ClusterRoleBinding{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "rbac.authorization.k8s.io/v1",
			Kind:       "ClusterRoleBinding",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:   types.ClusterRoleBindingName(options.NamePrefix),
			Labels: kotsadmtypes.GetKotsadmLabels(AdditionalLabels(options.NamePrefix, options.AdditionalLabels)),
		},
		Subjects: []rbacv1.Subject{
			{
				Kind:      "ServiceAccount",
				Name:      types.ServiceAccountName(options.NamePrefix),
				Namespace: options.Namespace,
			},
		},
		RoleRef: rbacv1.RoleRef{
			APIGroup: "rbac.authorization.k8s.io",
			Kind:     "ClusterRole",
			Name:     types.ClusterRoleName(options.NamePrefix),
		},
	}
}

func ensureSecret(ctx context.Context, clientset kubernetes.Interface, dexConfig []byte, options Options) error {
	secret := secretResource(dexConfig, options)

	existingSecret, err := clientset.CoreV1().Secrets(options.Namespace).Get(ctx, secret.Name, metav1.GetOptions{})
	if err != nil {
		if !kuberneteserrors.IsNotFound(err) {
			return errors.Wrap(err, "failed to get existing secret")
		}

		_, err = clientset.CoreV1().Secrets(options.Namespace).Create(ctx, secret, metav1.CreateOptions{})
		if err != nil {
			return errors.Wrap(err, "failed to create secret")
		}

		return nil
	}

	existingSecret = updateSecret(existingSecret, secret)

	_, err = clientset.CoreV1().Secrets(options.Namespace).Update(ctx, existingSecret, metav1.UpdateOptions{})
	if err != nil {
		return errors.Wrap(err, "failed to update secret")
	}

	return nil
}

func secretResource(dexConfig []byte, options Options) *corev1.Secret {
	return &corev1.Secret{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "Secret",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      types.SecretName(options.NamePrefix),
			Namespace: options.Namespace,
			Labels:    kotsadmtypes.GetKotsadmLabels(AdditionalLabels(options.NamePrefix, options.AdditionalLabels)),
		},
		Data: map[string][]byte{
			"dexConfig.yaml": dexConfig,
		},
	}
}

func updateSecret(existingSecret, desiredSecret *corev1.Secret) *corev1.Secret {
	existingSecret.Data = desiredSecret.Data
	return existingSecret
}

func ensureDeployment(ctx context.Context, clientset kubernetes.Interface, issuerURL string, marshalledDexConfig []byte, options Options) error {
	configChecksum := fmt.Sprintf("%x", md5.Sum(marshalledDexConfig))

	deployment, err := deploymentResource(issuerURL, configChecksum, options)
	if err != nil {
		return errors.Wrap(err, "failed to get deployment resource")
	}

	existingDeployment, err := clientset.AppsV1().Deployments(options.Namespace).Get(ctx, deployment.Name, metav1.GetOptions{})
	if err != nil {
		if !kuberneteserrors.IsNotFound(err) {
			return errors.Wrap(err, "failed to get existing deployment")
		}

		_, err = clientset.AppsV1().Deployments(options.Namespace).Create(ctx, deployment, metav1.CreateOptions{})
		if err != nil {
			return errors.Wrap(err, "failed to create deployment")
		}

		return nil
	}

	existingDeployment = updateDeployment(options.NamePrefix, existingDeployment, deployment)

	_, err = clientset.AppsV1().Deployments(options.Namespace).Update(ctx, existingDeployment, metav1.UpdateOptions{})
	if err != nil {
		return errors.Wrap(err, "failed to update deployment")
	}

	return nil
}

func patchDeploymentSecret(ctx context.Context, clientset kubernetes.Interface, issuerURL string, marshalledDexConfig []byte, options Options) error {
	configChecksum := fmt.Sprintf("%x", md5.Sum(marshalledDexConfig))

	deployment, err := deploymentResource(issuerURL, configChecksum, options)
	if err != nil {
		return errors.Wrap(err, "failed to get deployment resource")
	}

	patch := fmt.Sprintf(`{"spec":{"template":{"metadata":{"annotations":{"kots.io/dex-secret-checksum":"%s"}}}}}`, deployment.Spec.Template.ObjectMeta.Annotations["kots.io/dex-secret-checksum"])

	// TODO (ethan): patch readiness and liveness checks if issuer url changes

	_, err = clientset.AppsV1().Deployments(options.Namespace).Patch(ctx, deployment.Name, k8stypes.StrategicMergePatchType, []byte(patch), metav1.PatchOptions{})
	if err != nil {
		return errors.Wrap(err, "failed to patch deployment")
	}

	return nil
}

var (
	dexCPUResource    = resource.MustParse("100m")
	dexMemoryResource = resource.MustParse("50Mi")
)

func deploymentResource(issuerURL, configChecksum string, options Options) (*appsv1.Deployment, error) {
	// TODO: use GetAdminConsoleImage function
	dexVersion, err := image.GetTag(image.Dex)
	if err != nil {
		return nil, err
	}
	image := fmt.Sprintf("kotsadm/dex:%s", dexVersion)
	imagePullSecrets := []corev1.LocalObjectReference{}
	if options.ImageRewriteFn != nil {
		var err error
		image, imagePullSecrets, err = options.ImageRewriteFn(image, false)
		if err != nil {
			return nil, errors.Wrap(err, "failed to rewrite image")
		}
	}

	var securityContext corev1.PodSecurityContext
	if !options.IsOpenShift {
		securityContext = corev1.PodSecurityContext{
			RunAsUser: pointer.Int64Ptr(1001),
		}
	}

	u, err := url.Parse(issuerURL)
	if err != nil {
		return nil, errors.Wrap(err, "failed to parse issuer url")
	}

	env := []corev1.EnvVar{clientSecretEnvVar(options.NamePrefix)}

	// TODO (ethan): this will not really work when kotsadm is not rendering this
	for name, val := range options.ProxyEnv {
		env = append(env, corev1.EnvVar{Name: name, Value: val})
	}

	secretVolume := configSecretVolume(options.NamePrefix)
	themeVolume := dexThemeVolume(options.NamePrefix)

	return &appsv1.Deployment{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "apps/v1",
			Kind:       "Deployment",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      types.DeploymentName(options.NamePrefix),
			Namespace: options.Namespace,
			Labels:    kotsadmtypes.GetKotsadmLabels(AdditionalLabels(options.NamePrefix, options.AdditionalLabels)),
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: pointer.Int32Ptr(2),
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"app": prefixName(options.NamePrefix, "dex"),
				},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"app": prefixName(options.NamePrefix, "dex"),
					},
					Annotations: map[string]string{
						"kots.io/dex-secret-checksum": configChecksum,
					},
				},
				Spec: corev1.PodSpec{
					Affinity: &corev1.Affinity{
						PodAntiAffinity: &corev1.PodAntiAffinity{
							PreferredDuringSchedulingIgnoredDuringExecution: []corev1.WeightedPodAffinityTerm{{
								Weight: 2,
								PodAffinityTerm: corev1.PodAffinityTerm{
									LabelSelector: &metav1.LabelSelector{
										MatchExpressions: []metav1.LabelSelectorRequirement{{
											Key:      "app",
											Operator: metav1.LabelSelectorOpIn,
											Values: []string{
												prefixName(options.NamePrefix, "dex"),
											},
										}},
									},
									TopologyKey: corev1.LabelHostname,
								},
							},
							}},
					},
					SecurityContext:    &securityContext,
					ImagePullSecrets:   imagePullSecrets,
					ServiceAccountName: types.ServiceAccountName(options.NamePrefix),
					Containers: []corev1.Container{
						{
							Image:           image,
							ImagePullPolicy: corev1.PullIfNotPresent,
							Name:            "dex",
							Command:         []string{"/usr/local/bin/dex", "serve", "/etc/dex/cfg/dexConfig.yaml"},
							Ports: []corev1.ContainerPort{
								{Name: "http", ContainerPort: 5556},
							},
							Env: env,
							VolumeMounts: []corev1.VolumeMount{
								{Name: secretVolume.Name, MountPath: "/etc/dex/cfg"},
								{Name: themeVolume.Name, MountPath: "/web/themes/kots"},
							},
							Resources: corev1.ResourceRequirements{
								// Limits: corev1.ResourceList{
								// 	"cpu":    dexCPUResource,
								// 	"memory": dexMemoryResource,
								// },
								Requests: corev1.ResourceList{
									"cpu":    dexCPUResource,
									"memory": dexMemoryResource,
								},
							},
							LivenessProbe: &corev1.Probe{
								ProbeHandler: corev1.ProbeHandler{
									HTTPGet: &corev1.HTTPGetAction{
										Path: path.Join(u.Path, "healthz"),
										Port: intstr.FromInt(5556),
									},
								},
							},
							ReadinessProbe: &corev1.Probe{
								ProbeHandler: corev1.ProbeHandler{
									HTTPGet: &corev1.HTTPGetAction{
										Path: path.Join(u.Path, "healthz"),
										Port: intstr.FromInt(5556),
									},
								},
								InitialDelaySeconds: 5,
								TimeoutSeconds:      1,
							},
						},
					},
					Volumes: []corev1.Volume{
						secretVolume,
						themeVolume,
					},
				},
			},
		},
	}, nil
}

func configSecretVolume(namePrefix string) corev1.Volume {
	return corev1.Volume{
		Name: "config",
		VolumeSource: corev1.VolumeSource{
			Secret: &corev1.SecretVolumeSource{
				SecretName: types.SecretName(namePrefix),
			},
		},
	}
}

func dexThemeVolume(namePrefix string) corev1.Volume {
	return corev1.Volume{
		Name: "theme",
		VolumeSource: corev1.VolumeSource{
			ConfigMap: &corev1.ConfigMapVolumeSource{
				LocalObjectReference: corev1.LocalObjectReference{
					Name: prefixName(namePrefix, "dex-theme"),
				},
				Optional: pointer.Bool(true),
			},
		},
	}
}

func clientSecretEnvVar(namePrefix string) corev1.EnvVar {
	return corev1.EnvVar{
		Name: "DEX_CLIENT_SECRET",
		ValueFrom: &corev1.EnvVarSource{
			SecretKeyRef: &corev1.SecretKeySelector{
				LocalObjectReference: corev1.LocalObjectReference{
					Name: prefixName(namePrefix, "dex-client"),
				},
				Key: "DEX_CLIENT_SECRET",
			},
		},
	}
}

func updateDeployment(namePrefix string, existingDeployment, desiredDeployment *appsv1.Deployment) *appsv1.Deployment {
	if len(existingDeployment.Spec.Template.Spec.Containers) == 0 {
		return desiredDeployment
	}

	existingDeployment.Spec.Template.Spec.ServiceAccountName = desiredDeployment.Spec.Template.Spec.ServiceAccountName

	if existingDeployment.Spec.Template.Annotations == nil {
		existingDeployment.Spec.Template.ObjectMeta.Annotations = map[string]string{}
	}
	existingDeployment.Spec.Template.ObjectMeta.Annotations["kots.io/dex-secret-checksum"] = desiredDeployment.Spec.Template.ObjectMeta.Annotations["kots.io/dex-secret-checksum"]

	existingDeployment.Spec.Template.Spec.Containers[0].Image = desiredDeployment.Spec.Template.Spec.Containers[0].Image
	existingDeployment.Spec.Template.Spec.Containers[0].LivenessProbe = desiredDeployment.Spec.Template.Spec.Containers[0].LivenessProbe
	existingDeployment.Spec.Template.Spec.Containers[0].ReadinessProbe = desiredDeployment.Spec.Template.Spec.Containers[0].ReadinessProbe
	existingDeployment.Spec.Template.Spec.Containers[0].Env = desiredDeployment.Spec.Template.Spec.Containers[0].Env
	existingDeployment.Spec.Template.Spec.Containers[0].EnvFrom = desiredDeployment.Spec.Template.Spec.Containers[0].EnvFrom
	existingDeployment.Spec.Template.Spec.Containers[0].Resources = desiredDeployment.Spec.Template.Spec.Containers[0].Resources

	existingDeployment = updateDeploymentConfigSecretVolume(namePrefix, existingDeployment, desiredDeployment)
	existingDeployment = updateDeploymentClientSecretEnvVar(namePrefix, existingDeployment, desiredDeployment)

	return existingDeployment
}

func updateDeploymentConfigSecretVolume(namePrefix string, existingDeployment *appsv1.Deployment, desiredDeployment *appsv1.Deployment) *appsv1.Deployment {
	if len(existingDeployment.Spec.Template.Spec.Containers) == 0 {
		return desiredDeployment
	}

	newConfigSecretVolume := configSecretVolume(namePrefix)
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

func updateDeploymentClientSecretEnvVar(namePrefix string, existingDeployment *appsv1.Deployment, desiredDeployment *appsv1.Deployment) *appsv1.Deployment {
	if len(existingDeployment.Spec.Template.Spec.Containers) == 0 {
		return desiredDeployment
	}

	newClientSecretEnvVar := clientSecretEnvVar(namePrefix)

	for i, envVar := range existingDeployment.Spec.Template.Spec.Containers[0].Env {
		if envVar.Name == "DEX_CLIENT_SECRET" {
			existingDeployment.Spec.Template.Spec.Containers[0].Env[i] = newClientSecretEnvVar
			return existingDeployment
		}
	}

	existingDeployment.Spec.Template.Spec.Containers[0].Env =
		append(existingDeployment.Spec.Template.Spec.Containers[0].Env, newClientSecretEnvVar)

	return existingDeployment
}

func ensureService(ctx context.Context, clientset kubernetes.Interface, options Options) error {
	service := serviceResource(options)

	existingService, err := clientset.CoreV1().Services(options.Namespace).Get(ctx, service.Name, metav1.GetOptions{})
	if err != nil {
		if !kuberneteserrors.IsNotFound(err) {
			return errors.Wrap(err, "failed to get existing service")
		}

		_, err = clientset.CoreV1().Services(options.Namespace).Create(ctx, service, metav1.CreateOptions{})
		if err != nil {
			return errors.Wrap(err, "failed to create service")
		}

		return nil
	}

	existingService = updateService(existingService, service)

	_, err = clientset.CoreV1().Services(options.Namespace).Update(ctx, existingService, metav1.UpdateOptions{})
	if err != nil {
		return errors.Wrap(err, "failed to update service")
	}

	return nil
}

func serviceResource(options Options) *corev1.Service {
	ingressSpec := options.IdentityConfigSpec.IngressConfig
	serviceType := corev1.ServiceTypeClusterIP
	port := corev1.ServicePort{
		Name:       "http",
		Port:       types.ServicePort(),
		TargetPort: intstr.FromInt(int(types.ServicePort())),
	}
	if ingressSpec.Enabled && ingressSpec.NodePort != nil && ingressSpec.NodePort.Port != 0 {
		port.NodePort = int32(ingressSpec.NodePort.Port)
		serviceType = corev1.ServiceTypeNodePort
	}
	return &corev1.Service{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "Service",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      types.ServiceName(options.NamePrefix),
			Namespace: options.Namespace,
			Labels:    kotsadmtypes.GetKotsadmLabels(AdditionalLabels(options.NamePrefix, options.AdditionalLabels)),
		},
		Spec: corev1.ServiceSpec{
			Type: serviceType,
			Selector: map[string]string{
				"app": prefixName(options.NamePrefix, "dex"),
			},
			Ports: []corev1.ServicePort{
				port,
			},
		},
	}
}

func updateService(existingService, desiredService *corev1.Service) *corev1.Service {
	existingService.Spec.Ports = desiredService.Spec.Ports

	return existingService
}

func ensureIngress(ctx context.Context, clientset kubernetes.Interface, options Options) error {
	ingressSpec := options.IdentityConfigSpec.IngressConfig
	if !ingressSpec.Enabled || ingressSpec.Ingress == nil {
		return deleteIngress(ctx, clientset, options.Namespace, options.NamePrefix)
	}
	dexIngress := ingressResource(options)
	return ingress.EnsureIngress(ctx, clientset, options.Namespace, dexIngress)
}

func ingressResource(options Options) *networkingv1.Ingress {
	ingressSpec := options.IdentityConfigSpec.IngressConfig
	if ingressSpec.Ingress == nil {
		return nil
	}
	return ingress.IngressFromConfig(
		options.Namespace,
		*ingressSpec.Ingress,
		prefixName(options.NamePrefix, "dex"),
		types.ServiceName(options.NamePrefix),
		types.ServicePort(),
		AdditionalLabels(options.NamePrefix, options.AdditionalLabels),
	)
}

func prefixName(prefix, name string) string {
	return fmt.Sprintf("%s-%s", prefix, name)
}
