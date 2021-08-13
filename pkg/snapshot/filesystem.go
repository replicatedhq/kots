package snapshot

import (
	"bufio"
	"bytes"
	"context"
	"crypto/md5"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	"github.com/google/uuid"
	"github.com/pkg/errors"
	"github.com/replicatedhq/kots/pkg/k8sutil"
	kotsadmresources "github.com/replicatedhq/kots/pkg/kotsadm/resources"
	kotsadmtypes "github.com/replicatedhq/kots/pkg/kotsadm/types"
	kotsadmversion "github.com/replicatedhq/kots/pkg/kotsadm/version"
	"github.com/replicatedhq/kots/pkg/kotsutil"
	kotss3 "github.com/replicatedhq/kots/pkg/s3"
	types "github.com/replicatedhq/kots/pkg/snapshot/types"
	"github.com/replicatedhq/kots/pkg/util"
	velerov1api "github.com/vmware-tanzu/velero/pkg/apis/velero/v1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	kuberneteserrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/client-go/kubernetes"
	"k8s.io/utils/pointer"
	k8syaml "sigs.k8s.io/yaml"
)

const (
	FileSystemMinioConfigMapName, FileSystemMinioSecretName                   = "kotsadm-fs-minio", "kotsadm-fs-minio-creds"
	FileSystemMinioDeploymentName, FileSystemMinioServiceName                 = "kotsadm-fs-minio", "kotsadm-fs-minio"
	FileSystemMinioProvider, FileSystemMinioBucketName, FileSystemMinioRegion = "aws", "velero", "minio"
	FileSystemMinioServicePort                                                = 9000
)

type FileSystemDeployOptions struct {
	Namespace        string
	IsOpenShift      bool
	ForceReset       bool
	FileSystemConfig types.FileSystemConfig
}

type ResetFileSystemError struct {
	Message string
}

func (e ResetFileSystemError) Error() string {
	return e.Message
}

func DeployFileSystemMinio(ctx context.Context, clientset kubernetes.Interface, deployOptions FileSystemDeployOptions, registryOptions kotsadmtypes.KotsadmOptions) error {
	// file system minio can be deployed before installing kotsadm or the application (e.g. disaster recovery)
	err := kotsadmresources.EnsurePrivateKotsadmRegistrySecret(deployOptions.Namespace, registryOptions, clientset)
	if err != nil {
		return errors.Wrap(err, "failed to ensure private kotsadm registry secret")
	}

	// configure fs directory/mount
	shouldReset, hasMinioConfig, _, err := shouldResetFileSystemMount(ctx, clientset, deployOptions, registryOptions)
	if err != nil {
		return errors.Wrap(err, "failed to check if should reset file system mount")
	}
	if shouldReset {
		if !deployOptions.ForceReset {
			return &ResetFileSystemError{Message: getFileSystemResetWarningMsg(deployOptions.FileSystemConfig)}
		}
		err := resetFileSystemMount(ctx, clientset, deployOptions, registryOptions)
		if err != nil {
			return errors.Wrap(err, "failed to reset file system mount")
		}
	}
	if shouldReset || !hasMinioConfig {
		// restart file system minio to regenerate the config
		err := k8sutil.ScaleDownDeployment(ctx, clientset, deployOptions.Namespace, FileSystemMinioDeploymentName)
		if err != nil {
			return errors.Wrap(err, "failed to scale down file system minio")
		}
	}

	// deploy resources
	err = ensureFileSystemConfigMap(ctx, clientset, deployOptions)
	if err != nil {
		return errors.Wrap(err, "failed to ensure file system minio secret")
	}
	secret, err := ensureFileSystemMinioSecret(ctx, clientset, deployOptions.Namespace)
	if err != nil {
		return errors.Wrap(err, "failed to ensure file system minio secret")
	}
	err = writeMinioKeysSHAFile(ctx, clientset, secret, deployOptions, registryOptions)
	if err != nil {
		return errors.Wrap(err, "failed to write minio keys sha file")
	}
	marshalledSecret, err := k8syaml.Marshal(secret)
	if err != nil {
		return errors.Wrap(err, "failed to marshal file system minio secret")
	}
	if err := ensureFileSystemMinioDeployment(ctx, clientset, deployOptions, registryOptions, marshalledSecret); err != nil {
		return errors.Wrap(err, "failed to ensure file system minio deployment")
	}
	if err := ensureFileSystemMinioService(ctx, clientset, deployOptions.Namespace); err != nil {
		return errors.Wrap(err, "failed to ensure service")
	}

	return nil
}

func DeployFileSystemLvp(ctx context.Context, clientset kubernetes.Interface, deployOptions FileSystemDeployOptions, registryOptions kotsadmtypes.KotsadmOptions) error {

	veleroNamespace, err := DetectVeleroNamespace(ctx, clientset, deployOptions.Namespace)
	if err != nil {
		return errors.Wrap(err, "failed to detect velero namespace")
	}

	// Deploy the config map exists for the plugin
	if err = EnsureLocalVolumeProviderConfigMap(deployOptions, veleroNamespace); err != nil {
		return errors.Wrap(err, "failed to configure local volume provider plugin config map")
	}

	return nil
}

