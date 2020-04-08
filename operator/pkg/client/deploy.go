package client

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/pkg/errors"
	"github.com/replicatedhq/kotsadm/operator/pkg/applier"
	"github.com/replicatedhq/kotsadm/operator/pkg/util"
	corev1 "k8s.io/api/core/v1"
	kuberneteserrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
)

func (c *Client) diffAndRemovePreviousManifests(applicationManifests ApplicationManifests) error {
	decodedPrevious, err := base64.StdEncoding.DecodeString(applicationManifests.PreviousManifests)
	if err != nil {
		return errors.Wrap(err, "failed to decode previous manifests")
	}

	decodedCurrent, err := base64.StdEncoding.DecodeString(applicationManifests.Manifests)
	if err != nil {
		return errors.Wrap(err, "failed to decode manifests")
	}

	// we need to find the gvk+names that are present in the previous, but not in the current and then remove them
	decodedPreviousStrings := strings.Split(string(decodedPrevious), "\n---\n")
	decodedPreviousMap := map[string]string{}
	for _, decodedPreviousString := range decodedPreviousStrings {
		decodedPreviousMap[GetGVKWithName([]byte(decodedPreviousString))] = decodedPreviousString
	}

	// now get the current names
	decodedCurrentStrings := strings.Split(string(decodedCurrent), "\n---\n")
	decodedCurrentMap := map[string]string{}
	for _, decodedCurrentString := range decodedCurrentStrings {
		decodedCurrentMap[GetGVKWithName([]byte(decodedCurrentString))] = decodedCurrentString
	}

	// now remove anything that's in previous but not in current
	kubectl, err := util.FindKubectlVersion(applicationManifests.KubectlVersion)
	if err != nil {
		return errors.Wrap(err, "failed to find kubectl")
	}
	config, err := rest.InClusterConfig()
	if err != nil {
		return errors.Wrap(err, "failed to get in cluster config")
	}

	// this is pretty raw, and required kubectl...  we should
	// consider some other options here?
	kubernetesApplier := applier.NewKubectl(kubectl, "", "", config)
	targetNamespace := c.TargetNamespace
	if applicationManifests.Namespace != "." {
		targetNamespace = applicationManifests.Namespace
	}

	for k, oldContents := range decodedPreviousMap {
		if _, ok := decodedCurrentMap[k]; !ok {
			gv, k, n, err := ParseSimpleGVK([]byte(oldContents))
			if err != nil {
				log.Printf("deleting unidentified manifest. unable to parse error: %s", err.Error())
			} else {
				log.Printf("deleting manifest(s): %s/%s/%s", gv, k, n)
			}
			stdout, stderr, err := kubernetesApplier.Remove(targetNamespace, []byte(oldContents), applicationManifests.Wait)
			if err != nil {
				log.Printf("stdout (delete) = %s", stdout)
				log.Printf("stderr (delete) = %s", stderr)
				log.Printf("error: %s", err.Error())
			} else {
				log.Printf("manifest(s) deleted: %s/%s/%s", gv, k, n)
			}
		}
	}

	for _, namespace := range applicationManifests.ClearNamespaces {
		log.Printf("Ensuring all %s objects have been removed from namespace %s\n", applicationManifests.AppSlug, namespace)
		for i := 10; i >= 0; i-- {
			gone, err := c.clearNamespace(applicationManifests.AppSlug, namespace)
			if err != nil {
				log.Printf("Failed to check if app %s objects have been removed from namespace %s: %v\n", applicationManifests.AppSlug, namespace, err)
			} else if gone {
				break
			}
			if i == 0 {
				return fmt.Errorf("Failed to clear app %s from namespace %s\n", applicationManifests.AppSlug, namespace)
			}
			log.Printf("Namespace %s still has objects from app %s: sleeping...\n", namespace, applicationManifests.AppSlug)
			time.Sleep(time.Second * 2)
		}
		log.Printf("Namepsace %s successfully cleared of app %s\n", namespace, applicationManifests.AppSlug)
	}

	return nil
}

