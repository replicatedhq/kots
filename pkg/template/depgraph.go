package template

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"
	"text/template"

	kotsv1beta1 "github.com/replicatedhq/kots/kotskinds/apis/kots/v1beta1"
)

type depGraph struct {
	Dependencies        map[string]map[string]struct{}
	CertItems           map[string]map[string]struct{} // items that use the TLSCert template function. map of certnames to configoptions that provide the certname
	KeyItems            map[string]map[string]struct{} // items that use the TLSKey template function. map of TLSKey configoptions to the certnames they use
	CAItems             map[string]map[string]struct{} // items that use the TLSCACert template function. map of canames to configoptions that provide the caname
	CAFromCertItems     map[string]map[string]struct{} // items that use the TLSCertFromCA template function. map of TLSCertFromCA configoptions to the canames they use
	CertFromCACertItems map[string]map[string]struct{} // items that use the TLSCertFromCA template function. map of canames and certnames to configoptions that provide the certname
	CAItemsFromKey      map[string]map[string]struct{} // items that use the TLSKeyFromCA template function. map of TLSKeyFromCA configoptions to the canames they use
	CACertItemsFromKey  map[string]map[string]struct{} // items that use the TLSKeyFromCA template function. map of TLSKeyFromCA configoptions to the canames and certnames they use
}

// these config functions are used to add their dependencies to the depGraph
func (d *depGraph) funcMap(parent string) template.FuncMap {
	addDepFunc := func(dep string, _ ...string) string {
		d.AddDep(parent, dep)
		return dep
	}

	addCertFunc := func(certName string, _ ...string) string {
		d.AddCert(parent, certName)
		return certName
	}

	addKeyFunc := func(certName string, _ ...string) string {
		d.AddKey(parent, certName)
		return certName
	}

	addCAFunc := func(caName string, _ ...string) string {
		d.AddCA(parent, caName)
		return caName
	}

	addCertFromCAFunc := func(caName, certName string, _ ...string) string {
		d.AddCertFromCA(parent, caName, certName)
		return certName
	}

	addKeyFromCAFunc := func(caName, certName string, _ ...string) string {
		d.AddKeyFromCA(parent, caName, certName)
		return certName
	}

	return template.FuncMap{
		"ConfigOption":          addDepFunc,
		"ConfigOptionIndex":     addDepFunc,
		"ConfigOptionData":      addDepFunc,
		"ConfigOptionEquals":    addDepFunc,
		"ConfigOptionNotEquals": addDepFunc,
		"TLSCACert":             addCAFunc,
		"TLSCert":               addCertFunc,
		"TLSCertFromCA":         addCertFromCAFunc,
		"TLSKey":                addKeyFunc,
		"TLSKeyFromCA":          addKeyFromCAFunc,
	}
}

func (d *depGraph) AddNode(source string) {
	if d.Dependencies == nil {
		d.Dependencies = make(map[string]map[string]struct{})
	}

	if _, ok := d.Dependencies[source]; !ok {
		d.Dependencies[source] = make(map[string]struct{})
	}
}

func (d *depGraph) AddDep(source, newDependency string) {
	d.AddNode(source)

	d.Dependencies[source][newDependency] = struct{}{}
}

func (d *depGraph) AddCert(source, certName string) {
	d.CertItems = addDepGraphItem(d.CertItems, certName, source)
}

func (d *depGraph) AddKey(source, certName string) {
	d.KeyItems = addDepGraphItem(d.KeyItems, source, certName)
}

func (d *depGraph) AddCA(source, caName string) {
	d.CAItems = addDepGraphItem(d.CAItems, caName, source)
}

func (d *depGraph) AddCertFromCA(source, caName, certName string) {
	d.CAFromCertItems = addDepGraphItem(d.CAFromCertItems, source, caName)
	d.CertFromCACertItems = addDepGraphItem(d.CertFromCACertItems, caName+certName, source)
}

