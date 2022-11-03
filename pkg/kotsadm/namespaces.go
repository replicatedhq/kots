package kotsadm

import (
	"context"

	"github.com/pkg/errors"
	"github.com/replicatedhq/kots/pkg/kotsadm/types"
	"github.com/replicatedhq/kots/pkg/kotsutil"
	"github.com/replicatedhq/kots/pkg/logger"
	corev1 "k8s.io/api/core/v1"
	kuberneteserrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

func ensureAdditionalNamespaces(deployOptions *types.DeployOptions, clientset *kubernetes.Clientset, log *logger.CLILogger) error {
	// try to parse
	if deployOptions.ApplicationMetadata == nil {
		return nil
	}

	application, err := kotsutil.LoadKotsAppFromContents(deployOptions.ApplicationMetadata)
	if err != nil {
		return errors.Wrap(err, "failed to load kots app from contents")
	}

	for _, additionalNamespace := range application.Spec.AdditionalNamespaces {
		// We support "*" for additional namespaces to handle pullsecret propagation
		if additionalNamespace == "*" {
			continue
		}

		_, err := clientset.CoreV1().Namespaces().Get(context.TODO(), additionalNamespace, metav1.GetOptions{})
		if kuberneteserrors.IsNotFound(err) {
			log.ChildActionWithSpinner("Creating namespace %s", additionalNamespace)
			namespace := &corev1.Namespace{
				TypeMeta: metav1.TypeMeta{
					APIVersion: "v1",
					Kind:       "Namespace",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name: additionalNamespace,
				},
			}

			_, err = clientset.CoreV1().Namespaces().Create(context.TODO(), namespace, metav1.CreateOptions{})
			if err != nil {
				log.FinishSpinnerWithError()
				return errors.Wrap(err, "failed to create namespace")
			}
			log.FinishChildSpinner()
		}
	}

	return nil
}
