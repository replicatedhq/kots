package kotsadm

import (
	"context"
	"fmt"
	"os"

	"github.com/manifoldco/promptui"
	"github.com/pkg/errors"
	"github.com/replicatedhq/kots/pkg/kotsadm/types"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

func maybeGetNamespaceLimitRanges(clientset kubernetes.Interface, namespace string) (*corev1.LimitRange, error) {
	limitRanges, err := clientset.CoreV1().LimitRanges(namespace).List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return nil, errors.Wrap(err, "failed to list limit ranges")
	}

	if len(limitRanges.Items) == 0 {
		return nil, nil
	}

	return &limitRanges.Items[0], nil
}

func getSize(deployOptions types.DeployOptions, label string, size resource.Quantity) (resource.Quantity, error) {
	if deployOptions.LimitRange != nil {
		var allowedMax *resource.Quantity
		var allowedMin *resource.Quantity

		for _, limit := range deployOptions.LimitRange.Spec.Limits {
			if limit.Type == corev1.LimitTypePersistentVolumeClaim {
				max, ok := limit.Max["storage"]
				if ok {
					allowedMax = &max
				}

				min, ok := limit.Min["storage"]
				if ok {
					allowedMin = &min
				}
			}
		}

		newSize := promptForSizeIfNotBetween(label, &size, allowedMin, allowedMax)
		if newSize == nil {
			os.Exit(-1)
		}

		size = *newSize
	}

	return size, nil
}

func promptForSizeIfNotBetween(label string, desired *resource.Quantity, min *resource.Quantity, max *resource.Quantity) *resource.Quantity {
	actualSize := desired

	if max != nil {
		if max.Cmp(*desired) == -1 {
			/// desired is too big
			actualSize = max
		}
	}
	if min != nil {
		if min.Cmp(*desired) == 1 {
			/// desired is too small, yeap, you read that right
			actualSize = min
		}
	}

	if actualSize.Cmp(*desired) == 0 {
		return desired
	}

	prompt := promptui.Prompt{
		Label:     fmt.Sprintf("The storage request for %s is not acceptable for the current namespace. KOTS recommends a size of %s, but will attempt to proceed with %s to meet the namespace limits. Do you want to continue", label, desired.String(), actualSize.String()),
		IsConfirm: true,
	}

	for {
		_, err := prompt.Run()
		if err != nil {
			if err == promptui.ErrInterrupt {
				os.Exit(-1)
			}
			continue
		}

		return actualSize
	}
}
