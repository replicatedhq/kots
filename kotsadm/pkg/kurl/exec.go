package kurl

import (
	"bytes"

	"github.com/pkg/errors"
	corev1client "k8s.io/client-go/kubernetes/typed/core/v1"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/remotecommand"
	"k8s.io/utils/exec"
)

// SyncExec returns exitcode, stdout, stderr. A non-zero exit code from the command is not considered an error.
func SyncExec(coreClient corev1client.CoreV1Interface, clientConfig *rest.Config, ns, pod, container string, command ...string) (int, string, string, error) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	req := coreClient.RESTClient().Post().
		Resource("pods").
		Name(pod).
		Namespace(ns).
		SubResource("exec").
		Param("container", container).
		Param("stdout", "true").
		Param("stderr", "true")
	for _, c := range command {
		req = req.Param("command", c)
	}

	executor, err := remotecommand.NewSPDYExecutor(clientConfig, "POST", req.URL())
	if err != nil {
		return 0, "", "", errors.Wrap(err, "create exec")
	}

	if err := executor.Stream(remotecommand.StreamOptions{
		Stdout: &stdout,
		Stderr: &stderr,
	}); err != nil {
		if err, ok := err.(exec.CodeExitError); ok {
			return err.Code, stdout.String(), stderr.String(), nil
		}
		return 0, stdout.String(), stderr.String(), errors.Wrap(err, "stream exec")
	}

	return 0, stdout.String(), stderr.String(), err
}
