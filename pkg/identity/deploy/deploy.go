package deploy

import (
	"context"
	"crypto/md5"
	"fmt"
	"os"

	"github.com/pkg/errors"
	kotsv1beta1 "github.com/replicatedhq/kots/kotskinds/apis/kots/v1beta1"
	"github.com/replicatedhq/kots/pkg/ingress"
	kotsadmtypes "github.com/replicatedhq/kots/pkg/kotsadm/types"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	extensionsv1beta1 "k8s.io/api/extensions/v1beta1"
	kuberneteserrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8stypes "k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/client-go/kubernetes"
)

var (
	KotsIdentityLabelKey = "kots.io/identity"
)

type ImageRewriteFunc func(upstreamImage, path, tag string) (image string, imagePullSecrets []corev1.LocalObjectReference)

type Options struct {
	NamePrefix         string
	IdentitySpec       kotsv1beta1.IdentitySpec
	IdentityConfigSpec kotsv1beta1.IdentityConfigSpec
	ImageRewriteFn     ImageRewriteFunc
}

func Deploy(ctx context.Context, clientset kubernetes.Interface, namespace string, options Options) error {
	dexConfig, err := getDexConfig(ctx, options.IdentitySpec, options.IdentityConfigSpec)
	if err != nil {
		return errors.Wrap(err, "failed to get dex config")
	}
	if err := ensureSecret(ctx, clientset, namespace, options.NamePrefix, dexConfig); err != nil {
		return errors.Wrap(err, "failed to ensure secret")
	}
	if err := ensureDeployment(ctx, clientset, namespace, options.NamePrefix, dexConfig, options.ImageRewriteFn); err != nil {
		return errors.Wrap(err, "failed to ensure deployment")
	}
	if err := ensureService(ctx, clientset, namespace, options.NamePrefix, options.IdentityConfigSpec.IngressConfig); err != nil {
		return errors.Wrap(err, "failed to ensure service")
	}
	if err := ensureIngress(ctx, clientset, namespace, options.NamePrefix, options.IdentityConfigSpec.IngressConfig); err != nil {
		return errors.Wrap(err, "failed to ensure ingress")
	}
	return nil
}

func Configure(ctx context.Context, clientset kubernetes.Interface, namespace string, options Options) error {
	dexConfig, err := getDexConfig(ctx, options.IdentitySpec, options.IdentityConfigSpec)
	if err != nil {
		return errors.Wrap(err, "failed to get dex config")
	}
	if err := ensureSecret(ctx, clientset, namespace, options.NamePrefix, dexConfig); err != nil {
		return errors.Wrap(err, "failed to ensure secret")
	}
	if err := patchDeploymentSecret(ctx, clientset, namespace, options.NamePrefix, dexConfig); err != nil {
		return errors.Wrap(err, "failed to patch deployment secret")
	}
	return nil
}

func AdditionalLabels(namePrefix string) map[string]string {
	return map[string]string{
		KotsIdentityLabelKey: namePrefix,
	}
}

