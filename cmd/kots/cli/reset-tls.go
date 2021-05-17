package cli

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/pkg/errors"
	"github.com/replicatedhq/kots/pkg/k8sutil"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	kuberneteserrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func ResetTLSCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:           "reset-tls [namespace]",
		Short:         "Reverts kurl_proxy to a self-signed TLS certificate",
		Long:          ``,
		SilenceUsage:  true,
		SilenceErrors: false,
		PreRun: func(cmd *cobra.Command, args []string) {
			viper.BindPFlags(cmd.Flags())
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			v := viper.GetViper()

			namespace := v.GetString("namespace")
			if len(args) == 1 {
				namespace = args[0]
			} else if len(args) > 1 {
				fmt.Printf("more than one argument supplied: %+v\n", args)
				os.Exit(1)
			}

			if namespace == "" {
				fmt.Printf("a namespace must be provided as an argument or via -n/--namespace\n")
				os.Exit(1)
			}

			err := deleteKotsTLSSecret(namespace)
			if err != nil {
				return err
			}

			err = resetKurlProxyPod(namespace)
			if err != nil {
				return err
			}

			err = checkTLSSecret(namespace, time.Second*10)
			if err != nil {
				return err
			}

			return nil
		},
	}

	return cmd
}

func deleteKotsTLSSecret(namespace string) error {
	clientset, err := k8sutil.GetClientset()
	if err != nil {
		return errors.Wrap(err, "failed to create k8s client")
	}

	existingSecret, err := clientset.CoreV1().Secrets(namespace).Get(context.TODO(), "kotsadm-tls", metav1.GetOptions{})
	if err != nil {
		if !kuberneteserrors.IsNotFound(err) {
			return errors.Wrap(err, "failed to lookup secret")
		}
	}

	if existingSecret.Name != "" {
		err = clientset.CoreV1().Secrets(namespace).Delete(context.TODO(), existingSecret.Name, metav1.DeleteOptions{})
		if err != nil {
			return errors.Wrap(err, "failed to delete secret")
		}
	}

	return nil
}

func resetKurlProxyPod(namespace string) error {
	clientset, err := k8sutil.GetClientset()
	if err != nil {
		return errors.Wrap(err, "failed to create k8s client")
	}

	kurlProxyPods, err := clientset.CoreV1().Pods(namespace).List(context.TODO(), metav1.ListOptions{LabelSelector: "app=kurl-proxy"})
	if err != nil || len(kurlProxyPods.Items) == 0 {
		return errors.Wrap(err, "failed to list kurl_proxy pods before restarting")
	}

	// loop through and delete the pods
	for _, pod := range kurlProxyPods.Items {
		err = clientset.CoreV1().Pods(namespace).Delete(context.TODO(), pod.Name, metav1.DeleteOptions{})
		if err != nil {
			return errors.Wrap(err, "could not delete kurl_proxy pod")
		}
	}

	return nil
}

func checkTLSSecret(namespace string, timeout time.Duration) error {
	clientset, err := k8sutil.GetClientset()
	if err != nil {
		return errors.Wrap(err, "failed to create k8s client")
	}

	kurlProxyPods, err := clientset.CoreV1().Pods(namespace).List(context.TODO(), metav1.ListOptions{LabelSelector: "app=kurl-proxy"})
	if err != nil {
		return errors.Wrap(err, "failed to list kurl_proxy pods before restarting")
	}
	if len(kurlProxyPods.Items) == 0 {
		return errors.Wrap(err, "kurl_proxy pod not found after restart")
	}

	start := time.Now()

	for {
		_, err = clientset.CoreV1().Secrets(namespace).Get(context.TODO(), "kotsadm-tls", metav1.GetOptions{})
		if err != nil {
			if !kuberneteserrors.IsNotFound(err) {
				return errors.Wrap(err, "failed to lookup secret")
			}
		} else {
			break
		}

		time.Sleep(time.Second)

		if time.Now().Sub(start) > timeout {
			return fmt.Errorf("timeout waiting for tls configuration")
		}
	}

	return nil
}