func (c *Client) ensureNamespacePresent(name string) error {
	restconfig, err := rest.InClusterConfig()
	if err != nil {
		return errors.Wrap(err, "failed to get in cluster config")
	}
	clientset, err := kubernetes.NewForConfig(restconfig)
	if err != nil {
		return errors.Wrap(err, "failed to get new kubernetes client")
	}

	_, err = clientset.CoreV1().Namespaces().Get(name, metav1.GetOptions{})
	if kuberneteserrors.IsNotFound(err) {
		namespace := &corev1.Namespace{
			TypeMeta: metav1.TypeMeta{
				APIVersion: "v1",
				Kind:       "Namespace",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name: name,
			},
		}

		_, err = clientset.CoreV1().Namespaces().Create(namespace)
		if err != nil {
			return errors.Wrap(err, "failed to create namespace")
		}
	}

	return nil
}

func (c *Client) ensureResourcesPresent(applicationManifests ApplicationManifests) error {
	targetNamespace := c.TargetNamespace
	if applicationManifests.Namespace != "." {
		targetNamespace = applicationManifests.Namespace
	}

	kubernetesApplier, err := c.getApplier(applicationManifests.KubectlVersion)
	if err != nil {
		return errors.Wrap(err, "failed to get applier")
	}

	decoded, err := base64.StdEncoding.DecodeString(applicationManifests.Manifests)
	if err != nil {
		return errors.Wrap(err, "failed to decode manifests")
	}

	customResourceDefinitions, otherDocs, err := splitMutlidocYAMLIntoCRDsAndOthers(decoded)
	if err != nil {
		return errors.Wrap(err, "failed to split decoded into crds and other")
	}

	// We don't dry run if there's a crd becasue there's a likely chance that the
	// other docs has a custom resource using it
	shouldDryRun := customResourceDefinitions == nil
	if shouldDryRun {
		byNamespace, err := docsByNamespace(decoded, targetNamespace)
		if err != nil {
			return errors.Wrap(err, "failed to get docs by requested namespace")
		}

		for requestedNamespace, docs := range byNamespace {
			if len(docs) == 0 {
				continue
			}

			log.Printf("dry run applying manifests(s) in requested namespace: %s", requestedNamespace)
			dryrunStdout, dryrunStderr, dryRunErr := kubernetesApplier.Apply(requestedNamespace, applicationManifests.AppSlug, docs, true, applicationManifests.Wait)
			if dryRunErr != nil {
				log.Printf("stdout (dryrun) = %s", dryrunStdout)
				log.Printf("stderr (dryrun) = %s", dryrunStderr)
				log.Printf("error: %s", dryRunErr.Error())
			} else {
				log.Printf("dry run applied manifests(s) in requested namespace: %s", requestedNamespace)
			}

			if dryRunErr != nil {
				if err := c.sendResult(applicationManifests, true, dryrunStdout, dryrunStderr, []byte{}, []byte{}); err != nil {
					return errors.Wrap(err, "failed to report dry run status")
				}

				return nil // don't return an error because execution is proper, the api now has the error
			}
		}

	}

	if len(customResourceDefinitions) > 0 {
		log.Println("applying custom resource definition(s)")

		// CRDs don't have namespaces, so we can skip splitting

		applyStdout, applyStderr, applyErr := kubernetesApplier.Apply("", applicationManifests.AppSlug, customResourceDefinitions, false, applicationManifests.Wait)
		if applyErr != nil {
			log.Printf("stdout (apply CRDS) = %s", applyStdout)
			log.Printf("stderr (apply CRDS) = %s", applyStderr)
			log.Printf("error (CRDS): %s", applyErr.Error())

			if err := c.sendResult(applicationManifests, applyErr != nil, []byte{}, []byte{}, applyStdout, applyStderr); err != nil {
				return errors.Wrap(err, "failed to report crd status")
			}

			return nil
		} else {
			log.Println("custom resource definition(s) applied")
		}

		// Give the API server a minute (well, 5 seconds) to cache the CRDs
		time.Sleep(time.Second * 5)
	}

	byNamespace, err := docsByNamespace(otherDocs, targetNamespace)
	if err != nil {
		return errors.Wrap(err, "failed to get docs by requested namespace")
	}

	var hasErr bool
	var multiStdout, multiStderr [][]byte
	for requestedNamespace, docs := range byNamespace {
		if len(docs) == 0 {
			continue
		}

		log.Printf("applying manifest(s) in namespace %s", requestedNamespace)
		applyStdout, applyStderr, applyErr := kubernetesApplier.Apply(requestedNamespace, applicationManifests.AppSlug, docs, false, applicationManifests.Wait)
		if applyErr != nil {
			log.Printf("stdout (apply) = %s", applyStdout)
			log.Printf("stderr (apply) = %s", applyStderr)
			log.Printf("error: %s", applyErr.Error())
			hasErr = true
		} else {
			log.Printf("manifest(s) applied in namespace %s", requestedNamespace)
		}
		if len(applyStdout) > 0 {
			multiStdout = append(multiStdout, applyStdout)
		}
		if len(applyStderr) > 0 {
			multiStderr = append(multiStderr, applyStderr)
		}
	}

	if err := c.sendResult(
		applicationManifests, hasErr, []byte{}, []byte{},
		bytes.Join(multiStdout, []byte("\n")), bytes.Join(multiStderr, []byte("\n")),
	); err != nil {
		return errors.Wrap(err, "failed to report status")
	}

	return nil
}