func ensureSecret(ctx context.Context, clientset kubernetes.Interface, namespace string, namePrefix string, dexConfig []byte) error {
	secret := secretResource(namePrefix, dexConfig)

	existingSecret, err := clientset.CoreV1().Secrets(namespace).Get(ctx, secret.Name, metav1.GetOptions{})
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

func secretResource(namePrefix string, dexConfig []byte) *corev1.Secret {
	return &corev1.Secret{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "Secret",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:   prefixName(namePrefix, "dex"),
			Labels: kotsadmtypes.GetKotsadmLabels(AdditionalLabels(namePrefix)),
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

func ensureDeployment(ctx context.Context, clientset kubernetes.Interface, namespace string, namePrefix string, marshalledDexConfig []byte, imageRewriteFn ImageRewriteFunc) error {
	configChecksum := fmt.Sprintf("%x", md5.Sum(marshalledDexConfig))

	deployment := deploymentResource(namePrefix, configChecksum, imageRewriteFn)

	existingDeployment, err := clientset.AppsV1().Deployments(namespace).Get(ctx, deployment.Name, metav1.GetOptions{})
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

	existingDeployment = updateDeployment(namePrefix, existingDeployment, deployment)

	_, err = clientset.AppsV1().Deployments(namespace).Update(ctx, existingDeployment, metav1.UpdateOptions{})
	if err != nil {
		return errors.Wrap(err, "failed to update deployment")
	}

	return nil
}

func patchDeploymentSecret(ctx context.Context, clientset kubernetes.Interface, namespace string, namePrefix string, marshalledDexConfig []byte) error {
	configChecksum := fmt.Sprintf("%x", md5.Sum(marshalledDexConfig))

	deployment := deploymentResource(prefixName(namePrefix, "dex"), configChecksum, nil)

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

func deploymentResource(namePrefix, configChecksum string, imageRewriteFn ImageRewriteFunc) *appsv1.Deployment {
	replicas := int32(2)
	volume := configSecretVolume(namePrefix)

	image := "quay.io/dexidp/dex:v2.26.0"
	imagePullSecrets := []corev1.LocalObjectReference{}
	if imageRewriteFn != nil {
		image, imagePullSecrets = imageRewriteFn(image, "dex", "v2.26.0")
	}

	env := []corev1.EnvVar{clientSecretEnvVar(namePrefix)}

	// TODO (ethan): this will not really work when kotsadm is not rendering this
	for _, name := range []string{"HTTP_PROXY", "HTTPS_PROXY", "NO_PROXY", "http_proxy", "https_proxy", "no_proxy"} {
		if val := os.Getenv(name); val != "" {
			env = append(env, corev1.EnvVar{Name: name, Value: val})
		}
	}

	return &appsv1.Deployment{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "apps/v1",
			Kind:       "Deployment",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:   prefixName(namePrefix, "dex"),
			Labels: kotsadmtypes.GetKotsadmLabels(AdditionalLabels(namePrefix)),
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: &replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"app": prefixName(namePrefix, "dex"),
				},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"app": prefixName(namePrefix, "dex"),
					},
					Annotations: map[string]string{
						"kots.io/dex-secret-checksum": configChecksum,
					},
				},
				Spec: corev1.PodSpec{
					ImagePullSecrets: imagePullSecrets,
					Containers: []corev1.Container{
						{
							Image:           image,
							ImagePullPolicy: corev1.PullIfNotPresent,
							Name:            "dex",
							Command:         []string{"/usr/local/bin/dex", "serve", "/etc/dex/cfg/dexConfig.yaml"},
							Ports: []corev1.ContainerPort{
								{Name: "http", ContainerPort: 5556},
							},
							EnvFrom: []corev1.EnvFromSource{postgresSecretEnvFromSource(namePrefix)},
							Env:     env,
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

func configSecretVolume(namePrefix string) corev1.Volume {
	return corev1.Volume{
		Name: "config",
		VolumeSource: corev1.VolumeSource{
			Secret: &corev1.SecretVolumeSource{
				SecretName: prefixName(namePrefix, "dex"),
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

func postgresSecretEnvFromSource(namePrefix string) corev1.EnvFromSource {
	return corev1.EnvFromSource{
		SecretRef: &corev1.SecretEnvSource{
			LocalObjectReference: corev1.LocalObjectReference{
				Name: prefixName(namePrefix, "dex-postgres"),
			},
		},
	}
}

func updateDeployment(namePrefix string, existingDeployment, desiredDeployment *appsv1.Deployment) *appsv1.Deployment {
	if len(existingDeployment.Spec.Template.Spec.Containers) == 0 {
		// wtf
		return desiredDeployment
	}

	if existingDeployment.Spec.Template.Annotations == nil {
		existingDeployment.Spec.Template.ObjectMeta.Annotations = map[string]string{}
	}
	existingDeployment.Spec.Template.ObjectMeta.Annotations["kots.io/dex-secret-checksum"] = desiredDeployment.Spec.Template.ObjectMeta.Annotations["kots.io/dex-secret-checksum"]

	existingDeployment.Spec.Template.Spec.Containers[0].Image = desiredDeployment.Spec.Template.Spec.Containers[0].Image

	existingDeployment = updateDeploymentConfigSecretVolume(namePrefix, existingDeployment, desiredDeployment)

	existingDeployment = updateDeploymentClientSecretEnvVar(namePrefix, existingDeployment, desiredDeployment)

	existingDeployment = updateDeploymentPostgresSecretEnvFromSource(namePrefix, existingDeployment, desiredDeployment)

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

func updateDeploymentPostgresSecretEnvFromSource(namePrefix string, existingDeployment *appsv1.Deployment, desiredDeployment *appsv1.Deployment) *appsv1.Deployment {
	if len(existingDeployment.Spec.Template.Spec.Containers) == 0 {
		return desiredDeployment
	}

	newPostgresSecretEnvFromSource := postgresSecretEnvFromSource(namePrefix)

	for _, envFrom := range existingDeployment.Spec.Template.Spec.Containers[0].EnvFrom {
		if envFrom.SecretRef.Name == newPostgresSecretEnvFromSource.SecretRef.Name {
			return existingDeployment
		}
	}

	existingDeployment.Spec.Template.Spec.Containers[0].EnvFrom =
		append(existingDeployment.Spec.Template.Spec.Containers[0].EnvFrom, newPostgresSecretEnvFromSource)

	return existingDeployment
}

func ensureService(ctx context.Context, clientset kubernetes.Interface, namespace string, namePrefix string, ingressSpec kotsv1beta1.IngressConfigSpec) error {
	service := serviceResource(namePrefix, ingressSpec)

	existingService, err := clientset.CoreV1().Services(namespace).Get(ctx, service.Name, metav1.GetOptions{})
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

func serviceResource(namePrefix string, ingressSpec kotsv1beta1.IngressConfigSpec) *corev1.Service {
	serviceType := corev1.ServiceTypeClusterIP
	port := corev1.ServicePort{
		Name:       "http",
		Port:       5556,
		TargetPort: intstr.IntOrString{Type: intstr.Int, IntVal: 5556},
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
			Name:   prefixName(namePrefix, "dex"),
			Labels: kotsadmtypes.GetKotsadmLabels(AdditionalLabels(namePrefix)),
		},
		Spec: corev1.ServiceSpec{
			Type: serviceType,
			Selector: map[string]string{
				"app": prefixName(namePrefix, "dex"),
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

func ensureIngress(ctx context.Context, clientset kubernetes.Interface, namespace string, namePrefix string, ingressSpec kotsv1beta1.IngressConfigSpec) error {
	if !ingressSpec.Enabled || ingressSpec.Ingress == nil {
		return deleteIngress(ctx, clientset, namespace, namePrefix)
	}
	dexIngress := ingressResource(namePrefix, *ingressSpec.Ingress)
	return ingress.EnsureIngress(ctx, clientset, namespace, dexIngress)
}

func ingressResource(namePrefix string, ingressConfig kotsv1beta1.IngressResourceConfig) *extensionsv1beta1.Ingress {
	return ingress.IngressFromConfig(ingressConfig, prefixName(namePrefix, "dex"), prefixName(namePrefix, "dex"), 5556, AdditionalLabels(namePrefix))
}

func prefixName(prefix, name string) string {
	return fmt.Sprintf("%s-%s", prefix, name)
}
