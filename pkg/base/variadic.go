package base

import (
	"bytes"
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"text/template"

	"github.com/pkg/errors"
	"github.com/replicatedhq/kots/pkg/logger"
	upstreamtypes "github.com/replicatedhq/kots/pkg/upstream/types"
	"github.com/replicatedhq/kots/pkg/util"
	kotsv1beta1 "github.com/replicatedhq/kotskinds/apis/kots/v1beta1"
	yaml3 "gopkg.in/yaml.v3"
)

// Known issues and TODOs:
// This currently only addresses variadic items.  Variadic groups are not included yet and may require changes to these functions.
// getVariadicGroupsForTemplate should be split into subfunctions to make it easier to read.
// The last element in the YamlPath must be an array.

func processVariadicConfig(u *upstreamtypes.UpstreamFile, config *kotsv1beta1.Config, log *logger.CLILogger) ([]byte, error) {
	var finalDocs [][]byte

	multiDoc := util.YAMLBytesToSingleDocs(u.Content)

	for _, doc := range multiDoc {
		templateMetadata, node, err := getUpstreamTemplateData(doc)
		if err != nil {
			// if the upstream file can't be unmarshaled as a yaml manifest, this file should be skipped
			//log.Info("variadic processing on file %s skipped: %v", u.Path, err.Error())
			finalDocs = append(finalDocs, doc)

			continue
		}

		// collect all variadic config for this specific template
		variadicGroups := getVariadicGroupsForTemplate(config, templateMetadata)

		var generatedDocs [][]byte

		for _, vgroup := range variadicGroups {
			for _, vitem := range vgroup.items {
				// check for values that are assigned to this group
				if len(vitem.item.ValuesByGroup[vgroup.group.Name]) == 0 {
					// if no repeat values are provided, allow the default to be rendered as normal
					continue
				}

				// copy the entire yaml file if target yamlpath is empty
				if vitem.yamlPath == "" {
					c, err := renderRepeatFilesContent(node, vitem.item.Name, vitem.item.ValuesByGroup[vgroup.group.Name])
					if err != nil {
						return nil, errors.Wrapf(err, "failed to clone file for item %s", vitem.item.Name)
					}

					generatedDocs = c
				} else {
					yamlStack, err := buildStackFromYaml(vitem.yamlPath, node)
					if err != nil {
						return nil, errors.Wrapf(err, "failed to build yaml stack for item %s", vitem.item.Name)
					}

					err = yamlStack.renderRepeatNodes(vitem.item.Name, vitem.item.ValuesByGroup[vgroup.group.Name])
					if err != nil {
						return nil, errors.Wrapf(err, "failed to render repeat nodes for item %s", vitem.item.Name)
					}

					marshaled, err := yaml3.Marshal(buildYamlFromStack(yamlStack))
					if err != nil {
						return nil, errors.Wrap(err, "failed to marshal variadic config")
					}

					generatedDocs = [][]byte{marshaled}
				}
			}
		}

		if len(generatedDocs) > 0 {
			finalDocs = append(finalDocs, generatedDocs...)
			continue
		}

		finalDocs = append(finalDocs, doc)
	}

	return bytes.Join(finalDocs, []byte("\n---\n")), nil
}

func getUpstreamTemplateData(upstreamContent []byte) (kotsv1beta1.RepeatTemplate, map[string]interface{}, error) {
	var templateHeaders kotsv1beta1.RepeatTemplate

	node := map[string]interface{}{}

	if err := yaml3.Unmarshal(upstreamContent, node); err != nil {
		return templateHeaders, nil, errors.Wrap(err, "failed to unmarshal upstreamFile")
	}
	if apiVersion, ok := node["apiVersion"]; ok {
		switch v := apiVersion.(type) {
		case string:
			templateHeaders.APIVersion = v
		default:
			// upstream file 'apiVersion' is not a string, this cannot be a valid target file and should be skipped
			return templateHeaders, nil, fmt.Errorf("template apiVersion is not a string")
		}
	}
	if kind, ok := node["kind"]; ok {
		switch v := kind.(type) {
		case string:
			templateHeaders.Kind = v
		default:
			// upstream file 'kind' is not a string, this cannot be a valid target file and should be skipped
			return templateHeaders, nil, fmt.Errorf("template kind is not a string")
		}
	}

	metadataInterface, ok := node["metadata"]
	if !ok {
		return templateHeaders, nil, fmt.Errorf("template metadata not found")
	}

	switch metadata := metadataInterface.(type) {
	case map[string]interface{}:
		// ensure the map entry exists
		if metadataName, ok := metadata["name"]; ok {
			// ensure it's a string
			if name, ok := metadataName.(string); ok {
				templateHeaders.Name = name
			}
		}
		if metadataNamespace, ok := metadata["namespace"]; ok {
			if ns, ok := metadataNamespace.(string); ok {
				templateHeaders.Namespace = ns
			}
		}
	default:
		return templateHeaders, nil, fmt.Errorf("template metadata not of type map[string]interface{}")
	}

	return templateHeaders, node, nil
}