func ValidateFileSystemDeployment(ctx context.Context, clientset kubernetes.Interface, deployOptions FileSystemDeployOptions, registryOptions kotsadmtypes.KotsadmOptions) (bool, bool, error) {
	// configure fs directory/mount. This is a legacy check to see if this directory was migrated from Minio and has an intermediate directory
	_, hasMinioConfig, writable, err := shouldResetFileSystemMount(ctx, clientset, deployOptions, registryOptions)
	if err != nil {
		return false, false, errors.Wrap(err, "failed to check if should reset file system mount")
	}
	return hasMinioConfig, writable, nil
}

func ensureFileSystemConfigMap(ctx context.Context, clientset kubernetes.Interface, deployOptions FileSystemDeployOptions) error {
	configmap := fileSystemConfigMapResource(deployOptions.FileSystemConfig)

	existingConfigMap, err := clientset.CoreV1().ConfigMaps(deployOptions.Namespace).Get(ctx, configmap.Name, metav1.GetOptions{})
	if err != nil {
		if !kuberneteserrors.IsNotFound(err) {
			return errors.Wrap(err, "failed to get existing configmap")
		}

		_, err := clientset.CoreV1().ConfigMaps(deployOptions.Namespace).Create(ctx, configmap, metav1.CreateOptions{})
		if err != nil {
			return errors.Wrap(err, "failed to create configmap")
		}

		return nil
	}

	existingConfigMap = updateFileSystemConfigMap(existingConfigMap, configmap)

	_, err = clientset.CoreV1().ConfigMaps(deployOptions.Namespace).Update(ctx, existingConfigMap, metav1.UpdateOptions{})
	if err != nil {
		return errors.Wrap(err, "failed to update deployment")
	}

	return nil
}

func fileSystemConfigMapResource(fileSystemConfig types.FileSystemConfig) *corev1.ConfigMap {
	data := map[string]string{}

	if fileSystemConfig.HostPath != nil {
		data["HOSTPATH"] = *fileSystemConfig.HostPath
	} else if fileSystemConfig.NFS != nil {
		data["NFS_PATH"] = fileSystemConfig.NFS.Path
		data["NFS_SERVER"] = fileSystemConfig.NFS.Server
	}

	return &corev1.ConfigMap{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "Secret",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: FileSystemMinioConfigMapName,
		},
		Data: data,
	}
}

func updateFileSystemConfigMap(existingConfigMap, desiredConfigMap *corev1.ConfigMap) *corev1.ConfigMap {
	existingConfigMap.Data = desiredConfigMap.Data

	return existingConfigMap
}

func ensureFileSystemMinioSecret(ctx context.Context, clientset kubernetes.Interface, namespace string) (*corev1.Secret, error) {
	secret := fileSystemMinioSecretResource()

	existingSecret, err := clientset.CoreV1().Secrets(namespace).Get(ctx, secret.Name, metav1.GetOptions{})
	if err != nil {
		if !kuberneteserrors.IsNotFound(err) {
			return nil, errors.Wrap(err, "failed to get existing secret")
		}

		s, err := clientset.CoreV1().Secrets(namespace).Create(ctx, secret, metav1.CreateOptions{})
		if err != nil {
			return nil, errors.Wrap(err, "failed to create secret")
		}

		return s, nil
	}

	// no patch needed

	return existingSecret, nil
}

func fileSystemMinioSecretResource() *corev1.Secret {
	accessKey := "kotsadm"
	secretKey := uuid.New().String()

	return &corev1.Secret{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "Secret",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: FileSystemMinioSecretName,
		},
		Data: map[string][]byte{
			"MINIO_ACCESS_KEY": []byte(accessKey),
			"MINIO_SECRET_KEY": []byte(secretKey),
		},
	}
}

func ensureFileSystemMinioDeployment(ctx context.Context, clientset kubernetes.Interface, deployOptions FileSystemDeployOptions, registryOptions kotsadmtypes.KotsadmOptions, marshalledSecret []byte) error {
	secretChecksum := fmt.Sprintf("%x", md5.Sum(marshalledSecret))

	deployment, err := fileSystemMinioDeploymentResource(clientset, secretChecksum, deployOptions, registryOptions)
	if err != nil {
		return errors.Wrap(err, "failed to get deployment resource")
	}

	existingDeployment, err := clientset.AppsV1().Deployments(deployOptions.Namespace).Get(ctx, deployment.Name, metav1.GetOptions{})
	if err != nil {
		if !kuberneteserrors.IsNotFound(err) {
			return errors.Wrap(err, "failed to get existing deployment")
		}

		_, err = clientset.AppsV1().Deployments(deployOptions.Namespace).Create(ctx, deployment, metav1.CreateOptions{})
		if err != nil {
			return errors.Wrap(err, "failed to create deployment")
		}

		return nil
	}

	existingDeployment, err = updateFileSystemMinioDeployment(existingDeployment, deployment)
	if err != nil {
		return errors.Wrap(err, "failed to modify deployment fields")
	}

	_, err = clientset.AppsV1().Deployments(deployOptions.Namespace).Update(ctx, existingDeployment, metav1.UpdateOptions{})
	if err != nil {
		return errors.Wrap(err, "failed to update deployment")
	}

	return nil
}

