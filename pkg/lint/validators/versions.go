package validators

import (
	"fmt"
	"net/http"
	"os"
	"strings"
	"sync"

	"github.com/pkg/errors"
	"github.com/replicatedhq/kots/pkg/lint/types"
	kotsv1beta1 "github.com/replicatedhq/kotskinds/apis/kots/v1beta1"
	"gopkg.in/yaml.v2"
	"k8s.io/client-go/kubernetes/scheme"
)

var (
	kotsVersions map[string]bool
	rwMutex      sync.RWMutex
)

func init() {
	kotsVersions = make(map[string]bool)
}

// ValidateKOTSVersions validates that targetKotsVersion and minKotsVersion exist
func ValidateKOTSVersions(specFiles types.SpecFiles) ([]types.LintExpression, error) {
	lintExpressions := []types.LintExpression{}
	// separate multi docs because the manifest can be a part of a multi doc yaml file
	separatedSpecFiles, err := specFiles.Separate()
	if err != nil {
		return nil, errors.Wrap(err, "failed to separate multi docs")
	}

	lintConfig, err := findLintConfig(separatedSpecFiles)
	if err != nil {
		return nil, errors.Wrap(err, "failed to find lint config")
	}

	tvLintOff, mnLintOff := false, false
	if lintConfig != nil {
		for _, rule := range lintConfig.Spec.Rules {
			if rule.Name == "non-existent-target-kots-version" {
				tvLintOff = rule.Level == "off"
			}
			if rule.Name == "non-existent-min-kots-version" {
				mnLintOff = rule.Level == "off"
			}
		}
	}

	for _, spec := range separatedSpecFiles {
		var tv, mv string
		var tvExists, mvExists bool
		doc := map[string]interface{}{}
		if err := yaml.Unmarshal([]byte(spec.Content), &doc); err != nil {
			return nil, errors.Wrap(err, "failed to unmarshal spec content")
		}
		if doc["apiVersion"] == "kots.io/v1beta1" && doc["kind"] == "Application" {
			if specMap, ok := doc["spec"].(map[interface{}]interface{}); ok {
				tv, tvExists = specMap["targetKotsVersion"].(string)
				mv, mvExists = specMap["minKotsVersion"].(string)
			}
		}

		// if no min nor target kots version exists, continue to next file
		if !mvExists && !tvExists {
			continue
		}

		if tvExists {
			exists, err := checkIfKotsVersionExists(tv)
			if err != nil {
				return nil, errors.Wrap(err, "failed to check if kots version exists")
			}
			if !exists && !tvLintOff {
				targetVersionlintExpression := types.LintExpression{
					Rule:    "non-existent-target-kots-version",
					Type:    "error",
					Path:    spec.Path,
					Message: "Target KOTS version not found",
				}
				lintExpressions = append(lintExpressions, targetVersionlintExpression)
			}
		}

		if mvExists {
			exists, err := checkIfKotsVersionExists(mv)
			if err != nil {
				return nil, errors.Wrap(err, "failed to check if kots version exists")
			}
			if !exists && !mnLintOff {
				minVersionlintExpression := types.LintExpression{
					Rule:    "non-existent-min-kots-version",
					Type:    "error",
					Path:    spec.Path,
					Message: "Minimum KOTS version not found",
				}
				lintExpressions = append(lintExpressions, minVersionlintExpression)
			}
		}
	}

	return lintExpressions, nil
}

func checkIfKotsVersionExists(version string) (bool, error) {
	url := "https://api.github.com/repos/replicatedhq/kots/releases/tags/%s"

	if !strings.HasPrefix(version, "v") {
		version = "v" + version
	}

	rwMutex.RLock()
	verIsCached := kotsVersions[version]
	rwMutex.RUnlock()

	if !verIsCached {
		req, err := http.NewRequest("GET", fmt.Sprintf(url, version), nil)
		if err != nil {
			return false, errors.Wrap(err, "failed to create new request")
		}
		if token := os.Getenv("GITHUB_API_TOKEN"); token != "" {
			var bearer = "Bearer " + token
			req.Header.Set("Authorization", bearer)
		}
		client := &http.Client{}
		resp, err := client.Do(req)
		if err != nil {
			// Network error - don't fail, just skip
			return true, nil
		}
		defer resp.Body.Close()

		if resp.StatusCode == 404 {
			return false, nil
		} else if resp.StatusCode == 200 {
			rwMutex.Lock()
			kotsVersions[version] = true
			rwMutex.Unlock()
		} else {
			return false, errors.New(fmt.Sprintf("received non 200 status code (%d) from GitHub API request", resp.StatusCode))
		}
	}

	return true, nil
}

func findLintConfig(specFiles types.SpecFiles) (*kotsv1beta1.LintConfig, error) {
	var config *kotsv1beta1.LintConfig
	for _, file := range specFiles {
		document := &types.GVKDoc{}
		if err := yaml.Unmarshal([]byte(file.Content), document); err != nil {
			continue
		}
		if document.APIVersion != "kots.io/v1beta1" || document.Kind != "LintConfig" {
			continue
		}
		decode := scheme.Codecs.UniversalDeserializer().Decode
		obj, gvk, err := decode([]byte(file.Content), nil, nil)
		if err != nil {
			return nil, errors.Wrap(err, "failed to decode lint config content")
		}
		if gvk.Group == "kots.io" && gvk.Version == "v1beta1" && gvk.Kind == "LintConfig" {
			config = obj.(*kotsv1beta1.LintConfig)
		}
	}
	return config, nil
}