func (d *depGraph) AddKeyFromCA(source, caName, certName string) {
	d.CAItemsFromKey = addDepGraphItem(d.CAItemsFromKey, source, caName)
	d.CACertItemsFromKey = addDepGraphItem(d.CACertItemsFromKey, source, caName+certName)
}

func addDepGraphItem(m map[string]map[string]struct{}, key, value string) map[string]map[string]struct{} {
	if m == nil {
		m = make(map[string]map[string]struct{})
	}
	if _, ok := m[key]; !ok {
		m[key] = make(map[string]struct{})
	}

	m[key][value] = struct{}{}
	return m
}

func (d *depGraph) resolveCertKeys() {
	for source, certNameMap := range d.KeyItems {
		for certName := range certNameMap {
			for certProvider := range d.CertItems[certName] {
				if certProvider != source {
					d.AddDep(source, certProvider)
				}
			}
		}
	}
}

func (d *depGraph) resolveCACerts() {
	for source, caNameMap := range d.CAFromCertItems {
		for caName := range caNameMap {
			for caProvider := range d.CAItems[caName] {
				if caProvider != source {
					d.AddDep(source, caProvider)
				}
			}
		}
	}
}

func (d *depGraph) resolveCACertKeys() {
	for source, caNameMap := range d.CAItemsFromKey {
		for caName := range caNameMap {
			for caProvider := range d.CAItems[caName] {
				if caProvider != source {
					d.AddDep(source, caProvider)
				}
			}
		}
	}
	for source, caCertNameMap := range d.CACertItemsFromKey {
		for caCertName := range caCertNameMap {
			for certProvider := range d.CertFromCACertItems[caCertName] {
				if certProvider != source {
					d.AddDep(source, certProvider)
				}
			}
		}
	}
}

func (d *depGraph) ResolveDep(resolvedDependency string) {
	for _, depMap := range d.Dependencies {
		delete(depMap, resolvedDependency)
	}
	delete(d.Dependencies, resolvedDependency)
}

func (d *depGraph) GetHeadNodes() ([]string, error) {
	headNodes := []string{}

	for node, deps := range d.Dependencies {
		if len(deps) == 0 {
			headNodes = append(headNodes, node)
		}
	}

	if len(headNodes) == 0 && len(d.Dependencies) != 0 {
		waitList := []string{}
		for k, v := range d.Dependencies {
			depsList := []string{}
			for dep := range v {
				depsList = append(depsList, fmt.Sprintf("%q", dep))
			}
			waitItem := fmt.Sprintf(`%q depends on %s`, k, strings.Join(depsList, `, `))
			waitList = append(waitList, waitItem)
		}
		return headNodes, fmt.Errorf("no config options exist with 0 dependencies - %s", strings.Join(waitList, "; "))
	}

	return headNodes, nil
}

func (d *depGraph) PrintData() string {
	return fmt.Sprintf("deps: %+v", d.Dependencies)
}

// returns a deep copy of the dep graph
func (d *depGraph) Copy() (depGraph, error) {
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	dec := json.NewDecoder(&buf)
	err := enc.Encode(d)
	if err != nil {
		return depGraph{}, err
	}
	depCopy := depGraph{}
	err = dec.Decode(&depCopy)
	if err != nil {
		return depGraph{}, err
	}

	return depCopy, nil

}

func (d *depGraph) ParseConfigGroup(configGroups []kotsv1beta1.ConfigGroup) error {
	for _, configGroup := range configGroups {
		for _, configItem := range configGroup.Items {
			// add this to the dependency graph
			d.AddNode(configItem.Name)

			depBuilder := Builder{
				Ctx:    []Ctx{},
				Functs: d.funcMap(configItem.Name),
			}

			// while builder is normally stateless, the functions it uses within this loop are not
			// errors are also discarded as we do not have the full set of template functions available here, and errors from not having those functions are expected
			_, _ = depBuilder.String(configItem.Default.String())
			_, _ = depBuilder.String(configItem.Value.String())
		}
	}

	d.resolveCertKeys()
	d.resolveCACertKeys()

	return nil
}