func fileSystemMinioDeploymentResource(clientset kubernetes.Interface, secretChecksum string, deployOptions FileSystemDeployOptions, registryOptions kotsadmtypes.KotsadmOptions) (*appsv1.Deployment, error) {
	image := "minio/minio:RELEASE.2021-08-05T22-01-19Z"
	imagePullSecrets := []corev1.LocalObjectReference{}

	if !kotsutil.IsKurl(clientset) || deployOptions.Namespace != metav1.NamespaceDefault {
		var err error
		imageRewriteFn := kotsadmversion.DependencyImageRewriteKotsadmRegistry(deployOptions.Namespace, &registryOptions)
		image, imagePullSecrets, err = imageRewriteFn(image, false)
		if err != nil {
			return nil, errors.Wrap(err, "failed to rewrite image")
		}
	}

	var securityContext corev1.PodSecurityContext
	if !deployOptions.IsOpenShift {
		securityContext = corev1.PodSecurityContext{
			RunAsUser: util.IntPointer(1001),
			FSGroup:   util.IntPointer(1001),
		}
	}

	env := []corev1.EnvVar{
		{
			Name:  "MINIO_UPDATE",
			Value: "off",
		},
		{
			Name: "MINIO_ACCESS_KEY",
			ValueFrom: &corev1.EnvVarSource{
				SecretKeyRef: &corev1.SecretKeySelector{
					LocalObjectReference: corev1.LocalObjectReference{
						Name: FileSystemMinioSecretName,
					},
					Key: "MINIO_ACCESS_KEY",
				},
			},
		},
		{
			Name: "MINIO_SECRET_KEY",
			ValueFrom: &corev1.EnvVarSource{
				SecretKeyRef: &corev1.SecretKeySelector{
					LocalObjectReference: corev1.LocalObjectReference{
						Name: FileSystemMinioSecretName,
					},
					Key: "MINIO_SECRET_KEY",
				},
			},
		},
	}

	return &appsv1.Deployment{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "apps/v1",
			Kind:       "Deployment",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: FileSystemMinioDeploymentName,
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: pointer.Int32Ptr(1),
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"app": "kotsadm-fs-minio",
				},
			},
			Strategy: appsv1.DeploymentStrategy{
				Type: appsv1.RecreateDeploymentStrategyType,
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"app": "kotsadm-fs-minio",
					},
					Annotations: map[string]string{
						"kots.io/fs-minio-creds-secret-checksum": secretChecksum,
					},
				},
				Spec: corev1.PodSpec{
					SecurityContext:  &securityContext,
					ImagePullSecrets: imagePullSecrets,
					Containers: []corev1.Container{
						{
							Image:           image,
							ImagePullPolicy: corev1.PullIfNotPresent,
							Name:            "minio",
							Ports: []corev1.ContainerPort{
								{Name: "http", ContainerPort: 9000},
							},
							Env: env,
							VolumeMounts: []corev1.VolumeMount{
								{
									Name: "data", MountPath: "/data",
								},
							},
							Args: []string{"--quiet", "server", "data"},
							LivenessProbe: &corev1.Probe{
								Handler: corev1.Handler{
									HTTPGet: &corev1.HTTPGetAction{
										Path: "/minio/health/live",
										Port: intstr.FromInt(9000),
									},
								},
								InitialDelaySeconds: 5,
								PeriodSeconds:       20,
							},
							ReadinessProbe: &corev1.Probe{
								Handler: corev1.Handler{
									HTTPGet: &corev1.HTTPGetAction{
										Path: "/minio/health/ready",
										Port: intstr.FromInt(9000),
									},
								},
								InitialDelaySeconds: 5,
								PeriodSeconds:       20,
							},
						},
					},
					Volumes: []corev1.Volume{
						{
							Name:         "data",
							VolumeSource: volumeSourceFromFileSystemConfig(deployOptions.FileSystemConfig),
						},
					},
				},
			},
		},
	}, nil
}

func updateFileSystemMinioDeployment(existingDeployment, desiredDeployment *appsv1.Deployment) (*appsv1.Deployment, error) {
	if len(existingDeployment.Spec.Template.Spec.Containers) == 0 {
		// hmm
		return desiredDeployment, nil
	}

	existingDeployment.Spec.Replicas = desiredDeployment.Spec.Replicas

	if existingDeployment.Spec.Template.Annotations == nil {
		existingDeployment.Spec.Template.ObjectMeta.Annotations = map[string]string{}
	}
	existingDeployment.Spec.Template.ObjectMeta.Annotations["kots.io/fs-minio-creds-secret-checksum"] = desiredDeployment.Spec.Template.ObjectMeta.Annotations["kots.io/fs-minio-creds-secret-checksum"]

	containerIdx := -1
	for idx, c := range existingDeployment.Spec.Template.Spec.Containers {
		if c.Name == "minio" {
			containerIdx = idx
		}
	}
	if containerIdx == -1 {
		return nil, errors.New("failed to find minio container in deployment")
	}

	existingDeployment.Spec.Template.Spec.Containers[containerIdx].Image = desiredDeployment.Spec.Template.Spec.Containers[containerIdx].Image
	existingDeployment.Spec.Template.Spec.Containers[containerIdx].LivenessProbe = desiredDeployment.Spec.Template.Spec.Containers[containerIdx].LivenessProbe
	existingDeployment.Spec.Template.Spec.Containers[containerIdx].ReadinessProbe = desiredDeployment.Spec.Template.Spec.Containers[containerIdx].ReadinessProbe
	existingDeployment.Spec.Template.Spec.Containers[containerIdx].Env = desiredDeployment.Spec.Template.Spec.Containers[containerIdx].Env

	existingDeployment.Spec.Template.Spec.Volumes = desiredDeployment.Spec.Template.Spec.Volumes

	return existingDeployment, nil
}

