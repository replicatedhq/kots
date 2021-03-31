package client

import (
	"context"
	"encoding/base64"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/pkg/errors"
	"github.com/replicatedhq/kots/kotsadm/operator/pkg/applier"
	"github.com/replicatedhq/kots/kotsadm/operator/pkg/util"
	"github.com/replicatedhq/kots/pkg/k8sutil"
	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	batchv1beta1 "k8s.io/api/batch/v1beta1"
	corev1 "k8s.io/api/core/v1"
	kuberneteserrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8sruntime "k8s.io/apimachinery/pkg/runtime"
	k8sschema "k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
)

var metadataAccessor = meta.NewAccessor()

type applyResult struct {
	hasErr      bool
	multiStdout [][]byte
	multiStderr [][]byte
}

func (c *Client) diffAndRemovePreviousManifests(applicationManifests ApplicationManifests) error {
	decodedPrevious, err := base64.StdEncoding.DecodeString(applicationManifests.PreviousManifests)
	if err != nil {
		return errors.Wrap(err, "failed to decode previous manifests")
	}

	decodedCurrent, err := base64.StdEncoding.DecodeString(applicationManifests.Manifests)
	if err != nil {
		return errors.Wrap(err, "failed to decode manifests")
	}

	targetNamespace := c.TargetNamespace
	if applicationManifests.Namespace != "." {
		targetNamespace = applicationManifests.Namespace
	}

	// we need to find the gvk+names that are present in the previous, but not in the current and then remove them
	// namespaces that were remoed from YAML and added to additionalNamespaces should not be removed
	decodedPreviousStrings := strings.Split(string(decodedPrevious), "\n---\n")

	type previousObject struct {
		spec   string
		delete bool
	}
	decodedPreviousMap := map[string]previousObject{}
	for _, decodedPreviousString := range decodedPreviousStrings {
		k, o := GetGVKWithNameAndNs([]byte(decodedPreviousString), targetNamespace)

		delete := true
		if o.APIVersion == "v1" && o.Kind == "Namespace" {
			for _, n := range applicationManifests.AdditionalNamespaces {
				if o.Metadata.Name == n {
					delete = false
					break
				}
			}
		}
		if applicationManifests.IsRestore {
			if excludeLabel, exists := o.Metadata.Labels["velero.io/exclude-from-backup"]; exists {
				if excludeLabel == "true" {
					delete = false
				}
			}
		}
		decodedPreviousMap[k] = previousObject{
			spec:   decodedPreviousString,
			delete: delete,
		}
	}

	// now get the current names
	decodedCurrentStrings := strings.Split(string(decodedCurrent), "\n---\n")
	decodedCurrentMap := map[string]string{}
	for _, decodedCurrentString := range decodedCurrentStrings {
		k, _ := GetGVKWithNameAndNs([]byte(decodedCurrentString), targetNamespace)
		decodedCurrentMap[k] = decodedCurrentString
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

	allPVCs := make([]string, 0)
	for k, previous := range decodedPreviousMap {
		if _, ok := decodedCurrentMap[k]; ok {
			continue
		}
		if !previous.delete {
			continue
		}

		obj, gvk, err := parseK8sYaml([]byte(previous.spec))
		if err != nil {
			log.Printf("deleting unidentified manifest. unable to parse error: %s", err.Error())
		}

		group := ""
		kind := ""
		namespace := targetNamespace
		name := ""

		if obj != nil {
			if n, _ := metadataAccessor.Namespace(obj); n != "" {
				namespace = n
			}
			name, _ = metadataAccessor.Name(obj)
		}

		if obj != nil && gvk != nil {
			group = gvk.Group
			kind = gvk.Kind
			log.Printf("deleting manifest(s): %s/%s/%s", group, kind, name)

			pvcs, err := getPVCs(namespace, obj, gvk)
			if err != nil {
				return errors.Wrap(err, "failed to list PVCs")
			}
			allPVCs = append(allPVCs, pvcs...)
		}

		wait := applicationManifests.Wait
		if gvk != nil && gvk.Kind == "PersistentVolumeClaim" {
			// blocking on PVC delete will create a deadlock if
			// it's used by a pod that has not been deleted yet.
			wait = false
		}

		stdout, stderr, err := kubernetesApplier.Remove(namespace, []byte(previous.spec), wait)
		if err != nil {
			log.Printf("stdout (delete) = %s", stdout)
			log.Printf("stderr (delete) = %s", stderr)
			log.Printf("error: %s", err.Error())
		} else {
			log.Printf("manifest(s) deleted: %s/%s/%s", group, kind, name)
		}
	}

	if applicationManifests.ClearPVCs {
		log.Printf("deleting pvcs: %s", strings.Join(allPVCs, ","))
		// TODO: multi-namespace support
		err := deletePVCs(targetNamespace, allPVCs)
		if err != nil {
			return errors.Wrap(err, "failed to delete PVCs")
		}
	}

	for _, namespace := range applicationManifests.ClearNamespaces {
		log.Printf("Ensuring all %s objects have been removed from namespace %s\n", applicationManifests.AppSlug, namespace)
		sleepTime := time.Second * 2
		for i := 60; i >= 0; i-- { // 2 minute wait, 60 loops with 2 second sleep
			gone, err := c.clearNamespace(applicationManifests.AppSlug, namespace, applicationManifests.IsRestore)
			if err != nil {
				log.Printf("Failed to check if app %s objects have been removed from namespace %s: %v\n", applicationManifests.AppSlug, namespace, err)
			} else if gone {
				break
			}
			if i == 0 {
				return fmt.Errorf("Failed to clear app %s from namespace %s\n", applicationManifests.AppSlug, namespace)
			}
			log.Printf("Namespace %s still has objects from app %s: sleeping...\n", namespace, applicationManifests.AppSlug)
			time.Sleep(sleepTime)
		}
		log.Printf("Namepsace %s successfully cleared of app %s\n", namespace, applicationManifests.AppSlug)
	}
	if len(applicationManifests.ClearNamespaces) > 0 {
		// Extra time in case the app-slug annotation was not being used.
		time.Sleep(time.Second * 20)
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

	_, err = clientset.CoreV1().Namespaces().Get(context.TODO(), name, metav1.GetOptions{})
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

		_, err = clientset.CoreV1().Namespaces().Create(context.TODO(), namespace, metav1.CreateOptions{})
		if err != nil {
			return errors.Wrap(err, "failed to create namespace")
		}
	}

	return nil
}

func (c *Client) ensureResourcesPresent(applicationManifests ApplicationManifests) (*applyResult, error) {
	targetNamespace := c.TargetNamespace
	if applicationManifests.Namespace != "." {
		targetNamespace = applicationManifests.Namespace
	}

	kubernetesApplier, err := c.getApplier(applicationManifests.KubectlVersion)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get applier")
	}

	decoded, err := base64.StdEncoding.DecodeString(applicationManifests.Manifests)
	if err != nil {
		return nil, errors.Wrap(err, "failed to decode manifests")
	}

	firstApplyDocs, otherDocs, err := splitMutlidocYAMLIntoFirstApplyAndOthers(decoded)
	if err != nil {
		return nil, errors.Wrap(err, "failed to split decoded into crds and other")
	}

	// We don't dry run if there's a crd becasue there's a likely chance that the
	// other docs has a custom resource using it
	shouldDryRun := firstApplyDocs == nil
	if shouldDryRun {
		byNamespace, err := docsByNamespace(decoded, targetNamespace)
		if err != nil {
			return nil, errors.Wrap(err, "failed to get docs by requested namespace")
		}

		for requestedNamespace, docs := range byNamespace {
			if len(docs) == 0 {
				continue
			}

			log.Printf("dry run applying manifests(s) in requested namespace: %s", requestedNamespace)
			dryrunStdout, dryrunStderr, dryRunErr := kubernetesApplier.Apply(requestedNamespace, applicationManifests.AppSlug, docs, true, applicationManifests.Wait, applicationManifests.AnnotateSlug)
			if dryRunErr != nil {
				log.Printf("stdout (dryrun) = %s", dryrunStdout)
				log.Printf("stderr (dryrun) = %s", dryrunStderr)
				log.Printf("error: %s", dryRunErr.Error())
			} else {
				log.Printf("dry run applied manifests(s) in requested namespace: %s", requestedNamespace)
			}

			if dryRunErr != nil {
				if err := c.sendResult(applicationManifests, true, dryrunStdout, dryrunStderr, []byte{}, []byte{}); err != nil {
					return nil, errors.Wrap(err, "failed to report dry run status")
				}

				return nil, nil // don't return an error because execution is proper, the api now has the error
			}
		}

	}

	if len(firstApplyDocs) > 0 {
		log.Println("applying first apply docs (CRDs, Namespaces)")

		// CRDs don't have namespaces, so we can skip splitting

		applyStdout, applyStderr, applyErr := kubernetesApplier.Apply("", applicationManifests.AppSlug, firstApplyDocs, false, applicationManifests.Wait, applicationManifests.AnnotateSlug)
		if applyErr != nil {
			log.Printf("stdout (first apply) = %s", applyStdout)
			log.Printf("stderr (first apply) = %s", applyStderr)
			log.Printf("error (CRDS): %s", applyErr.Error())

			if err := c.sendResult(applicationManifests, applyErr != nil, []byte{}, []byte{}, applyStdout, applyStderr); err != nil {
				return nil, errors.Wrap(err, "failed to report crd status")
			}

			return nil, nil
		} else {
			log.Println("custom resource definition(s) applied")
		}

		// Give the API server a minute (well, 5 seconds) to cache the CRDs
		time.Sleep(time.Second * 5)
	}

	byNamespace, err := docsByNamespace(otherDocs, targetNamespace)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get docs by requested namespace")
	}

	var hasErr bool
	var multiStdout, multiStderr [][]byte
	for requestedNamespace, docs := range byNamespace {
		if len(docs) == 0 {
			continue
		}

		log.Printf("applying manifest(s) in namespace %s", requestedNamespace)
		applyStdout, applyStderr, applyErr := kubernetesApplier.Apply(requestedNamespace, applicationManifests.AppSlug, docs, false, applicationManifests.Wait, applicationManifests.AnnotateSlug)
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

	result := &applyResult{
		hasErr:      hasErr,
		multiStderr: multiStderr,
		multiStdout: multiStdout,
	}
	return result, nil
}

