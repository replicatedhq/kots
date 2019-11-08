package kotsadm

import (
	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

func minioConfigMap(namespace string) *corev1.ConfigMap {
	configMap := &corev1.ConfigMap{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "ConfigMap",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "kotsadm-minio",
			Namespace: namespace,
		},
		Data: map[string]string{
			"initialize": `#!/bin/sh
set -e ; # Have script exit in the event of a failed command.

# connectToMinio
# Use a check-sleep-check loop to wait for Minio service to be available
connectToMinio() {
SCHEME=$1
ATTEMPTS=0 ; LIMIT=29 ; # Allow 30 attempts
set -e ; # fail if we can't read the keys.
ACCESS=$(cat /config/accesskey) ; SECRET=$(cat /config/secretkey) ;
set +e ; # The connections to minio are allowed to fail.
echo "Connecting to Minio server: $SCHEME://$MINIO_ENDPOINT:$MINIO_PORT" ;
MC_COMMAND="mc config host add myminio $SCHEME://$MINIO_ENDPOINT:$MINIO_PORT $ACCESS $SECRET" ;
$MC_COMMAND ;
STATUS=$? ;
until [ $STATUS = 0 ]
do
ATTEMPTS=` + "`expr $ATTEMPTS + 1`" + `;
echo \"Failed attempts: $ATTEMPTS\" ;
if [ $ATTEMPTS -gt $LIMIT ]; then
exit 1 ;
fi ;
sleep 2 ; # 1 second intervals between attempts
$MC_COMMAND ;
STATUS=$? ;
done ;
set -e ; # reset ` + "`e`" + ` as active
return 0
}

# checkBucketExists ($bucket)
# Check if the bucket exists, by using the exit code of ` + "`mc ls`" + `
checkBucketExists() {
BUCKET=$1
CMD=$(/usr/bin/mc ls myminio/$BUCKET > /dev/null 2>&1)
return $?
}

# createBucket ($bucket, $policy, $purge)
# Ensure bucket exists, purging if asked to
createBucket() {
BUCKET=$1
POLICY=$2
PURGE=$3

# Purge the bucket, if set & exists
# Since PURGE is user input, check explicitly for ` + "`true`" + `
if [ $PURGE = true ]; then
if checkBucketExists $BUCKET ; then
echo "Purging bucket '$BUCKET'."
set +e ; # don't exit if this fails
/usr/bin/mc rm -r --force myminio/$BUCKET
set -e ; # reset ` + "`e`" + ` as active
else
echo "Bucket '$BUCKET' does not exist, skipping purge."
fi
fi

# Create the bucket if it does not exist
if ! checkBucketExists $BUCKET ; then
echo "Creating bucket '$BUCKET'"
/usr/bin/mc mb myminio/$BUCKET
else
echo "Bucket '$BUCKET' already exists."
fi

# At this point, the bucket should exist, skip checking for existence
# Set policy on the bucket
echo "Setting policy of bucket '$BUCKET' to '$POLICY'."
/usr/bin/mc policy $POLICY myminio/$BUCKET
}

# Try connecting to Minio instance
scheme=http
connectToMinio $scheme
# Create the bucket
createBucket kotsadm none false`,
		},
	}

	return configMap
}