func volumeSourceFromFileSystemConfig(fileSystemConfig types.FileSystemConfig) corev1.VolumeSource {
	volumeSource := corev1.VolumeSource{}
	if fileSystemConfig.HostPath != nil {
		volumeSource.HostPath = &corev1.HostPathVolumeSource{
			Path: *fileSystemConfig.HostPath,
		}
	} else if fileSystemConfig.NFS != nil {
		volumeSource.NFS = &corev1.NFSVolumeSource{
			Path:   fileSystemConfig.NFS.Path,
			Server: fileSystemConfig.NFS.Server,
		}
	}
	return volumeSource
}

func ensureFileSystemMinioService(ctx context.Context, clientset kubernetes.Interface, namespace string) error {
	service := fileSystemMinioServiceResource()

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

	existingService = updateFileSystemMinioService(existingService, service)

	_, err = clientset.CoreV1().Services(namespace).Update(ctx, existingService, metav1.UpdateOptions{})
	if err != nil {
		return errors.Wrap(err, "failed to update service")
	}

	return nil
}

func fileSystemMinioServiceResource() *corev1.Service {
	return &corev1.Service{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "Service",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: FileSystemMinioServiceName,
		},
		Spec: corev1.ServiceSpec{
			Type: corev1.ServiceTypeClusterIP,
			Selector: map[string]string{
				"app": "kotsadm-fs-minio",
			},
			Ports: []corev1.ServicePort{
				{
					Protocol:   corev1.ProtocolTCP,
					Port:       FileSystemMinioServicePort,
					TargetPort: intstr.FromInt(9000),
				},
			},
		},
	}
}

func updateFileSystemMinioService(existingService, desiredService *corev1.Service) *corev1.Service {
	existingService.Spec.Ports = desiredService.Spec.Ports

	return existingService
}

func shouldResetFileSystemMount(ctx context.Context, clientset kubernetes.Interface, deployOptions FileSystemDeployOptions, registryOptions kotsadmtypes.KotsadmOptions) (shouldReset bool, hasMinioConfig bool, writable bool, finalErr error) {
	checkPod, err := createFileSystemMinioCheckPod(ctx, clientset, deployOptions, registryOptions)
	if err != nil {
		finalErr = errors.Wrap(err, "failed to create file system minio check pod")
		return
	}

	if err := k8sutil.WaitForPod(ctx, clientset, deployOptions.Namespace, checkPod.Name, time.Minute*2); err != nil {
		finalErr = errors.Wrap(err, "failed to wait for file system minio check pod to complete")
		return
	}

	logs, err := k8sutil.GetPodLogs(ctx, clientset, checkPod, true, nil)
	if err != nil {
		finalErr = errors.Wrap(err, "failed to get file system minio check pod logs")
		return
	}
	if len(logs) == 0 {
		finalErr = errors.New("no logs found")
		return
	}

	type FileSystemMinioCheckPodOutput struct {
		HasMinioConfig bool   `json:"hasMinioConfig"`
		MinioKeysSHA   string `json:"minioKeysSHA"`
		Writable       bool   `json:"writable"`
	}

	checkPodOutput := FileSystemMinioCheckPodOutput{}

	scanner := bufio.NewScanner(bytes.NewReader(logs))
	for scanner.Scan() {
		line := scanner.Text()

		if err := json.Unmarshal([]byte(line), &checkPodOutput); err != nil {
			continue
		}

		break
	}

	writable = checkPodOutput.Writable

	// only delete pod if we know we have an actionable output
	clientset.CoreV1().Pods(deployOptions.Namespace).Delete(ctx, checkPod.Name, metav1.DeleteOptions{})

	if !checkPodOutput.HasMinioConfig {
		shouldReset = false
		hasMinioConfig = false
		return
	}

	if checkPodOutput.MinioKeysSHA == "" {
		shouldReset = true
		hasMinioConfig = true
		return
	}

	minioSecret, err := clientset.CoreV1().Secrets(deployOptions.Namespace).Get(ctx, FileSystemMinioSecretName, metav1.GetOptions{})
	if err != nil {
		if !kuberneteserrors.IsNotFound(err) {
			finalErr = errors.Wrap(err, "failed to get existing minio secret")
			return
		}
		shouldReset = true
		hasMinioConfig = true
		return
	}

	newMinioKeysSHA := getMinioKeysSHA(string(minioSecret.Data["MINIO_ACCESS_KEY"]), string(minioSecret.Data["MINIO_SECRET_KEY"]))
	if newMinioKeysSHA == checkPodOutput.MinioKeysSHA {
		shouldReset = false
		hasMinioConfig = true
		return
	}

	shouldReset = true
	hasMinioConfig = true
	return
}