func (c *Client) clearNamespace(slug string, namespace string) (bool, error) {
	cfg, err := config.GetConfig()
	if err != nil {
		return false, errors.Wrap(err, "failed to get config")
	}
	disc, err := discovery.NewDiscoveryClientForConfig(cfg)
	if err != nil {
		return false, errors.Wrap(err, "failed to create discovery client")
	}
	dyn, err := dynamic.NewForConfig(cfg)
	if err != nil {
		return false, errors.Wrap(err, "failed to create dynamic client")
	}
	resourceList, err := disc.ServerPreferredNamespacedResources()
	if err != nil {
		// An application can define an APIService handled by a Deployment in the application itself.
		// In that case there will be a race condition listing resources in that API group and there
		// could be an error here. Most of the API groups would still be in the resource list so the
		// error is not terminal.
		log.Printf("Failed to list all resources: %v", err)
	}
	gvrs, err := discovery.GroupVersionResources(resourceList)
	if err != nil {
		return false, errors.Wrap(err, "failed to convert resource list to groupversionresource map")
	}
	// skip resources that don't have API endpoints or don't have applied objects
	var skip = sets.NewString(
		"/v1/bindings",
		"/v1/events",
		"extensions/v1beta1/replicationcontrollers",
		"apps/v1/controllerrevisions",
		"authentication.k8s.io/v1/tokenreviews",
		"authorization.k8s.io/v1/localsubjectaccessreviews",
		"authorization.k8s.io/v1/subjectaccessreviews",
		"authorization.k8s.io/v1/selfsubjectaccessreviews",
		"authorization.k8s.io/v1/selfsubjectrulesreviews",
	)
	clear := true
	for gvr := range gvrs {
		s := fmt.Sprintf("%s/%s/%s", gvr.Group, gvr.Version, gvr.Resource)
		if skip.Has(s) {
			continue
		}
		// there may be other resources that can't be listed besides what's in the skip set so ignore error
		unstructuredList, _ := dyn.Resource(gvr).Namespace(namespace).List(metav1.ListOptions{})
		for _, u := range unstructuredList.Items {
			annotations := u.GetAnnotations()
			if annotations["kots.io/app-slug"] == slug {
				clear = false
				if u.GetDeletionTimestamp() != nil {
					log.Printf("%s %s is pending deletion\n", gvr, u.GetName())
					continue
				}
				log.Printf("Deleting %s/%s/%s\n", namespace, gvr, u.GetName())
				err := dyn.Resource(gvr).Namespace(namespace).Delete(u.GetName(), &metav1.DeleteOptions{})
				if err != nil {
					log.Printf("Resource %s (%s) in namepsace %s could not be deleted: %v\n", u.GetName(), gvr, namespace, err)
					return false, err
				}
			}
		}
	}

	return clear, nil
}
