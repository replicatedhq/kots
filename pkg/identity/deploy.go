package identity

import (
	"bytes"
	"context"
	"crypto/md5"
	"fmt"
	"text/template"

	dexstorage "github.com/dexidp/dex/storage"
	"github.com/ghodss/yaml"
	"github.com/pkg/errors"
	identitytypes "github.com/replicatedhq/kots/pkg/identity/types"
	dextypes "github.com/replicatedhq/kots/pkg/identity/types/dex"
	"github.com/replicatedhq/kots/pkg/ingress"
	ingresstypes "github.com/replicatedhq/kots/pkg/ingress/types"
	"github.com/replicatedhq/kots/pkg/kotsadm/types"
	kotsadmtypes "github.com/replicatedhq/kots/pkg/kotsadm/types"
	kotsadmversion "github.com/replicatedhq/kots/pkg/kotsadm/version"
	"github.com/segmentio/ksuid"
	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	extensionsv1beta1 "k8s.io/api/extensions/v1beta1"
	rbacv1 "k8s.io/api/rbac/v1"
	kuberneteserrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8stypes "k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/client-go/kubernetes"
)

const (
	DexDeploymentName, DexServiceName, DexIngressName      = "kotsadm-dex", "kotsadm-dex", "kotsadm-dex"
	DexServiceAccountName, DexRoleName, DexRoleBindingName = "kotsadm-dex", "kotsadm-dex", "kotsadm-dex"
	DexSecretName                                          = "kotsadm-dex"
	DexPostgresJobName, DexPostgresSecretName              = "kotsadm-dex-pg", "kotsadm-dex-pg"
)

var (
	AdditionalLabels = map[string]string{
		KotsIdentityLabelKey: KotsIdentityLabelValue,
	}
)

func Initialize(ctx context.Context, clientset kubernetes.Interface, namespace string, registryOptions *kotsadmtypes.KotsadmOptions) error {
	if err := ensurePostgresSecret(ctx, clientset, namespace); err != nil {
		return errors.Wrap(err, "failed to ensure postgres secret")
	}
	if err := ensurePostgresJob(ctx, clientset, namespace, registryOptions); err != nil {
		return errors.Wrap(err, "failed to ensure postgres job")
	}
	if err := deployServiceAccount(ctx, clientset, namespace); err != nil {
		return errors.Wrap(err, "failed to deploy service account")
	}
	return nil
}

func Deploy(ctx context.Context, clientset kubernetes.Interface, namespace string, identityConfig identitytypes.Config, ingressConfig ingresstypes.Config, registryOptions *kotsadmtypes.KotsadmOptions) error {
	marshalledDexConfig, err := getDexConfig(ctx, clientset, namespace, identityConfig, ingressConfig)
	if err != nil {
		return errors.Wrap(err, "failed to marshal dex config")
	}
	if err := ensureSecret(ctx, clientset, namespace, marshalledDexConfig); err != nil {
		return errors.Wrap(err, "failed to ensure secret")
	}
	if err := ensureDeployment(ctx, clientset, namespace, marshalledDexConfig, registryOptions); err != nil {
		return errors.Wrap(err, "failed to ensure deployment")
	}
	if err := ensureService(ctx, clientset, namespace, identityConfig.IngressConfig.NodePort); err != nil {
		return errors.Wrap(err, "failed to ensure service")
	}
	if err := ensureIngress(ctx, clientset, namespace, identityConfig.IngressConfig); err != nil {
		return errors.Wrap(err, "failed to ensure ingress")
	}
	return nil
}

func Configure(ctx context.Context, clientset kubernetes.Interface, namespace string, identityConfig identitytypes.Config, ingressConfig ingresstypes.Config) error {
	marshalledDexConfig, err := getDexConfig(ctx, clientset, namespace, identityConfig, ingressConfig)
	if err != nil {
		return errors.Wrap(err, "failed to marshal dex config")
	}
	if err := ensureSecret(ctx, clientset, namespace, marshalledDexConfig); err != nil {
		return errors.Wrap(err, "failed to ensure secret")
	}
	if err := patchDeploymentSecret(ctx, clientset, namespace, marshalledDexConfig); err != nil {
		return errors.Wrap(err, "failed to patch deployment secret")
	}
	return nil
}