func resetFileSystemMount(ctx context.Context, clientset kubernetes.Interface, deployOptions FileSystemDeployOptions, registryOptions kotsadmtypes.KotsadmOptions) error {
	resetPod, err := createFileSystemMinioResetPod(ctx, clientset, deployOptions, registryOptions)
	if err != nil {
		return errors.Wrap(err, "failed to create file system minio reset pod")
	}

	if err := k8sutil.WaitForPod(ctx, clientset, deployOptions.Namespace, resetPod.Name, time.Minute*2); err != nil {
		return errors.Wrap(err, "failed to wait for file system minio reset pod")
	}

	logs, err := k8sutil.GetPodLogs(ctx, clientset, resetPod, true, nil)
	if err != nil {
		return errors.Wrap(err, "failed to get file system minio reset pod logs")
	}
	if len(logs) == 0 {
		return errors.New("no logs found")
	}

	type FileSystemMinioResetPodOutput struct {
		Success bool `json:"success"`
	}

	resetPodOutput := FileSystemMinioResetPodOutput{}

	scanner := bufio.NewScanner(bytes.NewReader(logs))
	for scanner.Scan() {
		line := scanner.Text()

		if err := json.Unmarshal([]byte(line), &resetPodOutput); err != nil {
			continue
		}

		break
	}

	if !resetPodOutput.Success {
		return errors.Wrapf(err, "failed to reset, please check %s pod logs for more details", resetPod.Name)
	}

	// only delete the pod on success
	clientset.CoreV1().Pods(deployOptions.Namespace).Delete(ctx, resetPod.Name, metav1.DeleteOptions{})

	return nil
}

func writeMinioKeysSHAFile(ctx context.Context, clientset kubernetes.Interface, minioSecret *corev1.Secret, deployOptions FileSystemDeployOptions, registryOptions kotsadmtypes.KotsadmOptions) error {
	minioKeysSHA := getMinioKeysSHA(string(minioSecret.Data["MINIO_ACCESS_KEY"]), string(minioSecret.Data["MINIO_SECRET_KEY"]))

	keysSHAPod, err := createFileSystemMinioKeysSHAPod(ctx, clientset, deployOptions, registryOptions, minioKeysSHA)
	if err != nil {
		return errors.Wrap(err, "failed to create file system minio keysSHA pod")
	}

	if err := k8sutil.WaitForPod(ctx, clientset, deployOptions.Namespace, keysSHAPod.Name, time.Minute*2); err != nil {
		return errors.Wrap(err, "failed to wait for file system minio keysSHA pod to complete")
	}

	logs, err := k8sutil.GetPodLogs(ctx, clientset, keysSHAPod, true, nil)
	if err != nil {
		return errors.Wrap(err, "failed to get file system minio keysSHA pod logs")
	}
	if len(logs) == 0 {
		return errors.New("no logs found")
	}

	type FileSystemMinioKeysSHAPodOutput struct {
		Success bool `json:"success"`
	}

	keysSHAPodOutput := FileSystemMinioKeysSHAPodOutput{}

	scanner := bufio.NewScanner(bytes.NewReader(logs))
	for scanner.Scan() {
		line := scanner.Text()

		if err := json.Unmarshal([]byte(line), &keysSHAPodOutput); err != nil {
			continue
		}

		break
	}

	if !keysSHAPodOutput.Success {
		return errors.Wrapf(err, "failed to write keys sha, please check %s pod logs for more details", keysSHAPod.Name)
	}

	// only delete the pod on success
	clientset.CoreV1().Pods(deployOptions.Namespace).Delete(ctx, keysSHAPod.Name, metav1.DeleteOptions{})

	return nil
}

func getMinioKeysSHA(accessKey, secretKey string) string {
	return fmt.Sprintf("%x", sha256.Sum256([]byte(fmt.Sprintf("%s,%s", accessKey, secretKey))))
}

func createFileSystemMinioCheckPod(ctx context.Context, clientset kubernetes.Interface, deployOptions FileSystemDeployOptions, registryOptions kotsadmtypes.KotsadmOptions) (*corev1.Pod, error) {
	pod, err := fileSystemMinioCheckPod(ctx, clientset, deployOptions, registryOptions)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get pod resource")
	}

	p, err := clientset.CoreV1().Pods(deployOptions.Namespace).Create(ctx, pod, metav1.CreateOptions{})
	if err != nil {
		return nil, errors.Wrap(err, "failed to create pod")
	}

	return p, nil
}

