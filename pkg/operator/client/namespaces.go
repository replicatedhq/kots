package client

import (
	"context"
	"log"

	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	kuberneteserrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"
)

func (c *Client) runNamespacesInformer() error {
	restconfig, err := rest.InClusterConfig()
	if err != nil {
		return errors.Wrap(err, "failed to get in cluster config")
	}
	clientset, err := kubernetes.NewForConfig(restconfig)
	if err != nil {
		return errors.Wrap(err, "failed to get new kubernetes client")
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

				decode := scheme.Codecs.UniversalDeserializer().Decode
				obj, _, err := decode([]byte(c.imagePullSecret), nil, nil)
				if err != nil {
					log.Print(err)
					return
				}

				secret := obj.(*corev1.Secret)
				secret.Namespace = addedNamespace.Name

				foundSecret, err := clientset.CoreV1().Secrets(addedNamespace.Name).Get(context.TODO(), secret.Name, metav1.GetOptions{})
				if err != nil {
					if kuberneteserrors.IsNotFound(err) {
						// create it
						_, err := clientset.CoreV1().Secrets(addedNamespace.Name).Create(context.TODO(), secret, metav1.CreateOptions{})
						if err != nil {
							log.Print(err)
							return
						}
					} else {
						log.Print(err)
						return
					}
				} else {
					// Update it
					foundSecret.Data[".dockerconfigjson"] = secret.Data[".dockerconfigjson"]
					if _, err := clientset.CoreV1().Secrets(addedNamespace.Name).Update(context.TODO(), secret, metav1.UpdateOptions{}); err != nil {
						log.Print(err)
						return
					}
				}
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