func (c *Client) clearNamespace(slug string, namespace string, isRestore bool) (bool, error) {
	cfg, err := k8sutil.GetClusterConfig()
	if err != nil {
		return false, errors.Wrap(err, "failed to get cluster config")
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
		unstructuredList, err := dyn.Resource(gvr).Namespace(namespace).List(context.TODO(), metav1.ListOptions{})
		if unstructuredList == nil {
			if err != nil {
				log.Printf("failed to list namespace resources: %s", err.Error())
			}
			continue
		}
		for _, u := range unstructuredList.Items {
			labels := u.GetLabels()
			if isRestore {
				if excludeLabel, exists := labels["velero.io/exclude-from-backup"]; exists {
					if excludeLabel == "true" {
						continue
					}
				}
			}

			annotations := u.GetAnnotations()
			if annotations["kots.io/app-slug"] == slug {
				clear = false
				if u.GetDeletionTimestamp() != nil {
					log.Printf("%s %s is pending deletion\n", gvr, u.GetName())
					continue
				}
				log.Printf("Deleting %s/%s/%s\n", namespace, gvr, u.GetName())
				err := dyn.Resource(gvr).Namespace(namespace).Delete(context.TODO(), u.GetName(), metav1.DeleteOptions{})
				if err != nil {
					log.Printf("Resource %s (%s) in namepsace %s could not be deleted: %v\n", u.GetName(), gvr, namespace, err)
					return false, err
				}
			}
		}
	}

	return clear, nil
}