func createFileSystemMinioResetPod(ctx context.Context, clientset kubernetes.Interface, deployOptions FileSystemDeployOptions, registryOptions kotsadmtypes.KotsadmOptions) (*corev1.Pod, error) {
	pod, err := fileSystemMinioResetPod(ctx, clientset, deployOptions, registryOptions)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get pod resource")
	}

	p, err := clientset.CoreV1().Pods(deployOptions.Namespace).Create(ctx, pod, metav1.CreateOptions{})
	if err != nil {
		return nil, errors.Wrap(err, "failed to create pod")
	}

	return p, nil
}

func createFileSystemMinioKeysSHAPod(ctx context.Context, clientset kubernetes.Interface, deployOptions FileSystemDeployOptions, registryOptions kotsadmtypes.KotsadmOptions, minioKeysSHA string) (*corev1.Pod, error) {
	pod, err := fileSystemMinioKeysSHAPod(ctx, clientset, deployOptions, registryOptions, minioKeysSHA)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get pod resource")
	}

	p, err := clientset.CoreV1().Pods(deployOptions.Namespace).Create(ctx, pod, metav1.CreateOptions{})
	if err != nil {
		return nil, errors.Wrap(err, "failed to create pod")
	}

	return p, nil
}

func fileSystemMinioCheckPod(ctx context.Context, clientset kubernetes.Interface, deployOptions FileSystemDeployOptions, registryOptions kotsadmtypes.KotsadmOptions) (*corev1.Pod, error) {
	podName := fmt.Sprintf("kotsadm-fs-minio-check-%d", time.Now().Unix())
	command := []string{"/fs-minio-check.sh"}
	return fileSystemMinioConfigPod(clientset, deployOptions, registryOptions, podName, command, nil, true)
}

func fileSystemMinioResetPod(ctx context.Context, clientset kubernetes.Interface, deployOptions FileSystemDeployOptions, registryOptions kotsadmtypes.KotsadmOptions) (*corev1.Pod, error) {
	podName := fmt.Sprintf("kotsadm-fs-minio-reset-%d", time.Now().Unix())
	command := []string{"/fs-minio-reset.sh"}
	return fileSystemMinioConfigPod(clientset, deployOptions, registryOptions, podName, command, nil, false)
}

func fileSystemMinioKeysSHAPod(ctx context.Context, clientset kubernetes.Interface, deployOptions FileSystemDeployOptions, registryOptions kotsadmtypes.KotsadmOptions, minioKeysSHA string) (*corev1.Pod, error) {
	podName := fmt.Sprintf("kotsadm-fs-minio-keys-sha-%d", time.Now().Unix())
	command := []string{"/fs-minio-keys-sha.sh"}
	args := []string{minioKeysSHA}
	return fileSystemMinioConfigPod(clientset, deployOptions, registryOptions, podName, command, args, false)
}

func fileSystemMinioConfigPod(clientset kubernetes.Interface, deployOptions FileSystemDeployOptions, registryOptions kotsadmtypes.KotsadmOptions, podName string, command []string, args []string, readOnly bool) (*corev1.Pod, error) {
	var securityContext corev1.PodSecurityContext
	if !deployOptions.IsOpenShift {
		securityContext = corev1.PodSecurityContext{
			RunAsUser: util.IntPointer(1001),
			FSGroup:   util.IntPointer(1001),
		}
	}

	kotsadmTag := kotsadmversion.KotsadmTag(kotsadmtypes.KotsadmOptions{}) // default tag
	image := fmt.Sprintf("kotsadm/kotsadm:%s", kotsadmTag)
	imagePullSecrets := []corev1.LocalObjectReference{}

	if !kotsutil.IsKurl(clientset) || deployOptions.Namespace != metav1.NamespaceDefault {
		var err error
		imageRewriteFn := kotsadmversion.KotsadmImageRewriteKotsadmRegistry(deployOptions.Namespace, &registryOptions)
		image, imagePullSecrets, err = imageRewriteFn(image, false)
		if err != nil {
			return nil, errors.Wrap(err, "failed to rewrite image")
		}
	}

	pod := &corev1.Pod{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "Pod",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      podName,
			Namespace: deployOptions.Namespace,
			Labels: map[string]string{
				"app": "kotsadm-fs-minio",
			},
		},
		Spec: corev1.PodSpec{
			SecurityContext:  &securityContext,
			RestartPolicy:    corev1.RestartPolicyOnFailure,
			ImagePullSecrets: imagePullSecrets,
			Volumes: []corev1.Volume{
				{
					Name:         "fs",
					VolumeSource: volumeSourceFromFileSystemConfig(deployOptions.FileSystemConfig),
				},
			},
			Containers: []corev1.Container{
				{
					Image:           image,
					ImagePullPolicy: corev1.PullIfNotPresent,
					Name:            "fs-minio",
					Command:         command,
					Args:            args,
					VolumeMounts: []corev1.VolumeMount{
						{
							Name:      "fs",
							MountPath: "/fs",
						},
					},
					Resources: corev1.ResourceRequirements{
						Limits: corev1.ResourceList{
							"cpu":    resource.MustParse("100m"),
							"memory": resource.MustParse("100Mi"),
						},
						Requests: corev1.ResourceList{
							"cpu":    resource.MustParse("50m"),
							"memory": resource.MustParse("50Mi"),
						},
					},
				},
			},
		},
	}

	return pod, nil
}