type yamlStack []yamlStackItem

type yamlStackItem struct {
	NodeName string
	Type     string
	Index    int
	Data     map[string]interface{}
	Array    []interface{}
}

// buildStackFromYaml deconstructs a nested yaml object into an array of objects
func buildStackFromYaml(yamlPath string, yaml map[string]interface{}) (yamlStack, error) {
	// top node should contain the entire yaml without a NodeName
	stack := yamlStack{
		{
			NodeName: "",
			Type:     "map",
			Data:     yaml,
		},
	}

	currentMap := yaml
	currentArray := []interface{}{}

	pathNodes := strings.Split(yamlPath, ".")
	// traverse the yamlPath to split the structure into a stack of objects
	for _, nextPathNode := range pathNodes {
		nodeShortName, nodeIndex, err := getNodeNameAndIndex(nextPathNode)
		if err != nil {
			return nil, errors.Wrap(err, "failed to collect nodename and index")
		}

		switch nextStep := currentMap[nodeShortName].(type) {
		case []interface{}:
			nodeType := "array"
			// progress both the currentArray and currentMap, 2 steps into the stack
			// we only need the indexed position from the array to select the next node
			currentArray = nextStep
			currentMap = currentArray[*nodeIndex].(map[string]interface{})

			stack = append(stack, yamlStackItem{
				NodeName: nodeShortName,
				Type:     nodeType,
				Index:    *nodeIndex,
				Array:    currentArray,
				Data:     currentMap,
			})

		case map[string]interface{}:
			nodeType := "map"
			// progress only the currentMap, 1 step into the stack
			currentMap = nextStep

			stack = append(stack, yamlStackItem{
				NodeName: nodeShortName,
				Type:     nodeType,
				Data:     currentMap,
			})

		default:
			return nil, fmt.Errorf("failed to process yaml node %s: neither map nor array: %+v", nodeShortName, currentMap[nodeShortName])
		}
	}

	return stack, nil
}

// getNodeNameAndIndex formats the yamlPath node string into a nodeName and index
func getNodeNameAndIndex(name string) (string, *int, error) {
	nodeShortName := strings.Split(name, "[")[0]
	if strings.Contains(name, "[") {
		nodeIndexString := strings.Split(name, "[")[1]
		nodeIndexString = strings.Split(nodeIndexString, "]")[0]
		nodeIndex, err := strconv.Atoi(nodeIndexString)
		if err != nil {
			return "", nil, err
		}
		return nodeShortName, &nodeIndex, nil
	}
	return nodeShortName, nil, nil
}

// buildYamlFromStack reconstructs the yamlStack into a single nested object
func buildYamlFromStack(stack yamlStack) map[string]interface{} {
	var finalNode interface{}
	previousNodeIsDefined := false
	previousNode := yamlStackItem{}

	// reverse the order to rebuild the stack
	bottomUpStack := yamlStack{}
	for i := range stack {
		n := stack[len(stack)-1-i]
		bottomUpStack = append(bottomUpStack, n)
	}

	for _, item := range bottomUpStack {
		if previousNodeIsDefined {
			if item.Type == "map" {
				// insert previous node into the new parent node
				item.Data[previousNode.NodeName] = finalNode
			} else {
				// insert previous node into the new parent node
				// insert map at array index, 2 steps out of the stack
				item.Data[previousNode.NodeName] = finalNode
				item.Array[item.Index] = item.Data
			}
		}
		// prepare finalNode and previoudNode for next loop
		if item.Type == "map" {
			finalNode = item.Data
			previousNode = item
		} else {
			finalNode = item.Array
			previousNode = item
		}
		previousNodeIsDefined = true
	}

	// top level yaml should always be map[string]interface{}
	return finalNode.(map[string]interface{})
}

// renderRepeatNodes duplicates the target item,
// renders each copy with the provided values,
// and merges them in to the last stack array entry
func (stack yamlStack) renderRepeatNodes(optionName string, values map[string]string) error {
	target := stack[len(stack)-1]

	// build new array with existing values from around the target
	var newArray []interface{}
	newArray = append(newArray, target.Array[:target.Index]...)
	newArray = append(newArray, target.Array[target.Index+1:]...)

	for valueName := range values {
		// copy all values into a new map
		newMap := map[string]interface{}{}
		for targetField, targetData := range target.Data {
			var err error
			// replace the target value
			newMap[targetField], err = replaceTemplateValue(targetData, optionName, valueName)
			if err != nil {
				return errors.Wrapf(err, "failed to replace template value on target %s", targetField)
			}
		}

		newArray = append(newArray, newMap)
	}

	// insert new array into stack
	target.Array = newArray
	stack[len(stack)-1] = target

	return nil
}

