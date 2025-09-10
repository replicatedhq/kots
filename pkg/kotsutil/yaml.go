package kotsutil

import (
	"bytes"
	"fmt"
	"strings"

	"github.com/pkg/errors"
	"github.com/replicatedhq/kots/pkg/util"
	"go.yaml.in/yaml/v3"
	k8syaml "sigs.k8s.io/yaml"
)

// FixUpYAML is a general purpose function that will ensure that YAML is compatible with KOTS
// This ensures that lines aren't wrapped at 80 chars which breaks template functions
func FixUpYAML(inputContent []byte) ([]byte, error) {
	docs := util.ConvertToSingleDocs(inputContent)

	var buf bytes.Buffer
	enc := yaml.NewEncoder(&buf)
	enc.SetIndent(2)

	fixedUpDocs := make([][]byte, 0)
	for i, doc := range docs {
		yamlObj := map[string]interface{}{}

		err := yaml.Unmarshal(doc, &yamlObj)
		if err != nil {
			return nil, errors.Wrapf(err, "unmarshal yaml doc %d", i)
		}

		buf.Reset()
		err = enc.Encode(yamlObj)
		if err != nil {
			return nil, errors.Wrapf(err, "encode yaml doc %d", i)
		}

		fixedUpDocs = append(fixedUpDocs, buf.Bytes())
	}

	// MarshalIndent add a line break at the end of each file
	return bytes.Join(fixedUpDocs, []byte("---\n")), nil
}

// RemoveNilFieldsFromYAML removes nil fields from a yaml document.
// This is necessary because kustomize will fail to apply a kustomization if these fields contain nil values: https://github.com/kubernetes-sigs/kustomize/issues/5050
func RemoveNilFieldsFromYAML(input []byte) ([]byte, error) {
	var data map[string]interface{}
	err := k8syaml.Unmarshal([]byte(input), &data)
	if err != nil {
		return nil, errors.Wrap(err, "failed to unmarshal yaml")
	}

	removedItems := removeNilFieldsFromMap(data)
	if !removedItems {
		// no changes were made, return the original input
		return input, nil
	}

	output, err := k8syaml.Marshal(data)
	if err != nil {
		return nil, errors.Wrap(err, "failed to marshal yaml")
	}

	return output, nil
}

func removeNilFieldsFromMap(input map[string]interface{}) bool {
	removedItems := false

	for key, value := range input {
		if value == nil {
			delete(input, key)
			removedItems = true
			continue
		}

		if valueMap, ok := value.(map[string]interface{}); ok {
			removedItems = removeNilFieldsFromMap(valueMap) || removedItems
			continue
		}

		if valueSlice, ok := value.([]interface{}); ok {
			for idx := range valueSlice {
				if itemMap, ok := valueSlice[idx].(map[string]interface{}); ok {
					removedItems = removeNilFieldsFromMap(itemMap) || removedItems
				}
			}
			continue
		}
	}

	return removedItems
}

