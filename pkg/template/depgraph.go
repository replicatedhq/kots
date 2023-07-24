package template

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"regexp"
	"strings"
	"text/template"

	kotsv1beta1 "github.com/replicatedhq/kotskinds/apis/kots/v1beta1"
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

// These functions will be used to figure out dependency order in the event standard rendering fails
// The first argument to each is a string with the dependent config item.
// TLS<blah> functions are deprecated and not included.
const replFuncReExpr = `(?:ConfigOption|ConfigOptionIndex|ConfigData|ConfigOptionFilename|ConfigOptionEquals|ConfigOptionNotEquals) +"[^"]+"`

var re = regexp.MustCompile(replFuncReExpr)

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

	// Also note that if you add a function here, more than likely you will need to add it
	// the regular expression constant in this file, which filters out non-repl functions
	return template.FuncMap{
		"ConfigOption":          addDepFunc,
		"ConfigOptionIndex":     addDepFunc,
		"ConfigOptionData":      addDepFunc,
		"ConfigOptionFilename":  addDepFunc,
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

func (d *depGraph) ResolveMissing(knownConfigs map[string]kotsv1beta1.ConfigItem) {
	for k1, depMap := range d.Dependencies {
		for k2, _ := range depMap {
			_, known := knownConfigs[k2]
			if !known {
				delete(depMap, k2)
			}
		}

		_, known := knownConfigs[k1]
		if !known {
			delete(d.Dependencies, k1)
		}
	}
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

// ParseConfigGroup iterates through config groups and items and runs the template with the
// dependency builder. The builder maps replicated config functions to mocks that analyze dependencies.
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
			// we do not have the full set of template functions available here, and errors from not having those functions are expected
			// when errors are received, we fall back to trying to capture only the replicated functions
			_, err := depBuilder.String(configItem.Default.String())
			if err != nil {
				parseReplFuncs(&depBuilder, configItem.Default.String(), configItem.Name)
			}

			_, err = depBuilder.String(configItem.Value.String())
			if err != nil {
				parseReplFuncs(&depBuilder, configItem.Value.String(), configItem.Name)
			}
		}
	}

	d.resolveCertKeys()
	d.resolveCACertKeys()

	return nil
}

// parseReplFuncs takes in a template string and attempts to filter out the replicated-only functions with a regex.
// It does not return an error to keep the rendering process moving forward.
func parseReplFuncs(depBuilder *Builder, rawTemplate string, itemName string) {
	replFuncs := re.FindAllString(rawTemplate, -1)
	if len(replFuncs) == 0 {
		return
	}

	// separate repl function occurrences so they can be evaluated individually
	for idx := range replFuncs {
		replFuncs[idx] = fmt.Sprintf("repl{{ %s }}", replFuncs[idx])
	}
	cleanedTemplate := strings.Join(replFuncs, " ")

	// We don't exit or return the error here to keep the config rendering.
	_, err := depBuilder.String(cleanedTemplate)
	if err != nil {
		log.Printf("INFO: could not determine config dependencies for item '%s': %s", itemName, err.Error())
	}
}
