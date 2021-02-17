package kurl

import (
	"context"
	"fmt"

	"github.com/pkg/errors"
	"github.com/replicatedhq/kots/pkg/logger"
	"go.uber.org/zap"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

var ErrNoEkco = errors.New("Ekco not found")

func DeleteNode(ctx context.Context, client kubernetes.Interface, restconfig *rest.Config, node *corev1.Node) error {
	runningField := map[string]string{"status.phase": "Running"}
	ekcoLabel := map[string]string{"app": "ekc-operator"}
	opts := metav1.ListOptions{
		FieldSelector: fields.SelectorFromSet(runningField).String(),
		LabelSelector: labels.SelectorFromSet(ekcoLabel).String(),
	}
	pods, err := client.CoreV1().Pods("kurl").List(ctx, opts)
	if err != nil {
		return errors.Wrap(err, "list ekco pods")
	}
	if len(pods.Items) == 0 {
		return ErrNoEkco
	}
	pod := pods.Items[0]

	logger.Debugf("Executing purge-node %s in pod %s/%s", node.Name, pod.Namespace, pod.Name)
	statusCode, stdout, stderr, err := SyncExec(client.CoreV1(), restconfig, pod.Namespace, pod.Name, "ekc-operator", "ekco", "purge-node", node.Name)
	if err != nil {
		return errors.Wrap(err, "executed purge-node in ekco pod")
	}
	logger.Debug("Executed purge-node in ekco pod",
		zap.String("node", node.Name),
		zap.Int("status-code", statusCode),
		zap.String("stdout", stdout),
		zap.String("stderr", stderr))
	if statusCode != 0 {
		return fmt.Errorf("execute purge-node failed with status code %d", statusCode)
	}
	logger.Infof("Successfully purged node %s", node.Name)

	return nil
}