func MergeYAMLNodes(targetNodes []*yaml.Node, overrideNodes []*yaml.Node) []*yaml.Node {
	// Since inputs are arrays and not maps, we need to:
	// 1. Copy all keys in targetNodes, overriding the ones that match from overrideNodes
	// 2. Add all keys from overrideNodes that don't exist in targetNodes

	if len(overrideNodes) == 0 {
		return targetNodes
	}

	if len(targetNodes) == 0 {
		return overrideNodes
	}

	// Special case where top level node is either a mapping node or an array
	if len(targetNodes) == 1 && len(overrideNodes) == 1 {
		if targetNodes[0].Kind == yaml.MappingNode && overrideNodes[0].Kind == yaml.MappingNode {
			return []*yaml.Node{
				{
					Kind:    yaml.MappingNode,
					Content: MergeYAMLNodes(targetNodes[0].Content, overrideNodes[0].Content),
				},
			}
		}

		if targetNodes[0].Value == overrideNodes[0].Value {
			return overrideNodes
		}

		return append(targetNodes, overrideNodes...)
	}

	// 1. Copy all keys in targetNodes, overriding the ones that match from overrideNodes
	newNodes := make([]*yaml.Node, 0)
	for i := 0; i < len(targetNodes)-1; i += 2 {
		var additionalNode *yaml.Node
		for j := 0; j < len(overrideNodes)-1; j += 2 {
			nodeNameI := targetNodes[i]
			nodeValueI := targetNodes[i+1]

			nodeNameJ := overrideNodes[j]
			nodeValueJ := overrideNodes[j+1]

			if nodeNameI.Value != nodeNameJ.Value {
				continue
			}

			additionalNode = &yaml.Node{
				Kind:        nodeValueJ.Kind,
				Tag:         nodeValueJ.Tag,
				Line:        nodeValueJ.Line,
				Style:       nodeValueJ.Style,
				Anchor:      nodeValueJ.Anchor,
				Value:       nodeValueJ.Value,
				Alias:       nodeValueJ.Alias,
				HeadComment: nodeValueJ.HeadComment,
				LineComment: nodeValueJ.LineComment,
				FootComment: nodeValueJ.FootComment,
				Column:      nodeValueJ.Column,
			}

			if nodeValueI.Kind == yaml.MappingNode && nodeValueJ.Kind == yaml.MappingNode {
				additionalNode.Content = MergeYAMLNodes(nodeValueI.Content, nodeValueJ.Content)
			} else {
				additionalNode.Content = nodeValueJ.Content
			}

			break
		}

		if additionalNode != nil {
			newNodes = append(newNodes, targetNodes[i], additionalNode)
		} else {
			newNodes = append(newNodes, targetNodes[i], targetNodes[i+1])
		}
	}

	// 2. Add all keys from overrideNodes that don't exist in targetNodes
	for j := 0; j < len(overrideNodes)-1; j += 2 {
		isFound := false
		for i := 0; i < len(newNodes)-1; i += 2 {
			nodeNameI := newNodes[i]
			nodeValueI := newNodes[i+1]

			additionalNodeName := overrideNodes[j]
			additionalNodeValue := overrideNodes[j+1]

			if nodeNameI.Value != additionalNodeName.Value {
				continue
			}

			if nodeValueI.Kind == yaml.MappingNode && additionalNodeValue.Kind == yaml.MappingNode {
				nodeValueI.Content = MergeYAMLNodes(nodeValueI.Content, additionalNodeValue.Content)
			}

			isFound = true
			break
		}

		if !isFound {
			newNodes = append(newNodes, overrideNodes[j], overrideNodes[j+1])
		}
	}

	return newNodes
}

func ContentToDocNode(doc *yaml.Node, nodes []*yaml.Node) *yaml.Node {
	if doc == nil {
		return &yaml.Node{
			Kind:    yaml.DocumentNode,
			Content: nodes,
		}
	}
	return &yaml.Node{
		Kind:        doc.Kind,
		Tag:         doc.Tag,
		Line:        doc.Line,
		Style:       doc.Style,
		Anchor:      doc.Anchor,
		Value:       doc.Value,
		Alias:       doc.Alias,
		HeadComment: doc.HeadComment,
		LineComment: doc.LineComment,
		FootComment: doc.FootComment,
		Column:      doc.Column,
		Content:     nodes,
	}
}

func NodeToYAML(node *yaml.Node) ([]byte, error) {
	var renderedContents bytes.Buffer
	yamlEncoder := yaml.NewEncoder(&renderedContents)
	yamlEncoder.SetIndent(2) // this may change indentations of the original values.yaml, but this matches out tests
	err := yamlEncoder.Encode(node)
	if err != nil {
		return nil, errors.Wrap(err, "marshal")
	}

	return renderedContents.Bytes(), nil
}

// Handy functions for printing YAML nodes
func PrintNodes(nodes []*yaml.Node, i int) {
	for _, n := range nodes {
		PrintNode(n, i)
	}
}
func PrintNode(n *yaml.Node, i int) {
	if n == nil {
		return
	}
	indent := strings.Repeat(" ", i*2)
	fmt.Printf("%stag:%v, style:%v, kind:%v, value:%v\n", indent, n.Tag, n.Style, n.Kind, n.Value)
	PrintNodes(n.Content, i+1)
}
