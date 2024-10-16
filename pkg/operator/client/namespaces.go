package client

import (
	"log"

	"github.com/pkg/errors"
	"github.com/replicatedhq/kots/pkg/k8sutil"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/tools/cache"
)

func (c *Client) runNamespacesInformer() error {
	clientset, err := k8sutil.GetClientset()
	if err != nil {
		return errors.Wrap(err, "failed to get clientset")
	}

	c.namespaceStopChan = make(chan struct{})

	factory := informers.NewSharedInformerFactory(clientset, 0)
	nsInformer := factory.Core().V1().Namespaces().Informer()

	nsInformer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			addedNamespace := obj.(*corev1.Namespace)

			for _, watchedNamespace := range c.watchedNamespaces {
				deployImagePullSecret := false

				if watchedNamespace == "*" {
					deployImagePullSecret = true
				} else {
					if watchedNamespace == addedNamespace.Name {
						deployImagePullSecret = true
					}
				}

				if !deployImagePullSecret {
					continue
				}

				if err := c.ensureImagePullSecretsPresent(addedNamespace.Name, c.imagePullSecrets); err != nil {
					// we don't fail here...
					log.Printf("error ensuring image pull secrets for namespace %s: %s", addedNamespace.Name, err.Error())
				}

				if err := c.ensureEmbeddedClusterCAPresent(addedNamespace.Name); err != nil {
					// we don't fail here...
					log.Printf("error ensuring cluster ca present for namespace %s: %s", addedNamespace.Name, err.Error())
				}

				c.ApplyHooksInformer([]string{addedNamespace.Name})
			}
		},
	})

	go nsInformer.Run(c.namespaceStopChan)

	return nil
}

func (c *Client) shutdownNamespacesInformer() {
	if c.namespaceStopChan != nil {
		c.namespaceStopChan <- struct{}{}
	}
	c.namespaceStopChan = nil
}