func parseK8sYaml(doc []byte) (k8sruntime.Object, *k8sschema.GroupVersionKind, error) {
	decode := scheme.Codecs.UniversalDeserializer().Decode
	obj, gvk, err := decode(doc, nil, nil)
	if err != nil {
		return nil, nil, errors.Wrap(err, "failed to decode k8s yaml")
	}
	return obj, gvk, err
}

func getPVCs(targetNamespace string, obj k8sruntime.Object, gvk *k8sschema.GroupVersionKind) ([]string, error) {
	var err error
	var pods []*corev1.Pod

	ns := func(objNs string) string {
		if objNs != "" {
			return objNs
		}
		return targetNamespace
	}

	if gvk.Group == "apps" && gvk.Version == "v1" && gvk.Kind == "Deployment" {
		o := obj.(*appsv1.Deployment)
		pods, err = findPodsByOwner(o.Name, ns(o.Namespace), gvk)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to find pods for deployment %s", o.Name)
		}
	} else if gvk.Group == "apps" && gvk.Version == "v1" && gvk.Kind == "StatefulSet" {
		o := obj.(*appsv1.StatefulSet)
		pods, err = findPodsByOwner(o.Name, ns(o.Namespace), gvk)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to find pods for stateful set %s", o.Name)
		}
	} else if gvk.Group == "batch" && gvk.Version == "v1" && gvk.Kind == "Job" {
		o := obj.(*batchv1.Job)
		pods, err = findPodsByOwner(o.Name, ns(o.Namespace), gvk)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to find pods for job %s", o.Name)
		}
	} else if gvk.Group == "batch" && gvk.Version == "v1beta1" && gvk.Kind == "CronJob" {
		o := obj.(*batchv1beta1.CronJob)
		pods, err = findPodsByOwner(o.Name, ns(o.Namespace), gvk)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to find pods for cron job %s", o.Name)
		}
	} else if gvk.Group == "" && gvk.Version == "v1" && gvk.Kind == "Pod" {
		o := obj.(*corev1.Pod)
		pod, err := findPodByName(o.Name, ns(o.Namespace))
		if err != nil {
			if !kuberneteserrors.IsNotFound(errors.Cause(err)) {
				return nil, errors.Wrapf(err, "failed to find pod %s", o.Name)
			}
		}
		if pod != nil {
			pods = []*corev1.Pod{pod}
		}
	}

	pvcs := make([]string, 0)
	for _, pod := range pods {
		for _, v := range pod.Spec.Volumes {
			if v.PersistentVolumeClaim != nil {
				pvcs = append(pvcs, v.PersistentVolumeClaim.ClaimName)
			}
		}
	}

	return pvcs, nil
}

func deletePVCs(namespace string, pvcs []string) error {
	clientset, err := k8sutil.GetClientset()
	if err != nil {
		return errors.Wrap(err, "failed to get k8s client set")
	}

	for _, pvc := range pvcs {
		grace := int64(0)
		policy := metav1.DeletePropagationBackground
		opts := metav1.DeleteOptions{
			GracePeriodSeconds: &grace,
			PropagationPolicy:  &policy,
		}
		log.Printf("deleting pvc: %s", pvc)
		err := clientset.CoreV1().PersistentVolumeClaims(namespace).Delete(context.TODO(), pvc, opts)
		if err != nil && !kuberneteserrors.IsNotFound(err) {
			return errors.Wrapf(err, "failed to delete pvc %s", pvc)
		}
	}

	return nil
}
