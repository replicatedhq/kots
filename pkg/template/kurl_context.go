package template

import (
	"reflect"
	"sort"
	"strconv"
	"strings"
	"text/template"

	"github.com/pkg/errors"
	kurlclientset "github.com/replicatedhq/kurl/kurlkinds/client/kurlclientset/typed/cluster/v1beta1"
	kurlv1beta1 "github.com/replicatedhq/kurl/kurlkinds/pkg/apis/cluster/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8sconfig "sigs.k8s.io/controller-runtime/pkg/client/config"
)

func GetKurlValues(installerName, nameSpace string) (*kurlv1beta1.Installer, error) {

	cfg, err := k8sconfig.GetConfig()

	if err != nil {
		return nil, errors.Wrap(err, "could not get config")
	}

	clientset := kurlclientset.NewForConfigOrDie(cfg)

	installers := clientset.Installers(nameSpace)

	retrieved, err := installers.Get(installerName, metav1.GetOptions{})

	if err != nil {
		return nil, errors.Wrap(err, "could not retrive installer crd object")
	}

	return retrieved, nil
}

func NewKurlContext(installerName, nameSpace string) (*KurlCtx, error) {
	ctx := &KurlCtx{
		KurlValues: make(map[string]interface{}),
	}

	retrieved, err := GetKurlValues(installerName, nameSpace)

	if err != nil {
		return nil, errors.Wrap(err, "could not retrieve kurl values")
	}

	ctx.AddValuesToKurlContext(retrieved)

	return ctx, nil
}

func (ctx KurlCtx) AddValuesToKurlContext(retrieved *kurlv1beta1.Installer) {
	Spec := reflect.ValueOf(retrieved.Spec)

	for i := 0; i < Spec.NumField(); i++ {
		Category := reflect.ValueOf(Spec.Field(i).Interface())

		TypeOfCategory := Category.Type()

		RawCategoryName := Category.String()
		TrimmedRight := strings.Split(RawCategoryName, ".")[1]
		CategoryName := strings.Split(TrimmedRight, " ")[0]

		for i := 0; i < Category.NumField(); i++ {
			if Category.Field(i).CanInterface() {
				ctx.KurlValues[CategoryName+"."+TypeOfCategory.Field(i).Name] = Category.Field(i).Interface()
			}
		}
	}

}

type KurlCtx struct {
	KurlValues map[string]interface{}
}

func (ctx KurlCtx) FuncMap() template.FuncMap {
	return template.FuncMap{
		"KurlString": ctx.kurlString,
		"KurlInt":    ctx.kurlInt,
		"KurlBool":   ctx.kurlBool,
		"KurlAll":    ctx.kurlAll,
	}
}

func (ctx KurlCtx) kurlBool(yamlPath string) bool {
	result, ok := ctx.KurlValues[yamlPath]
	if !ok {
		//TODO: log that key was not found
		return false
	}

	b, ok := result.(bool)
	if !ok {
		//TODO: log that type was bad
		return false
	}

	return b
}

func (ctx KurlCtx) kurlInt(yamlPath string) int {
	result, ok := ctx.KurlValues[yamlPath]
	if !ok {
		//TODO: log that key was not found
		return 0
	}

	i, ok := result.(int)
	if !ok {
		//TODO: log that type was bad
		return 0
	}

	return i
}

func (ctx KurlCtx) kurlString(yamlPath string) string {
	result, ok := ctx.KurlValues[yamlPath]
	if !ok {
		//TODO: log that key was not found
		return ""
	}

	s, ok := result.(string)
	if !ok {
		//TODO: log that type was bad
		return ""
	}

	return s
}

func (ctx KurlCtx) kurlAll() string {
	keys := make([]string, len(ctx.KurlValues))

	i := 0

	for k, v := range ctx.KurlValues {
		switch t := interface{}(v).(type) {
		case int:
			keys[i] = k + ":" + strconv.Itoa(t)
		case string:
			keys[i] = k + ":" + t
		case bool:
			keys[i] = k + ":" + strconv.FormatBool(t)
		}
		i++
	}

	sort.Strings(keys)

	return strings.Join(keys, " ")
}
