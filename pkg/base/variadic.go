package base

import (
	"fmt"
	"reflect"
	"strconv"
	"strings"

	"github.com/google/uuid"
	"github.com/pkg/errors"
	kotsv1beta1 "github.com/replicatedhq/kots/kotskinds/apis/kots/v1beta1"
	"github.com/replicatedhq/kots/pkg/logger"
	upstreamtypes "github.com/replicatedhq/kots/pkg/upstream/types"
	yaml3 "gopkg.in/yaml.v3"
)

// Known issues and TODOs:
// This currently only addresses variadic items.  Variadic groups are not included yet and may require changes to these functions.
// getVariadicGroupsForTemplate should be split into subfunctions to make it easier to read
// The last element in the YamlPath must be an array. We cannot copy whole files yet.

func processVariadicConfig(u *upstreamtypes.UpstreamFile, config *kotsv1beta1.Config, log *logger.CLILogger) ([]upstreamtypes.UpstreamFile, error) {
	templateMetadata, node, err := getUpstreamTemplateData(u.Content)
	if err != nil {
		// if the upstream file can't be unmarshaled as a yaml manifest, this file should be skipped
		log.Info("variadic processing on file %s skipped: %v", u.Path, err.Error())
		return nil, nil
	}

	// fill in templateMetadata data from unmarshaled yaml
	templateMetadata.Name, templateMetadata.Namespace, err = getTemplateMetadata(node)
	if err != nil {
		// if upstream metadata doesn't exist, this file will not match any templates and should be skipped
		log.Info("variadic processing on file %s skipped: %v", u.Path, err.Error())
		return nil, nil
	}

	// collect all variadic config for this specific template
	variadicGroups := getVariadicGroupsForTemplate(config, templateMetadata)

	var generatedFiles []upstreamtypes.UpstreamFile

	for _, vgroup := range variadicGroups {
		for _, vitem := range vgroup.items {
			// check for values that are assigned to this group
			if len(vitem.item.ValuesByGroup[vgroup.group.Name]) == 0 {
				// if no repeat values are provided, allow the default to be rendered as normal
				continue
			}

			// copy the entire yaml file if target yamlpath is empty
			if vitem.yamlPath == "" {
				newFilesContent, err := renderRepeatFilesContent(node, vitem.item.Name, vitem.item.ValuesByGroup[vgroup.group.Name])
				if err != nil {
					return nil, errors.Wrapf(err, "failed to clone file for item %s", vitem.item.Name)
				}

				for _, newFileContent := range newFilesContent {
					newFile := upstreamtypes.UpstreamFile{
						Content: newFileContent,
					}

					shortUUID := strings.Split(uuid.New().String(), "-")[0]
					pathParts := strings.Split(u.Path, ".")

					if len(pathParts) > 1 {
						newFile.Path = fmt.Sprintf("%s-%s.%s", pathParts[0], shortUUID, pathParts[1])
					} else {
						newFile.Path = fmt.Sprintf("%s-%s", pathParts[0], shortUUID)
					}

					generatedFiles = append(generatedFiles, newFile)
				}

			} else {

				yamlStack, err := buildStackFromYaml(vitem.yamlPath, node)
				if err != nil {
					return nil, errors.Wrapf(err, "failed to build yaml stack for item %s", vitem.item.Name)
				}

				yamlStack.renderRepeatNodes(vitem.item.Name, vitem.item.ValuesByGroup[vgroup.group.Name])

				node = buildYamlFromStack(yamlStack)
			}
		}
	}

	marshaled, err := yaml3.Marshal(node)
	if err != nil {
		return nil, errors.Wrap(err, "failed to marshal variadic config")
	}

	u.Content = marshaled

	return generatedFiles, nil
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
func (stack yamlStack) renderRepeatNodes(optionName string, values map[string]interface{}) {
	target := stack[len(stack)-1]

	// build new array with existing values from around the target
	var newArray []interface{}
	newArray = append(newArray, target.Array[:target.Index]...)
	newArray = append(newArray, target.Array[target.Index+1:]...)

	for valueName, value := range values {
		// copy all values into a new map
		newMap := map[string]interface{}{}
		for targetField, targetData := range target.Data {
			// replace the target value
			newMap[targetField] = replaceTemplateValue(targetData, optionName, valueName, value)
		}

		newArray = append(newArray, newMap)
	}

	// insert new array into stack
	target.Array = newArray
	stack[len(stack)-1] = target
}

// replaceTemplateValue searches all nested nodes of a value
// if the provided optionName is found within repl{{ ConfigOption "optionName" }}, the placeholder will be replaced with the repeatable value
func replaceTemplateValue(node interface{}, optionName, valueName string, value interface{}) interface{} {
	switch typedNode := node.(type) {
	case string:
		return generateTargetValue(optionName, valueName, typedNode, value)
	case map[string]interface{}:
		newMap := map[string]interface{}{}
		for subField, subNode := range typedNode {
			newMap[subField] = replaceTemplateValue(subNode, optionName, valueName, value)
		}
		return newMap
	case []interface{}:
		resultSet := []interface{}{}
		for _, subNode := range typedNode {
			results := replaceTemplateValue(subNode, optionName, valueName, value)
			resultSet = append(resultSet, results)
		}
		return resultSet
	}
	return node
}

// isTargetValue determines if a string is the appropriate templated value target
func generateTargetValue(configOptionName, valueName, target string, templateValue interface{}) interface{} {
	if strings.Contains(target, "repl{{") || strings.Contains(target, "{{repl") {
		variable := strings.Split(target, "\"")[1]
		if variable == configOptionName && strings.Contains(target, "ConfigOption ") {
			return templateValue
		} else if variable == configOptionName && strings.Contains(target, "ConfigOptionName") {
			return valueName
		} else if variable == configOptionName {
			return strings.Replace(target, variable, valueName, 1)
		}
	}
	// if no edits are needed, return the original target
	return target
}

func renderRepeatFilesContent(yaml map[string]interface{}, optionName string, values map[string]interface{}) ([][]byte, error) {
	var marshaledFiles [][]byte
	for valueName, value := range values {
		yaml = replaceNewYamlName(yaml, valueName)

		newYaml := replaceTemplateValue(yaml, optionName, valueName, value)

		marshaled, err := yaml3.Marshal(newYaml)
		if err != nil {
			return nil, errors.Wrap(err, "failed to marshal repeat file")
		}

		marshaledFiles = append(marshaledFiles, marshaled)
	}

	return marshaledFiles, nil
}

func replaceNewYamlName(yaml map[string]interface{}, name string) map[string]interface{} {
	metadata := yaml["metadata"].(map[string]interface{})
	metadata["name"] = name

	yaml["metadata"] = metadata
	return yaml
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

// TODO split this into nested functions
func getVariadicGroupsForTemplate(config *kotsv1beta1.Config, templateTarget kotsv1beta1.RepeatTemplate) []variadicGroup {
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

// getTemplateMetadata returns the name and namespace fields from "metadata" at the top level of a template
func getTemplateMetadata(template map[string]interface{}) (string, string, error) {
	metadataInterface, ok := template["metadata"]
	if !ok {
		return "", "", fmt.Errorf("template metadata not found")
	}

	var name, namespace string
	switch metadata := metadataInterface.(type) {
	case map[string]interface{}:
		// ensure the map entry exists
		if metadataName, ok := metadata["name"]; ok {
			// ensure it's a string
			if reflect.TypeOf(metadataName).Name() == "string" {
				name = metadataName.(string)
			}
		}
		if metadataNamespace, ok := metadata["namespace"]; ok {
			if reflect.TypeOf(metadataNamespace).Name() == "string" {
				namespace = metadataNamespace.(string)
			}
		}
	default:
		return "", "", fmt.Errorf("template metadata not of type map[string]interface{}")
	}
	return name, namespace, nil
}