func deployServiceAccount(ctx context.Context, clientset kubernetes.Interface, namespace string) error {
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

func ensureSecret(ctx context.Context, clientset kubernetes.Interface, namespace string, marshalledConfig []byte) error {
	secret, err := secretResource(DexSecretName, marshalledConfig)
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

func getExistingDexConfig(ctx context.Context, clientset kubernetes.Interface, namespace string) (*dextypes.Config, error) {
	dexConfig := dextypes.Config{}

	existingSecret, err := clientset.CoreV1().Secrets(namespace).Get(ctx, DexSecretName, metav1.GetOptions{})
	if err != nil {
		if !kuberneteserrors.IsNotFound(err) {
			return nil, errors.Wrap(err, "failed to get existing secret")
		}
		return &dexConfig, nil
	}

	err = yaml.Unmarshal(existingSecret.Data["dexConfig.yaml"], &dexConfig)
	if err != nil {
		return nil, errors.Wrap(err, "failed to unmarshal dexConfig.yaml")
	}

	return &dexConfig, nil
}

func getDexPostgresPassword(ctx context.Context, clientset kubernetes.Interface, namespace string) (string, error) {
	secret, err := clientset.CoreV1().Secrets(namespace).Get(ctx, DexPostgresSecretName, metav1.GetOptions{})
	if err != nil {
		return "", errors.Wrap(err, "failed to get postgress secret")
	}

	return string(secret.Data["password"]), nil
}

func getDexConfig(ctx context.Context, clientset kubernetes.Interface, namespace string, identityConfig identitytypes.Config, ingressConfig ingresstypes.Config) ([]byte, error) {
	existingConfig, err := getExistingDexConfig(ctx, clientset, namespace)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get existing dex config")
	}

	postgresPassword, err := getDexPostgresPassword(ctx, clientset, namespace)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get dex postgres password")
	}

	staticClients := existingConfig.StaticClients
	kotsadmClient := dexstorage.Client{
		ID:     "kotsadm",
		Name:   "kotsadm",
		Secret: ksuid.New().String(),
		RedirectURIs: []string{
			fmt.Sprintf("%s/api/v1/oidc/login/callback", ingress.GetAddress(ingressConfig)),
		},
	}
	foundKotsClient := false
	for i := range staticClients {
		if staticClients[i].ID == "kotsadm" {
			staticClients[i].RedirectURIs = kotsadmClient.RedirectURIs
			foundKotsClient = true
		}
	}
	if !foundKotsClient {
		staticClients = append(staticClients, kotsadmClient)
	}

	config := dextypes.Config{
		Logger: dextypes.Logger{
			Level:  "debug",
			Format: "text",
		},
		Issuer: DexIssuerURL(identityConfig.IngressConfig),
		Storage: dextypes.Storage{
			Type: "postgres",
			Config: dextypes.Postgres{
				NetworkDB: dextypes.NetworkDB{
					Database: "dex",
					User:     "dex",
					Host:     "kotsadm-postgres",
					Password: postgresPassword,
				},
				SSL: dextypes.SSL{
					Mode: "disable", // TODO ssl
				},
			},
		},
		Web: dextypes.Web{
			HTTP: "0.0.0.0:5556",
		},
		OAuth2: dextypes.OAuth2{
			SkipApprovalScreen: true,
		},
		StaticClients:    staticClients,
		EnablePasswordDB: false,
	}

	if len(identityConfig.DexConnectors) > 0 {
		config.StaticConnectors = identityConfig.DexConnectors
	}

	if err := config.Validate(); err != nil {
		return nil, errors.Wrap(err, "failed to validate dex config")
	}

	marshalledConfig, err := yaml.Marshal(config)
	if err != nil {
		return nil, errors.Wrap(err, "failed to marshal dex config")
	}

	buf := bytes.NewBuffer(nil)
	t, err := template.New("kotsadm-dex").Funcs(template.FuncMap{
		"OIDCIdentityCallbackURL": func() string { return DexCallbackURL(identityConfig.IngressConfig) },
	}).Parse(string(marshalledConfig))
	if err != nil {
		return nil, errors.Wrap(err, "failed to parse dex config for templating")
	}
	if err := t.Execute(buf, nil); err != nil {
		return nil, errors.Wrap(err, "failed to execute template")
	}

	return buf.Bytes(), nil
}

func secretResource(secretName string, marshalledConfig []byte) (*corev1.Secret, error) {
	return &corev1.Secret{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "Secret",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:   secretName,
			Labels: kotsadmtypes.GetKotsadmLabels(AdditionalLabels),
		},
		Data: map[string][]byte{
			"dexConfig.yaml": marshalledConfig,
		},
	}, nil
}

func updateSecret(existingSecret, desiredSecret *corev1.Secret) *corev1.Secret {
	existingSecret.Data = desiredSecret.Data
	return existingSecret
}

