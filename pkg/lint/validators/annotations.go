package validators

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/pkg/errors"
	"github.com/replicatedhq/kots/pkg/lint/types"
	kotsoperatortypes "github.com/replicatedhq/kots/pkg/operator/types"
	"gopkg.in/yaml.v2"
	"k8s.io/client-go/util/jsonpath"
)

// ValidateAnnotations validates KOTS-specific resource annotations
func ValidateAnnotations(specFiles types.SpecFiles) ([]types.LintExpression, error) {
	lintExpressions := []types.LintExpression{}

	separatedSpecFiles, err := specFiles.Separate()
	if err != nil {
		return nil, errors.Wrap(err, "failed to separate multi docs")
	}

	for _, spec := range separatedSpecFiles {
		var doc map[string]interface{}
		if err := yaml.Unmarshal([]byte(spec.Content), &doc); err != nil {
			return nil, errors.Wrap(err, "failed to unmarshal spec content")
		}

		metadata, ok := doc["metadata"].(map[interface{}]interface{})
		if !ok {
			continue
		}
		annotations, ok := metadata["annotations"].(map[interface{}]interface{})
		if !ok {
			continue
		}
		for k, v := range annotations {
			// convert the key and value to strings
			key, value := fmt.Sprintf("%v", k), fmt.Sprintf("%v", v)
			switch key {
			case kotsoperatortypes.CreationPhaseAnnotation, kotsoperatortypes.DeletionPhaseAnnotation:
				// check that the value is a parsable integer between -9999 and 9999
				parsed, err := strconv.ParseInt(value, 10, 64)
				if err != nil {
					lintExpression := types.LintExpression{
						Rule:    "deployment-phase-annotation",
						Type:    "error",
						Path:    spec.Path,
						Message: fmt.Sprintf("Resource annotation %s should be an integer", key),
					}
					lintExpressions = append(lintExpressions, lintExpression)
				} else if parsed < -9999 || parsed > 9999 {
					lintExpression := types.LintExpression{
						Rule:    "deployment-phase-annotation",
						Type:    "error",
						Path:    spec.Path,
						Message: fmt.Sprintf("Resource annotation %s should be between -9999 and 9999", key),
					}
					lintExpressions = append(lintExpressions, lintExpression)
				}
			case kotsoperatortypes.WaitForPropertiesAnnotation:
				// check that the value is a comma separated list of key=value pairs
				// where the key is a valid jsonpath and the value is not empty
				if value == "" {
					lintExpression := types.LintExpression{
						Rule:    "wait-for-properties-annotation",
						Type:    "error",
						Path:    spec.Path,
						Message: fmt.Sprintf("Resource annotation %s should not be empty", key),
					}
					lintExpressions = append(lintExpressions, lintExpression)
					break
				}

				for _, property := range strings.Split(value, ",") {
					parts := strings.SplitN(property, "=", 2)
					if len(parts) != 2 {
						lintExpression := types.LintExpression{
							Rule:    "wait-for-properties-annotation",
							Type:    "error",
							Path:    spec.Path,
							Message: fmt.Sprintf("Failed to parse %s annotation key=value pair: %s", key, property),
						}
						lintExpressions = append(lintExpressions, lintExpression)
						break
					}
					if parts[0] == "" {
						lintExpression := types.LintExpression{
							Rule:    "wait-for-properties-annotation",
							Type:    "error",
							Path:    spec.Path,
							Message: fmt.Sprintf("Resource annotation %s should not have an empty jsonpath key: %s", key, property),
						}
						lintExpressions = append(lintExpressions, lintExpression)
						break
					}
					if parts[1] == "" {
						lintExpression := types.LintExpression{
							Rule:    "wait-for-properties-annotation",
							Type:    "error",
							Path:    spec.Path,
							Message: fmt.Sprintf("Resource annotation %s should not have an empty value: %s", key, property),
						}
						lintExpressions = append(lintExpressions, lintExpression)
						break
					}
					if _, err := jsonpath.Parse("lint-jsonpath", fmt.Sprintf("{ %s }", parts[0])); err != nil {
						lintExpression := types.LintExpression{
							Rule:    "wait-for-properties-annotation",
							Type:    "error",
							Path:    spec.Path,
							Message: fmt.Sprintf("Resource annotation %s should have a valid jsonpath key: %s", key, property),
						}
						lintExpressions = append(lintExpressions, lintExpression)
						break
					}
				}
			}
		}
	}

	return lintExpressions, nil
}
