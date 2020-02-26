package template

import (
	"text/template"

	"github.com/pkg/errors"
	kurlclientset "github.com/replicatedhq/kurl/kurlkinds/client/kurlclientset/typed/cluster/v1beta1"
	kurlv1beta1 "github.com/replicatedhq/kurl/kurlkinds/pkg/apis/cluster/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8sconfig "sigs.k8s.io/controller-runtime/pkg/client/config"
)

func getKurlValues(installerName, nameSpace string) (*kurlv1beta1.Installer, error) {

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

<<<<<<< HEAD
func NewKurlContext(installerName, nameSpace string) (*KurlCtx, error) {
=======
func NewKurlContext() (*KurlCtx, error) {
>>>>>>> 23f99362268c737414473c3108adcee8c4f64077
	kurlCtx := &KurlCtx{
		KurlValues: make(map[string]interface{}),
	}

<<<<<<< HEAD
	retrieved, err := getKurlValues(installerName, nameSpace)
=======
	retrieved, err := getKurlValues("yaboi", "default")
>>>>>>> 23f99362268c737414473c3108adcee8c4f64077

	if err != nil {
		return nil, errors.Wrap(err, "could not retrieve kurl values")
	}

<<<<<<< HEAD
	kurlCtx.KurlValues["UI"] = retrieved.Spec.Kotsadm.UiBindPort
=======
	kurlCtx.KurlValues["U"] = retrieved.Spec.Kotsadm.UiBindPort
>>>>>>> 23f99362268c737414473c3108adcee8c4f64077

	return kurlCtx, nil
}

type KurlCtx struct {
	KurlValues map[string]interface{}
}

// FuncMap represents the available functions in the ConfigCtx.
func (ctx KurlCtx) FuncMap() template.FuncMap {
	return template.FuncMap{
		"KurlMike": ctx.kurlMike,
	}
}

func (ctx KurlCtx) kurlMike() int {
<<<<<<< HEAD
	result, ok := ctx.KurlValues["UI"]
=======
	result, ok := ctx.KurlValues["U"]
>>>>>>> 23f99362268c737414473c3108adcee8c4f64077

	if !ok {
		return 420
	}

	return result.(int)
}