func ensureDeployment(ctx context.Context, clientset kubernetes.Interface, namespace string, marshalledDexConfig []byte, registryOptions *kotsadmtypes.KotsadmOptions) error {
	configChecksum := fmt.Sprintf("%x", md5.Sum(marshalledDexConfig))

	deployment := deploymentResource(DexDeploymentName, DexServiceAccountName, configChecksum, namespace, registryOptions)

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

func patchDeploymentSecret(ctx context.Context, clientset kubernetes.Interface, namespace string, marshalledDexConfig []byte) error {
	configChecksum := fmt.Sprintf("%x", md5.Sum(marshalledDexConfig))

	deployment := deploymentResource(DexDeploymentName, DexServiceAccountName, configChecksum, namespace, nil)

	patch := fmt.Sprintf(`{"spec":{"template":{"metadata":{"annotations":{"kots.io/dex-secret-checksum":"%s"}}}}}`, deployment.Spec.Template.ObjectMeta.Annotations["kots.io/dex-secret-checksum"])

	_, err := clientset.AppsV1().Deployments(namespace).Patch(ctx, deployment.Name, k8stypes.StrategicMergePatchType, []byte(patch), metav1.PatchOptions{})
	if err != nil {
		return errors.Wrap(err, "failed to patch deployment")
	}

	return nil
}

var (
	dexCPUResource    = resource.MustParse("100m")
	dexMemoryResource = resource.MustParse("50Mi")
)

func deploymentResource(deploymentName, serviceAccountName, configChecksum, namespace string, registryOptions *kotsadmtypes.KotsadmOptions) *appsv1.Deployment {
	replicas := int32(2)
	volume := configSecretVolume()

	image := "quay.io/dexidp/dex:v2.26.0"
	imagePullSecrets := []corev1.LocalObjectReference{}
	if registryOptions != nil {
		if s := kotsadmversion.KotsadmPullSecret(namespace, *registryOptions); s != nil {
			image = fmt.Sprintf("%s/dex:%s", kotsadmversion.KotsadmRegistry(*registryOptions), kotsadmversion.KotsadmTag(*registryOptions))
			imagePullSecrets = []corev1.LocalObjectReference{
				{
					Name: s.ObjectMeta.Name,
				},
			}
		}
	}

	return &appsv1.Deployment{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "apps/v1",
			Kind:       "Deployment",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:   deploymentName,
			Labels: kotsadmtypes.GetKotsadmLabels(AdditionalLabels),
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
					Annotations: map[string]string{
						"kots.io/dex-secret-checksum": configChecksum,
					},
				},
				Spec: corev1.PodSpec{
					ServiceAccountName: serviceAccountName,
					ImagePullSecrets:   imagePullSecrets,
					Containers: []corev1.Container{
						{
							Image:           image,
							ImagePullPolicy: corev1.PullIfNotPresent,
							Name:            "dex",
							Command:         []string{"/usr/local/bin/dex", "serve", "/etc/dex/cfg/dexConfig.yaml"},
							Ports: []corev1.ContainerPort{
								{Name: "http", ContainerPort: 5556},
							},
							VolumeMounts: []corev1.VolumeMount{
								{Name: volume.Name, MountPath: "/etc/dex/cfg"},
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

	if existingDeployment.Spec.Template.Annotations == nil {
		existingDeployment.Spec.Template.ObjectMeta.Annotations = map[string]string{}
	}
	existingDeployment.Spec.Template.ObjectMeta.Annotations["kots.io/dex-secret-checksum"] = desiredDeployment.Spec.Template.ObjectMeta.Annotations["kots.io/dex-secret-checksum"]

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

func ensureService(ctx context.Context, clientset kubernetes.Interface, namespace string, nodePortConfig *ingresstypes.NodePortConfig) error {
	service := serviceResource(DexServiceName, nodePortConfig)

	existingService, err := clientset.CoreV1().Services(namespace).Get(ctx, DexServiceName, metav1.GetOptions{})
	if err != nil {
		if !kuberneteserrors.IsNotFound(err) {
			return errors.Wrap(err, "failed to get existing service")
		}

		_, err = clientset.CoreV1().Services(namespace).Create(ctx, service, metav1.CreateOptions{})
		if err != nil {
			return errors.Wrap(err, "failed to create service")
		}

		return nil
	}

	existingService = updateService(existingService, service)

	_, err = clientset.CoreV1().Services(namespace).Update(ctx, existingService, metav1.UpdateOptions{})
	if err != nil {
		return errors.Wrap(err, "failed to update service")
	}

	return nil
}

func serviceResource(serviceName string, nodePortConfig *ingresstypes.NodePortConfig) *corev1.Service {
	serviceType := corev1.ServiceTypeClusterIP
	port := corev1.ServicePort{
		Name:       "http",
		Port:       5556,
		TargetPort: intstr.IntOrString{Type: intstr.Int, IntVal: 5556},
	}
	if nodePortConfig != nil && nodePortConfig.Port != 0 {
		port.NodePort = int32(nodePortConfig.Port)
		serviceType = corev1.ServiceTypeNodePort
	}
	return &corev1.Service{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "Service",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:   serviceName,
			Labels: kotsadmtypes.GetKotsadmLabels(AdditionalLabels),
		},
		Spec: corev1.ServiceSpec{
			Type: serviceType,
			Selector: map[string]string{
				"app": "dex",
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
			Labels: kotsadmtypes.GetKotsadmLabels(AdditionalLabels),
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
			Labels: kotsadmtypes.GetKotsadmLabels(AdditionalLabels),
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
			Labels: kotsadmtypes.GetKotsadmLabels(AdditionalLabels),
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

func ensurePostgresSecret(ctx context.Context, clientset kubernetes.Interface, namespace string) error {
	secret := postgresSecretResource(DexPostgresSecretName)

	_, err := clientset.CoreV1().Secrets(namespace).Get(ctx, DexPostgresSecretName, metav1.GetOptions{})
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

	// no patch needed

	return nil
}

func postgresSecretResource(secretName string) *corev1.Secret {
	generatedPassword := ksuid.New().String()

	return &corev1.Secret{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "Secret",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:   secretName,
			Labels: kotsadmtypes.GetKotsadmLabels(AdditionalLabels),
		},
		Data: map[string][]byte{
			"password": []byte(generatedPassword),
		},
	}
}

func ensurePostgresJob(ctx context.Context, clientset kubernetes.Interface, namespace string, registryOptions *kotsadmtypes.KotsadmOptions) error {
	job := postgresJobResource(DexPostgresJobName, namespace, registryOptions)

	_, err := clientset.BatchV1().Jobs(namespace).Get(ctx, DexPostgresJobName, metav1.GetOptions{})
	if err != nil {
		if !kuberneteserrors.IsNotFound(err) {
			return errors.Wrap(err, "failed to get existing postgres job")
		}

		_, err = clientset.BatchV1().Jobs(namespace).Create(ctx, job, metav1.CreateOptions{})
		if err != nil {
			return errors.Wrap(err, "failed to create job")
		}

		return nil
	}

	// TODO: recreate job?

	return nil
}

func postgresJobResource(name string, namespace string, registryOptions *kotsadmtypes.KotsadmOptions) *batchv1.Job {
	image := "postgres:10.7"
	imagePullSecrets := []corev1.LocalObjectReference{}
	if registryOptions != nil {
		if s := kotsadmversion.KotsadmPullSecret(namespace, *registryOptions); s != nil {
			image = fmt.Sprintf("%s/postgres:%s", kotsadmversion.KotsadmRegistry(*registryOptions), kotsadmversion.KotsadmTag(*registryOptions))
			imagePullSecrets = []corev1.LocalObjectReference{
				{
					Name: s.ObjectMeta.Name,
				},
			}
		}
	}

	job := &batchv1.Job{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "batch/v1",
			Kind:       "Job",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:   name,
			Labels: types.GetKotsadmLabels(),
		},
		Spec: batchv1.JobSpec{
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Name:   name,
					Labels: types.GetKotsadmLabels(),
				},
				Spec: corev1.PodSpec{
					RestartPolicy:    "OnFailure",
					ImagePullSecrets: imagePullSecrets,
					Containers: []corev1.Container{
						{
							Image:           image,
							ImagePullPolicy: corev1.PullIfNotPresent,
							Name:            name,
							Command: []string{
								"psql",
							},
							Args: []string{
								"-h",
								"kotsadm-postgres",
								"-U",
								"kotsadm",
								"-c",
								"CREATE DATABASE dex;",
								"-c",
								"CREATE USER dex;",
								"-c",
								"ALTER USER dex WITH PASSWORD '$(DEX_PG_PASSWORD)';",
								"-c",
								"GRANT ALL PRIVILEGES ON DATABASE dex TO dex;",
							},
							Env: []corev1.EnvVar{
								{
									Name: "PGPASSWORD",
									ValueFrom: &corev1.EnvVarSource{
										SecretKeyRef: &corev1.SecretKeySelector{
											LocalObjectReference: corev1.LocalObjectReference{
												Name: "kotsadm-postgres",
											},
											Key: "password",
										},
									},
								},
								{
									Name: "DEX_PG_PASSWORD",
									ValueFrom: &corev1.EnvVarSource{
										SecretKeyRef: &corev1.SecretKeySelector{
											LocalObjectReference: corev1.LocalObjectReference{
												Name: DexPostgresSecretName,
											},
											Key: "password",
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

	return job
}

func ensureIngress(ctx context.Context, clientset kubernetes.Interface, namespace string, ingressConfig ingresstypes.Config) error {
	if ingressConfig.Ingress == nil {
		return deleteIngress(ctx, clientset, namespace)
	}
	dexIngress := ingressResource(namespace, ingressConfig)
	return ingress.EnsureIngress(ctx, clientset, namespace, dexIngress)
}

func ingressResource(namespace string, ingressConfig ingresstypes.Config) *extensionsv1beta1.Ingress {
	return ingress.IngressFromConfig(*ingressConfig.Ingress, DexIngressName, DexServiceName, 5556, AdditionalLabels)
}
