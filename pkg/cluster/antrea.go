package cluster

import (
	"bufio"
	"context"
	_ "embed"
	"encoding/json"
	"io"
	"strings"

	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer/yaml"
	"k8s.io/apimachinery/pkg/types"
	utilyaml "k8s.io/apimachinery/pkg/util/yaml"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/discovery/cached/memory"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/restmapper"
	"k8s.io/client-go/tools/clientcmd"
)

// antreaManifests will contain the yaml necessary to install the antrea cni into the embedded cluster
//go:embed antrea.yaml
var antreaManifests string

// note: https://github.com/antrea-io/antrea/releases/download/v1.2.2/antrea.yml

func installCNI(kubeconfigPath string) error {
	config, err := clientcmd.BuildConfigFromFlags("", kubeconfigPath)
	if err != nil {
		return errors.Wrap(err, "build config")
	}

	// Create dynamic client for loading CNI resources in the cluster
	dyn, err := dynamic.NewForConfig(config)
	if err != nil {
		return errors.Wrap(err, "new dynamic config")
	}

	// Create DiscoveryClient
	dc, err := discovery.NewDiscoveryClientForConfig(config)
	if err != nil {
		return errors.Wrap(err, "new discovery client for config")
	}

	mapper := restmapper.NewDeferredDiscoveryRESTMapper(memory.NewMemCacheClient(dc))

	var decUnstructured = yaml.NewDecodingSerializer(unstructured.UnstructuredJSONScheme)

	// Pass one: install CRDs
	if err = applyUnstructuredManifests(antreaManifests, dc, dyn, mapper, decUnstructured, true); err != nil {
		return errors.Wrap(err, "apply unstructured CRDs")
	}

	// Pass two: install CNI objects
	if err = applyUnstructuredManifests(antreaManifests, dc, dyn, mapper, decUnstructured, false); err != nil {
		return errors.Wrap(err, "apply unstructured manifests")
	}

	return nil
}

// Reference: https://ymmt2005.hatenablog.com/entry/2020/04/14/An_example_of_using_dynamic_client_of_k8s.io/client-go
func applyUnstructuredManifests(manifests string, dc *discovery.DiscoveryClient, dyn dynamic.Interface, mapper *restmapper.DeferredDiscoveryRESTMapper, decUnstructured runtime.Serializer, filterCrds bool) error {

	multidocReader := utilyaml.NewYAMLReader(bufio.NewReader(strings.NewReader(manifests)))
	for {
		buf, err := multidocReader.Read()
		if err != nil {
			if err == io.EOF {
				break
			}

			return errors.Wrap(err, "reading multidoc")
		}

		obj := &unstructured.Unstructured{}
		_, gvk, err := decUnstructured.Decode(buf, nil, obj)
		if err != nil {
			return errors.Wrap(err, "decode doc")
		}

		if filterCrds && gvk.Kind != "CustomResourceDefinition" {
			continue
		}

		mapping, err := mapper.RESTMapping(gvk.GroupKind(), gvk.Version)
		if err != nil {
			return errors.Wrap(err, "rest mapping")
		}

		var dr dynamic.ResourceInterface
		if mapping.Scope.Name() == meta.RESTScopeNameNamespace {
			// namespaced resources should specify the namespace
			dr = dyn.Resource(mapping.Resource).Namespace(obj.GetNamespace())
		} else {
			// for cluster-wide resources
			dr = dyn.Resource(mapping.Resource)
		}

		data, err := json.Marshal(obj)
		if err != nil {
			return errors.Wrap(err, "json marshal")
		}

		_, err = dr.Patch(context.Background(), obj.GetName(), types.ApplyPatchType, data, metav1.PatchOptions{
			FieldManager: "kots",
		})
		if err != nil {
			return errors.Wrap(err, "patch")
		}
	}
	return nil
}
