package snapshot

import (
	"context"
	"fmt"

	"github.com/pkg/errors"
	kotsadmtypes "github.com/replicatedhq/kots/pkg/kotsadm/types"
	"github.com/replicatedhq/kots/pkg/logger"
	velerov1 "github.com/vmware-tanzu/velero/pkg/apis/velero/v1"
	veleroclientv1 "github.com/vmware-tanzu/velero/pkg/generated/clientset/versioned/typed/velero/v1"
	"github.com/vmware-tanzu/velero/pkg/kuberesource"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
)

type InstanceRestoreOptions struct {
	BackupName string
	Silent     bool
}

func InstanceRestore(instanceRestoreOptions InstanceRestoreOptions) error {
	log := logger.NewLogger()
	if instanceRestoreOptions.Silent {
		log.Silence()
	}

	bsl, err := findBackupStoreLocation()
	if err != nil {
		return errors.Wrap(err, "failed to get velero namespace")
	}

	veleroNamespace := bsl.Namespace

	// get the backup
	cfg, err := config.GetConfig()
	if err != nil {
		return errors.Wrap(err, "failed to get cluster config")
	}

	veleroClient, err := veleroclientv1.NewForConfig(cfg)
	if err != nil {
		return errors.Wrap(err, "failed to create clientset")
	}

	backup, err := veleroClient.Backups(veleroNamespace).Get(context.TODO(), instanceRestoreOptions.BackupName, metav1.GetOptions{})
	if err != nil {
		return errors.Wrap(err, "failed to find backup")
	}

	kotsadmImage, ok := backup.Annotations["kots.io/kotsadm-image"]
	if !ok {
		return errors.Wrap(err, "failed to find kotsadm image annotation")
	}

	kotsadmNamespace, ok := backup.Annotations["kots.io/kotsadm-deploy-namespace"]
	if !ok {
		return errors.Wrap(err, "failed to find kotsadm deploy namespace annotation")
	}

	trueVal := true
	restore := &velerov1.Restore{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: veleroNamespace,
			Name:      instanceRestoreOptions.BackupName, // restore name same as backup name
		},
		Spec: velerov1.RestoreSpec{
			BackupName: instanceRestoreOptions.BackupName,
			LabelSelector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					kotsadmtypes.BackupLabel: kotsadmtypes.BackupLabelValue,
				},
			},
			RestorePVs:              &trueVal,
			IncludeClusterResources: &trueVal,
			Hooks: velerov1.RestoreHooks{
				Resources: []velerov1.RestoreResourceHookSpec{
					{
						Name:               "kotsadm-restore-hook",
						IncludedNamespaces: []string{kotsadmNamespace},
						IncludedResources:  []string{kuberesource.Pods.Resource},
						LabelSelector: &metav1.LabelSelector{
							MatchLabels: map[string]string{
								kotsadmtypes.KotsadmKey: kotsadmtypes.KotsadmLabelValue,
								"app":                   "kotsadm",
							},
						},
						PostHooks: []velerov1.RestoreResourceHook{
							{
								Init: &velerov1.InitRestoreHook{
									InitContainers: []corev1.Container{
										{
											Name:            "restore-db",
											Image:           kotsadmImage,
											ImagePullPolicy: corev1.PullAlways,
											Command: []string{
												"/restore-db.sh",
											},
											VolumeMounts: []corev1.VolumeMount{
												{
													Name:      "backup",
													MountPath: "/backup",
												},
											},
											Env: []corev1.EnvVar{
												{
													Name: "POSTGRES_PASSWORD",
													ValueFrom: &corev1.EnvVarSource{
														SecretKeyRef: &corev1.SecretKeySelector{
															LocalObjectReference: corev1.LocalObjectReference{
																Name: "kotsadm-postgres",
															},
															Key: "password",
														},
													},
												},
											},
											Resources: corev1.ResourceRequirements{
												Limits: corev1.ResourceList{
													"cpu":    resource.MustParse("500m"),
													"memory": resource.MustParse("500Mi"),
												},
												Requests: corev1.ResourceList{
													"cpu":    resource.MustParse("100m"),
													"memory": resource.MustParse("100Mi"),
												},
											},
										},
										{
											Name:            "restore-s3",
											Image:           kotsadmImage,
											ImagePullPolicy: corev1.PullAlways,
											Command: []string{
												"/restore-s3.sh",
											},
											VolumeMounts: []corev1.VolumeMount{
												{
													Name:      "backup",
													MountPath: "/backup",
												},
											},
											Env: []corev1.EnvVar{
												{
													Name:  "S3_ENDPOINT",
													Value: "http://kotsadm-minio:9000",
												},
												{
													Name:  "S3_BUCKET_NAME",
													Value: "kotsadm",
												},
												{
													Name: "S3_ACCESS_KEY_ID",
													ValueFrom: &corev1.EnvVarSource{
														SecretKeyRef: &corev1.SecretKeySelector{
															LocalObjectReference: corev1.LocalObjectReference{
																Name: "kotsadm-minio",
															},
															Key: "accesskey",
														},
													},
												},
												{
													Name: "S3_SECRET_ACCESS_KEY",
													ValueFrom: &corev1.EnvVarSource{
														SecretKeyRef: &corev1.SecretKeySelector{
															LocalObjectReference: corev1.LocalObjectReference{
																Name: "kotsadm-minio",
															},
															Key: "secretkey",
														},
													},
												},
												{
													Name:  "S3_BUCKET_ENDPOINT",
													Value: "true",
												},
											},
											Resources: corev1.ResourceRequirements{
												Limits: corev1.ResourceList{
													"cpu":    resource.MustParse("500m"),
													"memory": resource.MustParse("500Mi"),
												},
												Requests: corev1.ResourceList{
													"cpu":    resource.MustParse("100m"),
													"memory": resource.MustParse("100Mi"),
												},
											},
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

	restore, err = veleroClient.Restores(veleroNamespace).Create(context.TODO(), restore, metav1.CreateOptions{})
	if err != nil {
		return errors.Wrap(err, "failed to create restore")
	}

	log.ActionWithoutSpinner(fmt.Sprintf("Restore request has been created. Restore name is %s", restore.ObjectMeta.Name))

	return nil
}

func findBackupStoreLocation() (*velerov1.BackupStorageLocation, error) {
	cfg, err := config.GetConfig()
	if err != nil {
		return nil, errors.Wrap(err, "failed to get cluster config")
	}

	veleroClient, err := veleroclientv1.NewForConfig(cfg)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create velero clientset")
	}

	backupStorageLocations, err := veleroClient.BackupStorageLocations("").List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return nil, errors.Wrap(err, "failed to list backupstoragelocations")
	}

	for _, backupStorageLocation := range backupStorageLocations.Items {
		if backupStorageLocation.Name == "default" {
			return &backupStorageLocation, nil
		}
	}

	return nil, errors.New("global config not found")
}
