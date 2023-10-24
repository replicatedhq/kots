package embeddedcluster

import (
	corev1client "k8s.io/client-go/kubernetes/typed/core/v1"
	"k8s.io/client-go/rest"
)

// SyncExec returns exitcode, stdout, stderr. A non-zero exit code from the command is not considered an error.
func SyncExec(coreClient corev1client.CoreV1Interface, clientConfig *rest.Config, ns, pod, container string, command ...string) (int, string, string, error) {
	return 0, "", "", nil
}
