package s3

import (
	"os"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"

	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/pkg/errors"
	"github.com/replicatedhq/kots/pkg/k8sutil"
	kotsadmtypes "github.com/replicatedhq/kots/pkg/kotsadm/types"
	kotsadmversion "github.com/replicatedhq/kots/pkg/kotsadm/version"
	"github.com/replicatedhq/kots/pkg/kotsutil"
	"github.com/replicatedhq/kots/pkg/util"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

type S3OpsPodOptions struct {
	PodName         string
	Endpoint        string
	BucketName      string
	AccessKeyID     string
	SecretAccessKey string
	Namespace       string
	IsOpenShift     bool
	RegistryOptions *kotsadmtypes.KotsadmOptions
}

func GetConfig() *aws.Config {
	forcePathStyle := false
	if os.Getenv("S3_BUCKET_ENDPOINT") == "true" {
		forcePathStyle = true
	}

	region := os.Getenv("S3_REGION")
	if region == "" {
		region = "us-east-1"
	}

	s3Config := &aws.Config{
		Credentials:      credentials.NewStaticCredentials(os.Getenv("S3_ACCESS_KEY_ID"), os.Getenv("S3_SECRET_ACCESS_KEY"), ""),
		Endpoint:         aws.String(os.Getenv("S3_ENDPOINT")),
		Region:           aws.String(region),
		DisableSSL:       aws.Bool(true),
		S3ForcePathStyle: aws.Bool(forcePathStyle),
	}

	return s3Config
}

// CreateS3BucketUsingAPod is helpful when trying to hit a cluster s3 service using the CLI since that could be used outside the cluster, or due to firewall restrictions
func CreateS3BucketUsingAPod(ctx context.Context, clientset kubernetes.Interface, podOptions S3OpsPodOptions) error {
	command := []string{"/s3-bucket-create.sh"}
	pod, err := s3BucketPod(clientset, podOptions, command)
	if err != nil {
		return errors.Wrap(err, "failed to get pod resource")
	}

	createBucketPod, err := clientset.CoreV1().Pods(podOptions.Namespace).Create(ctx, pod, metav1.CreateOptions{})
	if err != nil {
		return errors.Wrap(err, "failed to create pod")
	}

	if err := k8sutil.WaitForPod(ctx, clientset, podOptions.Namespace, createBucketPod.Name, time.Minute*2); err != nil {
		return errors.Wrap(err, "failed to wait for pod")
	}

	logs, err := k8sutil.GetPodLogs(ctx, clientset, createBucketPod, true, nil)
	if err != nil {
		return errors.Wrap(err, "failed to get pod logs")
	}
	if len(logs) == 0 {
		return errors.New("no logs found")
	}

	type CreateBucketPodOutput struct {
		Success bool `json:"success"`
	}

	createBucketPodOutput := CreateBucketPodOutput{}
	if err := json.Unmarshal(logs, &createBucketPodOutput); err != nil {
		return errors.Wrapf(err, "failed to unmarshal %s pod logs", createBucketPod.Name)
	}

	if !createBucketPodOutput.Success {
		return errors.Wrapf(err, "failed to create bucket, please check %s pod logs for more details", createBucketPod.Name)
	}

	// only delete the pod on success
	clientset.CoreV1().Pods(podOptions.Namespace).Delete(ctx, createBucketPod.Name, metav1.DeleteOptions{})

	return nil
}

// HeadS3BucketUsingAPod is helpful when trying to hit a cluster s3 service using the CLI since that could be used outside the cluster, or due to firewall restrictions
func HeadS3BucketUsingAPod(ctx context.Context, clientset kubernetes.Interface, podOptions S3OpsPodOptions) error {
	command := []string{"/s3-bucket-head.sh"}
	pod, err := s3BucketPod(clientset, podOptions, command)
	if err != nil {
		return errors.Wrap(err, "failed to get pod resource")
	}

	headBucketPod, err := clientset.CoreV1().Pods(podOptions.Namespace).Create(ctx, pod, metav1.CreateOptions{})
	if err != nil {
		return errors.Wrap(err, "failed to create pod")
	}

	if err := k8sutil.WaitForPod(ctx, clientset, podOptions.Namespace, headBucketPod.Name, time.Minute*2); err != nil {
		return errors.Wrap(err, "failed to wait for pod")
	}

	logs, err := k8sutil.GetPodLogs(ctx, clientset, headBucketPod, true, nil)
	if err != nil {
		return errors.Wrap(err, "failed to get pod logs")
	}
	if len(logs) == 0 {
		return errors.New("no logs found")
	}

	type HeadBucketPodOutput struct {
		Success bool `json:"success"`
	}

	headBucketPodOutput := HeadBucketPodOutput{}
	if err := json.Unmarshal(logs, &headBucketPodOutput); err != nil {
		return errors.Wrapf(err, "failed to unmarshal %s pod logs", headBucketPod.Name)
	}

	if !headBucketPodOutput.Success {
		return errors.Wrapf(err, "failed to head bucket, please check %s pod logs for more details", headBucketPod.Name)
	}

	// only delete the pod on success
	clientset.CoreV1().Pods(podOptions.Namespace).Delete(ctx, headBucketPod.Name, metav1.DeleteOptions{})

	return nil
}

func s3BucketPod(clientset kubernetes.Interface, podOptions S3OpsPodOptions, command []string) (*corev1.Pod, error) {
	var securityContext corev1.PodSecurityContext
	if !podOptions.IsOpenShift {
		securityContext = corev1.PodSecurityContext{
			RunAsUser: util.IntPointer(1001),
			FSGroup:   util.IntPointer(1001),
		}
	}

	kotsadmTag := kotsadmversion.KotsadmTag(kotsadmtypes.KotsadmOptions{}) // default tag
	image := fmt.Sprintf("kotsadm/kotsadm:%s", kotsadmTag)
	imagePullSecrets := []corev1.LocalObjectReference{}

	if !kotsutil.IsKurl(clientset) || podOptions.Namespace != metav1.NamespaceDefault {
		var err error
		imageRewriteFn := kotsadmversion.ImageRewriteKotsadmRegistry(podOptions.Namespace, podOptions.RegistryOptions)
		image, imagePullSecrets, err = imageRewriteFn(image, false)
		if err != nil {
			return nil, errors.Wrap(err, "failed to rewrite image")
		}
	}

	env := []corev1.EnvVar{
		{
			Name:  "TMP_S3_ENDPOINT",
			Value: podOptions.Endpoint,
		},
		{
			Name:  "TMP_S3_BUCKET_NAME",
			Value: podOptions.BucketName,
		},
		{
			Name:  "TMP_S3_ACCESS_KEY_ID",
			Value: podOptions.AccessKeyID,
		},
		{
			Name:  "TMP_S3_SECRET_ACCESS_KEY",
			Value: podOptions.SecretAccessKey,
		},
	}

	pod := &corev1.Pod{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "Pod",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      podOptions.PodName,
			Namespace: podOptions.Namespace,
			Labels: map[string]string{
				"app": "kotsadm-s3-ops",
			},
		},
		Spec: corev1.PodSpec{
			SecurityContext:  &securityContext,
			RestartPolicy:    corev1.RestartPolicyOnFailure,
			ImagePullSecrets: imagePullSecrets,
			Containers: []corev1.Container{
				{
					Image:           image,
					ImagePullPolicy: corev1.PullIfNotPresent,
					Name:            "s3-bucket",
					Command:         command,
					Env:             env,
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