func CreateFileSystemMinioBucket(ctx context.Context, clientset kubernetes.Interface, namespace string, registryOptions kotsadmtypes.KotsadmOptions) error {
	storeFileSystem, err := BuildMinioStoreFileSystem(ctx, clientset, namespace)
	if err != nil {
		return errors.Wrap(err, "failed to build file system store")
	}

	podName := fmt.Sprintf("kotsadm-fs-minio-bucket-%d", time.Now().Unix())

	options := kotss3.S3OpsPodOptions{
		PodName:         podName,
		Endpoint:        storeFileSystem.Endpoint,
		BucketName:      FileSystemMinioBucketName,
		AccessKeyID:     storeFileSystem.AccessKeyID,
		SecretAccessKey: storeFileSystem.SecretAccessKey,
		Namespace:       namespace,
		IsOpenShift:     k8sutil.IsOpenShift(clientset),
		RegistryOptions: &registryOptions,
	}
	return kotss3.CreateS3BucketUsingAPod(ctx, clientset, options)
}

func getFileSystemResetWarningMsg(fileSystemConfig types.FileSystemConfig) string {
	path := ""
	if fileSystemConfig.HostPath != nil {
		path = *fileSystemConfig.HostPath
	} else if fileSystemConfig.NFS != nil {
		path = fileSystemConfig.NFS.Path
	}
	return fmt.Sprintf("The %s directory was previously configured by a different minio instance.\nProceeding will re-configure it to be used only by the minio instance we deploy to configure the file system, and any other minio instance using this location will no longer have access.\nIf you are attempting to fully restore a prior installation, such as a disaster recovery scenario, this action is expected.", path)
}

func GetCurrentFileSystemConfig(ctx context.Context, namespace string, isMinioDisabled bool) (*types.FileSystemConfig, error) {
	var fileSystemConfig *types.FileSystemConfig

	if !isMinioDisabled {
		fileSystemConfig, err := GetCurrentMinioFileSystemConfig(ctx, namespace)
		if err != nil {
			return nil, errors.Wrap(err, "failed to get current minio file system config")
		}
		if fileSystemConfig != nil {
			return fileSystemConfig, nil
		}
		return nil, nil
	}

	fileSystemConfig, err := GetCurrentLvpFileSystemConfig(ctx, namespace)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get current lvp file system config")
	}

	return fileSystemConfig, nil
}

func GetCurrentLvpFileSystemConfig(ctx context.Context, namespace string) (*types.FileSystemConfig, error) {
	bsl, err := FindBackupStoreLocation(ctx, namespace)
	if err != nil {
		return nil, errors.Wrap(err, "failed to find velero backup storage location")
	}
	if bsl == nil {
		return nil, nil
	}

	var fileSystemConfig *types.FileSystemConfig

	switch bsl.Spec.Provider {
	case "replicated.com/hostpath":
		fileSystemConfig = &types.FileSystemConfig{}
		hostPath := bsl.Spec.Config["path"]
		fileSystemConfig.HostPath = &hostPath
		return fileSystemConfig, nil
	case "replicated.com/nfs":
		fileSystemConfig = &types.FileSystemConfig{
			NFS: &types.NFSConfig{
				Path:   bsl.Spec.Config["path"],
				Server: bsl.Spec.Config["server"],
			},
		}
		return fileSystemConfig, nil
	}
	return nil, nil
}

func GetCurrentMinioFileSystemConfig(ctx context.Context, namespace string) (*types.FileSystemConfig, error) {

	clientset, err := k8sutil.GetClientset()
	if err != nil {
		return nil, errors.Wrap(err, "failed to get k8s clientset")
	}

	fileSystemConfigMap, err := clientset.CoreV1().ConfigMaps(namespace).Get(ctx, FileSystemMinioConfigMapName, metav1.GetOptions{})
	if err != nil {
		if kuberneteserrors.IsNotFound(err) {
			return nil, nil
		}
		return nil, errors.Wrap(err, "failed to get file system configmap")
	}

	if fileSystemConfigMap.Data == nil {
		return &types.FileSystemConfig{}, nil
	}

	fileSystemConfig := types.FileSystemConfig{}

	if hostPath, ok := fileSystemConfigMap.Data["HOSTPATH"]; ok {
		fileSystemConfig.HostPath = &hostPath
	} else if _, ok := fileSystemConfigMap.Data["NFS_PATH"]; ok {
		fileSystemConfig.NFS = &types.NFSConfig{
			Path:   fileSystemConfigMap.Data["NFS_PATH"],
			Server: fileSystemConfigMap.Data["NFS_SERVER"],
		}
	}

	return &fileSystemConfig, nil
}

// RevertToMinioFS will apply the spec of the previous BSL to the current one and then update.
// Used for recovery during a failed migration from Minio to LVP.
func RevertToMinioFS(ctx context.Context, kotsadmNamespace, veleroNamespace string, previousBsl *velerov1api.BackupStorageLocation) error {
	bsl, err := FindBackupStoreLocation(context.TODO(), kotsadmNamespace)
	if err != nil {
		return errors.Wrap(err, "failed to find backupstoragelocations")
	}

	bsl.Spec = previousBsl.Spec

	err = UpdateBackupStorageLocation(ctx, veleroNamespace, bsl)
	if err != nil {
		return errors.Wrap(err, "failed to revert to minio backup storage location")
	}
	return nil
}