func minioStatefulset(namespace string) *appsv1.StatefulSet {
	statefulset := &appsv1.StatefulSet{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "apps/v1",
			Kind:       "StatefulSet",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "kotsadm-minio",
			Namespace: namespace,
		},
		Spec: appsv1.StatefulSetSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"app": "kotsadm-minio",
				},
			},
			UpdateStrategy: appsv1.StatefulSetUpdateStrategy{
				Type: appsv1.RollingUpdateStatefulSetStrategyType,
			},
			VolumeClaimTemplates: []corev1.PersistentVolumeClaim{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "kotsadm-minio",
					},
					Spec: corev1.PersistentVolumeClaimSpec{
						AccessModes: []corev1.PersistentVolumeAccessMode{
							corev1.ReadWriteOnce,
						},
						Resources: corev1.ResourceRequirements{
							Requests: corev1.ResourceList{
								corev1.ResourceName(corev1.ResourceStorage): resource.MustParse("4Gi"),
							},
						},
					},
				},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"app": "kotsadm-minio",
					},
				},
				Spec: corev1.PodSpec{
					SecurityContext: &corev1.PodSecurityContext{
						// RunAsUser: util.IntPointer(1001), // TODO: make real user #
					},
					Volumes: []corev1.Volume{
						{
							Name: "kotsadm-minio",
							VolumeSource: corev1.VolumeSource{
								PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
									ClaimName: "kotsadm-minio",
								},
							},
						},
						{
							Name: "minio-config-dir",
							VolumeSource: corev1.VolumeSource{
								EmptyDir: &corev1.EmptyDirVolumeSource{},
							},
						},
					},
					Containers: []corev1.Container{
						{
							Image:           "minio/minio:RELEASE.2019-10-12T01-39-57Z",
							ImagePullPolicy: corev1.PullIfNotPresent,
							Name:            "kotsadm-minio",
							Command: []string{
								"/bin/sh",
								"-ce",
								"/usr/bin/docker-entrypoint.sh minio -C /root/.minio/ server /export",
							},
							Ports: []corev1.ContainerPort{
								{
									Name:          "service",
									ContainerPort: 9000,
								},
							},
							VolumeMounts: []corev1.VolumeMount{
								{
									Name:      "kotsadm-minio",
									MountPath: "/export",
								},
								{
									Name:      "minio-config-dir",
									MountPath: "/root/.minio/",
								},
							},
							Env: []corev1.EnvVar{
								{
									Name: "MINIO_ACCESS_KEY",
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
									Name: "MINIO_SECRET_KEY",
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
									Name:  "MINIO_BROWSER",
									Value: "on",
								},
							},
							LivenessProbe: &corev1.Probe{
								InitialDelaySeconds: 5,
								TimeoutSeconds:      1,
								FailureThreshold:    3,
								SuccessThreshold:    1,
								PeriodSeconds:       30,
								Handler: corev1.Handler{
									HTTPGet: &corev1.HTTPGetAction{
										Path:   "/minio/health/live",
										Port:   intstr.FromString("service"),
										Scheme: corev1.URISchemeHTTP,
									},
								},
							},
							ReadinessProbe: &corev1.Probe{
								InitialDelaySeconds: 5,
								TimeoutSeconds:      1,
								FailureThreshold:    3,
								SuccessThreshold:    1,
								PeriodSeconds:       15,
								Handler: corev1.Handler{
									HTTPGet: &corev1.HTTPGetAction{
										Path:   "/minio/health/ready",
										Port:   intstr.FromString("service"),
										Scheme: corev1.URISchemeHTTP,
									},
								},
							},
						},
					},
				},
			},
		},
	}

	return statefulset
}

func minioService(namespace string) *corev1.Service {
	service := &corev1.Service{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "Service",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "kotsadm-minio",
			Namespace: namespace,
		},
		Spec: corev1.ServiceSpec{
			Selector: map[string]string{
				"app": "kotsadm-minio",
			},
			Type: corev1.ServiceTypeClusterIP,
			Ports: []corev1.ServicePort{
				{
					Name:       "service",
					Port:       9000,
					TargetPort: intstr.FromInt(9000),
				},
			},
		},
	}

	return service
}

func minioJob(namespace string) *batchv1.Job {
	job := &batchv1.Job{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "batch/v1",
			Kind:       "Job",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "kotsadm-minio",
			Namespace: namespace,
		},
		Spec: batchv1.JobSpec{
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"app": "kotsadm-minio",
					},
				},
				Spec: corev1.PodSpec{
					SecurityContext: &corev1.PodSecurityContext{
						// RunAsUser: util.IntPointer(1001), // TODO: make real user #
					},
					RestartPolicy: corev1.RestartPolicyOnFailure,
					Volumes: []corev1.Volume{
						{
							Name: "minio-configuration",
							VolumeSource: corev1.VolumeSource{
								Projected: &corev1.ProjectedVolumeSource{
									Sources: []corev1.VolumeProjection{
										{
											ConfigMap: &corev1.ConfigMapProjection{
												LocalObjectReference: corev1.LocalObjectReference{
													Name: "kotsadm-minio",
												},
											},
										},
										{
											Secret: &corev1.SecretProjection{
												LocalObjectReference: corev1.LocalObjectReference{
													Name: "kotsadm-minio",
												},
											},
										},
									},
								},
							},
						},
					},
					Containers: []corev1.Container{
						{
							Command: []string{
								"/bin/sh",
								"/config/initialize",
							},
							Env: []corev1.EnvVar{
								{
									Name:  "MINIO_ENDPOINT",
									Value: "kotsadm-minio",
								},
								{
									Name:  "MINIO_PORT",
									Value: "9000",
								},
							},
							Image:           "minio/mc:RELEASE.2019-10-09T22-54-57Z",
							ImagePullPolicy: corev1.PullIfNotPresent,
							Name:            "kotsadm-minio-mc",
							VolumeMounts: []corev1.VolumeMount{
								{
									Name:      "minio-configuration",
									MountPath: "/config",
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