// replaceTemplateValue searches all nested nodes of a value
// if the provided optionName is found within repl{{ AnyFunction "optionName" }}, "optionName" will be replaced with the repeatable value name
// IE repl{{ ConfigOption "port" | ParseInt }} will become repl{{ ConfigOption "port-8jc8ud" | ParseInt }}, where "port" is the optionName and "port-8jc8ud" is the valueName
// the templating function will be executed with the new variable name after variadic processing is finished
func replaceTemplateValue(node interface{}, optionName, valueName string) (interface{}, error) {
	switch typedNode := node.(type) {
	case string:
		if strings.Contains(typedNode, optionName) {
			resultString, err := parseVariadicTarget(optionName, valueName, typedNode)
			if err != nil {
				return nil, errors.Wrapf(err, "failed to parse %s into %s", optionName, valueName)
			}

			return resultString, nil
		}
		return typedNode, nil
	case map[string]interface{}:
		newMap := map[string]interface{}{}
		for subField, subNode := range typedNode {
			//newMap[subField], err = replaceTemplateValue(subNode, optionName, valueName)
			newValue, err := replaceTemplateValue(subNode, optionName, valueName)
			if err != nil {
				// no need to wrap recursive errors
				return nil, err
			}
			newField, err := replaceTemplateValue(subField, optionName, valueName)
			if err != nil {
				// no need to wrap recursive errors
				return nil, err
			}
			if newField != subField {
				switch typedNewField := newField.(type) {
				case string:
					newMap[typedNewField] = newValue
					delete(newMap, subField)
				default:
					// if it's not a string, we don't want it
				}
			} else {
				newMap[subField] = newValue
			}
		}
		return newMap, nil
	case []interface{}:
		resultSet := []interface{}{}
		for _, subNode := range typedNode {
			results, err := replaceTemplateValue(subNode, optionName, valueName)
			if err != nil {
				// no need to wrap recursive errors
				return nil, err
			}

			resultSet = append(resultSet, results)
		}
		return resultSet, nil
	}
	return node, nil
}

// parseVariadicTarget replaces a variadic template entry with the variadic item name
func parseVariadicTarget(configOptionName, valueName, target string) (string, error) {
	delims := []struct {
		ldelim string
		rdelim string
	}{
		{"[[repl", "]]"},
		{"repl[[", "]]"},
	}

	replace := map[string]string{
		configOptionName: fmt.Sprintf("%s", valueName),
	}

	curText := target
	for _, d := range delims {
		tmpl, err := template.New(configOptionName).Delims(d.ldelim, d.rdelim).Parse(curText)
		if err != nil {
			return "", errors.Wrap(err, "failed to create new template")
		}

		var contents bytes.Buffer
		if err := tmpl.Execute(&contents, replace); err != nil {
			return "", errors.Wrap(err, "failed to execute template")
		}
		curText = contents.String()
	}

	return curText, nil
}

// renderRepeatFilesContent builds repeat files for each repeat value provided
func renderRepeatFilesContent(yaml map[string]interface{}, optionName string, values map[string]string) ([][]byte, error) {
	var marshaledFiles [][]byte
	for valueName := range values {
		var err error
		yaml, err = replaceNewYamlMetadataName(yaml, valueName)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to replace metadata name in repeat file for value %s", valueName)
		}

		newYaml, err := replaceTemplateValue(yaml, optionName, valueName)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to replace template values for value %s", valueName)
		}

		marshaled, err := yaml3.Marshal(newYaml)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to marshal repeat file for value %s", valueName)
		}

		marshaledFiles = append(marshaledFiles, marshaled)
	}

	return marshaledFiles, nil
}

func replaceNewYamlMetadataName(yaml map[string]interface{}, name string) (map[string]interface{}, error) {
	switch metadata := yaml["metadata"].(type) {
	case map[string]interface{}:
		metadata["name"] = name

		yaml["metadata"] = metadata
		return yaml, nil
	default:
		return nil, fmt.Errorf("yaml metadata is not map[string]interface{}")
	}
}

// variadicGroup lists all repeat items under a ConfigGroup
type variadicGroup struct {
	group kotsv1beta1.ConfigGroup
	items []variadicItem
}

// variadicItem ties a ConfigItem to the yamlPath where it should be found
type variadicItem struct {
	item     kotsv1beta1.ConfigItem
	yamlPath string
}

// getVariadicGroupsForTemplate identifies which ConfigItems should be processed for a template
func getVariadicGroupsForTemplate(config *kotsv1beta1.Config, templateTarget kotsv1beta1.RepeatTemplate) []variadicGroup {
	if config == nil {
		return nil
	}

	var variadicGroups []variadicGroup
	for _, group := range config.Spec.Groups {
		var variadicItems []variadicItem
		for _, item := range group.Items {
			for _, template := range item.Templates {
				// set this so the two objects can be directly compared
				templateTarget.YamlPath = template.YamlPath
				if reflect.DeepEqual(template, templateTarget) {
					variadicItems = append(variadicItems, variadicItem{
						item:     item,
						yamlPath: template.YamlPath,
					})
					continue
				}
			}
		}
		if len(variadicItems) > 0 {
			variadicGroups = append(variadicGroups, variadicGroup{
				group: group,
				items: variadicItems,
			})
		}
	}
	return variadicGroups
}