// DeleteFileSystemMinio cleans up the minio resources for hostpath and nfs snapshot deployments.
// The secret is not deleted, just in case.
func DeleteFileSystemMinio(ctx context.Context, kotsadmNamespace string) error {
	clientset, err := k8sutil.GetClientset()
	if err != nil {
		return errors.Wrap(err, "failed to get kubernetes clientset")
	}

	if err := clientset.CoreV1().ConfigMaps(kotsadmNamespace).Delete(ctx, FileSystemMinioConfigMapName, metav1.DeleteOptions{}); err != nil {
		return errors.Wrap(err, "failed to delete fs minio config map")
	}

	if err := clientset.AppsV1().Deployments(kotsadmNamespace).Delete(ctx, FileSystemMinioDeploymentName, metav1.DeleteOptions{}); err != nil {
		return errors.Wrap(err, "failed to delete fs minio deployment")
	}

	if err := clientset.CoreV1().Services(kotsadmNamespace).Delete(ctx, FileSystemMinioServiceName, metav1.DeleteOptions{}); err != nil {
		return errors.Wrap(err, "failed to delete fs minio service")
	}

	return nil
}

// EnsureLocalVolumeProviderConfigMap customizes the LVP plugin deployment with a config map
// based on the chosen file system backing and the detection of Openshift. This ensures
// the Velero and Restic have permissions to write to the disk.
func EnsureLocalVolumeProviderConfigMap(deployOptions FileSystemDeployOptions, veleroNamespace string) error {
	if deployOptions.IsOpenShift || veleroNamespace == "" {
		return nil
	}

	fsConfig := deployOptions.FileSystemConfig

	clientset, err := k8sutil.GetClientset()
	if err != nil {
		return errors.Wrap(err, "failed to get kubernetes clientset")
	}

	var pluginConfigMapLabel string
	if fsConfig.HostPath != nil {
		pluginConfigMapLabel = "replicated.com/hostpath"
	} else {
		pluginConfigMapLabel = "replicated.com/nfs"
	}

	listOpts := metav1.ListOptions{
		LabelSelector: fmt.Sprintf("%s=%s", pluginConfigMapLabel, "ObjectStore"),
	}

	configmaps, err := clientset.CoreV1().ConfigMaps(veleroNamespace).List(context.TODO(), listOpts)
	if err != nil {
		return errors.Wrap(err, "failed to list existing config maps")
	}

	if len(configmaps.Items) == 0 {
		// Create the config map
		configmap := &corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "local-volume-provider-config",
				Namespace: veleroNamespace,
				Labels: map[string]string{
					"velero.io/plugin-config": "",
					"replicated.com/nfs":      "ObjectStore",
					"replicated.com/hostpath": "ObjectStore",
				},
			},
			// These values are the settings used for the minio filesystem deployment
			Data: map[string]string{
				"securityContextRunAsUser": "1001",
				"securityContextFsGroup":   "1001",
			},
		}

		_, err = clientset.CoreV1().ConfigMaps(veleroNamespace).Create(context.TODO(), configmap, metav1.CreateOptions{})
		if err != nil {
			return errors.Wrap(err, "failed to create new local-volume-provider config map")
		}

		return nil
	}

	configmap := &configmaps.Items[0]
	configmap.Data["securityContextRunAsUser"] = "1001"
	configmap.Data["securityContextFsGroup"] = "1001"

	_, err = clientset.CoreV1().ConfigMaps(veleroNamespace).Update(context.TODO(), configmap, metav1.UpdateOptions{})
	if err != nil {
		return errors.Wrap(err, "failed to update local-volume-provider config map")
	}

	return nil
}

// IsFileSystemMinioDisable returns the value of an internal KOTS config map entry indicating
// if this installation has opted in or out of migrating from Minio to the LVP plugin.
func IsFileSystemMinioDisabled(kotsadmNamespace string) (bool, error) {
	clientset, err := k8sutil.GetClientset()
	if err != nil {
		return false, errors.Wrap(err, "failed to get kubernetes clientset")
	}

	// Get minio snapshot migration status v1.48.0
	kostadmConfig, err := clientset.CoreV1().ConfigMaps(kotsadmNamespace).Get(context.TODO(), "kotsadm-confg", metav1.GetOptions{})
	if err != nil {
		if kuberneteserrors.IsNotFound(err) {
			// TODO (dans) this behavior needs to change when this feature is opt-out.
			return false, nil
		}
		return false, errors.Wrap(err, "failed to get kotsadm-config map")
	}
	var minioEnabled bool
	if v, ok := kostadmConfig.Data["minio-enabled-snapshots"]; ok {
		minioEnabled, err = strconv.ParseBool(v)
		if err != nil {
			return false, errors.Wrap(err, "failed to parse minio-enabled-snapshots from kotsadm-confg")
		}
		return !minioEnabled, nil
	}

	return false, nil
}
