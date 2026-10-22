package kotsutil

import (
	"bytes"
	"context"
	"encoding/base64"
	"fmt"
	"strconv"

	"github.com/replicatedhq/kots/pkg/logger"
	"github.com/replicatedhq/kots/pkg/util"
	v2 "github.com/replicatedhq/troubleshoot/pkg/apis/troubleshoot/v1beta2"
	v3 "github.com/replicatedhq/troubleshoot/pkg/apis/troubleshoot/v1beta3"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes"
	"sigs.k8s.io/yaml"
)

const bundleReferencesAnnotation = "kots.io/support-bundle-references"

// PrepareSupportBundleReferences retains beta3 values in the archive without
// reading the cluster. Only the new database fields need deferred resolution;
// all other collectors, analyzers and hooks retain their existing representation.
func PrepareSupportBundleReferences(content []byte) ([]byte, error) {
	return prepareBundleReferences(context.Background(), content, nil, "")
}

func prepareBundleReferences(ctx context.Context, content []byte, client kubernetes.Interface, namespace string) ([]byte, error) {
	if docs := util.ConvertToSingleDocs(content); len(docs) > 1 {
		for i, doc := range docs {
			var err error
			docs[i], err = prepareBundleReferences(ctx, doc, client, namespace)
			if err != nil {
				return nil, err
			}
		}
		return bytes.Join(docs, []byte("\n---\n")), nil
	}
	var doc map[string]interface{}
	if yaml.Unmarshal(content, &doc) != nil {
		return content, nil // Let the existing loader report malformed documents.
	}
	if doc["kind"] == "Secret" || doc["kind"] == "ConfigMap" {
		for _, field := range []string{"data", "stringData"} {
			values, _ := doc[field].(map[string]interface{})
			for _, key := range []string{"support-bundle-spec", "preflight.yaml", "preflight-spec"} {
				value, ok := values[key].(string)
				if !ok {
					continue
				}
				encoded := doc["kind"] == "Secret" && field == "data"
				data := []byte(value)
				if encoded {
					var err error
					data, err = base64.StdEncoding.DecodeString(value)
					if err != nil {
						return nil, err
					}
				}
				prepared, err := prepareBundleReferences(ctx, data, client, namespace)
				if err != nil {
					return nil, err
				}
				if encoded {
					values[key] = base64.StdEncoding.EncodeToString(prepared)
				} else {
					values[key] = string(prepared)
				}
			}
		}
		return yaml.Marshal(doc)
	}
	if doc["apiVersion"] != "troubleshoot.sh/v1beta3" || (doc["kind"] != "SupportBundle" && doc["kind"] != "Preflight") {
		return content, nil
	}
	deferred := false
	spec, _ := doc["spec"].(map[string]interface{})
	collectors, _ := spec["collectors"].([]interface{})
	skipCollector := func(index int, name, field string) {
		message := fmt.Sprintf("Skipped %s collector %d: could not resolve %s. Other collectors continue.", name, index, field)
		logger.Warnf("%s", message)
		bundleName := (&unstructured.Unstructured{Object: doc}).GetName()
		collectors[index] = map[string]interface{}{"data": map[string]interface{}{
			"name": fmt.Sprintf("kots-reference-errors/%s/%s-%d.txt", bundleName, name, index), "data": message,
		}}
	}
collectorsLoop:
	for i, value := range collectors {
		collector, _ := value.(map[string]interface{})
		for _, name := range []string{"postgres", "mysql", "mssql", "redis", "clickhouse"} {
			db, _ := collector[name].(map[string]interface{})
			excluded, _ := strconv.ParseBool(fmt.Sprint(db["exclude"]))
			for _, path := range [][]string{{"uri"}, {"tls", "cacert"}, {"tls", "clientCert"}, {"tls", "clientKey"}} {
				parent := db
				if len(path) == 2 {
					parent, _ = db[path[0]].(map[string]interface{})
				}
				key := path[len(path)-1]
				object, ok := parent[key].(map[string]interface{})
				if !ok {
					continue
				}
				var ref v3.StringOrValueFrom
				if err := runtime.DefaultUnstructuredConverter.FromUnstructured(object, &ref); err != nil {
					if client == nil {
						// Do not log the conversion error: it can contain credential values.
						logger.Warnf("Skipping v1beta3 %s: invalid %s.%s reference", doc["kind"], name, key)
						return UnsupportedTroubleshootWarning(), nil
					}
					skipCollector(i, name, key)
					continue collectorsLoop
				}
				if ref.Value != nil {
					parent[key] = *ref.Value
					continue
				}
				if doc["kind"] == "Preflight" && !excluded {
					return UnsupportedTroubleshootWarning(), nil
				}
				deferred = deferred || client == nil
				parent[key] = ""
				if client != nil && !excluded {
					resolved, err := v3.ResolveStringOrValueFrom(ctx, ref, client, namespace)
					if err != nil {
						skipCollector(i, name, key)
						continue collectorsLoop
					}
					parent[key] = resolved
				}
			}
		}
	}
	if deferred {
		if doc["metadata"] == nil {
			delete(doc, "metadata") // Let the Kubernetes accessor create it.
		}
		object := &unstructured.Unstructured{Object: doc}
		annotations := object.GetAnnotations()
		if annotations == nil {
			annotations = map[string]string{}
		}
		annotations[bundleReferencesAnnotation] = string(content)
		object.SetAnnotations(annotations)
	}
	doc["apiVersion"] = "troubleshoot.sh/v1beta2"
	return yaml.Marshal(doc)
}

// ResolveSupportBundleReferences is called when building a collection, before
// merging specs or rewriting collectors. Never persist resolved values upstream.
func ResolveSupportBundleReferences(ctx context.Context, bundle *v2.SupportBundle, client kubernetes.Interface, namespace string) (*v2.SupportBundle, error) {
	if bundle == nil || bundle.Annotations[bundleReferencesAnnotation] == "" {
		return bundle, nil
	}
	data, err := prepareBundleReferences(ctx, []byte(bundle.Annotations[bundleReferencesAnnotation]), client, namespace)
	if err != nil {
		return nil, err
	}
	return LoadSupportBundleFromContents(data)
}
