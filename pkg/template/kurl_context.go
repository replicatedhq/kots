package template

import (
	"context"
	"fmt"
	"reflect"
	"sort"
	"strconv"
	"strings"
	"text/template"

	kurlclientset "github.com/replicatedhq/kurl/kurlkinds/client/kurlclientset/typed/cluster/v1beta1"
	kurlv1beta1 "github.com/replicatedhq/kurl/kurlkinds/pkg/apis/cluster/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8sconfig "sigs.k8s.io/controller-runtime/pkg/client/config"
)

func GetKurlValues(installerName, nameSpace string) *kurlv1beta1.Installer {

	cfg, err := k8sconfig.GetConfig()

	if err != nil {
		return nil
	}

	clientset := kurlclientset.NewForConfigOrDie(cfg)

	installers := clientset.Installers(nameSpace)

	retrieved, err := installers.Get(context.TODO(), installerName, metav1.GetOptions{})

	if err != nil {
		return nil
	}

	return retrieved
}

func NewKurlContext(installerName, nameSpace string) *KurlCtx {
	ctx := &KurlCtx{
		KurlValues: make(map[string]interface{}),
	}

	retrieved := GetKurlValues(installerName, nameSpace)

	if retrieved != nil {
		ctx.AddValuesToKurlContext(retrieved)
	}

	return ctx
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
		"KurlOption": ctx.kurlOption,
		"KurlAll":    ctx.kurlAll,
	}
}

func (ctx KurlCtx) kurlBool(yamlPath string) bool {
	if len(ctx.KurlValues) == 0 {
		return false
	}

	result, ok := ctx.KurlValues[yamlPath]
	if !ok {
		fmt.Printf("There is no value found at the yamlPath ''%s'\n", yamlPath)
		return false
	}

	b, ok := result.(bool)
	if !ok {
		fmt.Printf("The yamlPath '%s' corresponds to value '%v' of type '%T'. The KurlBool function supports only boolean values\n", yamlPath, result, result)
		return false
	}

	return b
}

func (ctx KurlCtx) kurlInt(yamlPath string) int {
	if len(ctx.KurlValues) == 0 {
		return 0
	}

	result, ok := ctx.KurlValues[yamlPath]
	if !ok {
		fmt.Printf("There is no value found at the yamlPath '%s'\n", yamlPath)
		return 0
	}

	i, ok := result.(int)
	if !ok {
		fmt.Printf("The yamlPath '%s' corresponds to value '%v' of type '%T'. The KurlInt function supports only integer values\n", yamlPath, result, result)
		return 0
	}

	return i
}

func (ctx KurlCtx) kurlString(yamlPath string) string {
	if len(ctx.KurlValues) == 0 {
		return ""
	}

	result, ok := ctx.KurlValues[yamlPath]
	if !ok {
		fmt.Printf("There is no value found at the yamlPath '%s'\n", yamlPath)
		return ""
	}

	s, ok := result.(string)
	if !ok {
		fmt.Printf("The yamlPath '%s' corresponds to value '%v' of type '%T'. The KurlString function supports only string values\n", yamlPath, result, result)
		return ""
	}

	return s
}

func (ctx KurlCtx) kurlOption(yamlPath string) string {
	if len(ctx.KurlValues) == 0 {
		return ""
	}

	result, ok := ctx.KurlValues[yamlPath]
	if !ok {
		fmt.Printf("There is no value found at the yamlPath '%s'\n", yamlPath)
		return ""
	}

	switch t := interface{}(result).(type) {
	case int:
		return strconv.Itoa(t)
	case string:
		return t
	case bool:
		return strconv.FormatBool(t)
	default:
		fmt.Printf("The yamlPath '%s' corresponds to value '%v' of type '%T'. The KurlOption function supports only string, integer, and boolean values\n", yamlPath, result, result)
		return ""
	}
}

func (ctx KurlCtx) kurlAll() string {
	//debug function to show all supported k:v pairs

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
