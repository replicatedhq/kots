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
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/runtime/serializer/yaml"
	"k8s.io/apimachinery/pkg/types"
	utilyaml "k8s.io/apimachinery/pkg/util/yaml"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/discovery/cached/memory"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/restmapper"
	"k8s.io/client-go/tools/clientcmd"
)

// antreaManifests will contain the yaml necessary to install the antrea cni into the embedded cluster
//go:embed antrea.yaml
var antreaManifests string

// note: https://github.com/antrea-io/antrea/releases/download/v0.13.5/antrea.yml

func installCNI(kubeconfigPath string) error {
	config, err := clientcmd.BuildConfigFromFlags("", kubeconfigPath)
	if err != nil {
		return errors.Wrap(err, "build config")
	}

	var decUnstructured = yaml.NewDecodingSerializer(unstructured.UnstructuredJSONScheme)

	multidocReader := utilyaml.NewYAMLReader(bufio.NewReader(strings.NewReader(antreaManifests)))
	for {
		buf, err := multidocReader.Read()
		if err != nil {
			if err == io.EOF {
				return nil
			}

			return errors.Wrap(err, "reading multidoc")
		}

		obj := &unstructured.Unstructured{}
		_, gvk, err := decUnstructured.Decode(buf, nil, obj)
		if err != nil {
			return err
		}

		dc, err := discovery.NewDiscoveryClientForConfig(config)
		if err != nil {
			return err
		}
		mapper := restmapper.NewDeferredDiscoveryRESTMapper(memory.NewMemCacheClient(dc))

		dyn, err := dynamic.NewForConfig(config)
		if err != nil {
			return err
		}

		mapping, err := mapper.RESTMapping(gvk.GroupKind(), gvk.Version)
		if err != nil {
			return err
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
			return err
		}

		_, err = dr.Patch(context.Background(), obj.GetName(), types.ApplyPatchType, data, metav1.PatchOptions{
			FieldManager: "kots",
		})

		if err != nil {
			return err
		}

		return nil

	}
}

// find the corresponding GVR (available in *meta.RESTMapping) for gvk
func findGVR(gvk *schema.GroupVersionKind, cfg *rest.Config) (*meta.RESTMapping, error) {

	// DiscoveryClient queries API server about the resources
	dc, err := discovery.NewDiscoveryClientForConfig(cfg)
	if err != nil {
		return nil, err
	}
	mapper := restmapper.NewDeferredDiscoveryRESTMapper(memory.NewMemCacheClient(dc))

	return mapper.RESTMapping(gvk.GroupKind(), gvk.Version)
}
