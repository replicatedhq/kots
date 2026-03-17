package base

import (
	"os"
	"regexp"
	"strings"

	"github.com/replicatedhq/kots/pkg/util"
)

var (
	// regex to extract the manifest name from a helm v3 and v4 manifest
	HelmV4ManifestNameRegex = regexp.MustCompile("^# Source: (.+)")
)

const NamespaceTemplateConst = "repl{{ Namespace}}"

func mergeBaseFiles(baseFiles []BaseFile) []BaseFile {
	merged := []BaseFile{}
	found := map[string]int{}
	for _, baseFile := range baseFiles {
		index, ok := found[baseFile.Path]
		if ok {
			merged[index].Content = append(merged[index].Content, []byte("\n---\n")...)
			merged[index].Content = append(merged[index].Content, baseFile.Content...)
		} else {
			found[baseFile.Path] = len(merged)
			merged = append(merged, baseFile)
		}
	}
	return merged
}

var sep = regexp.MustCompile("(?:^|\\s*\n)---\\s*")

func splitManifests(bigFile string) []string {
	res := []string{}
	bigFileTmp := strings.TrimSpace(bigFile)
	docs := sep.Split(bigFileTmp, -1)
	for _, d := range docs {
		d = strings.TrimSpace(d) + "\n"
		res = append(res, d)
	}
	return res
}

func namespace() string {
	if os.Getenv("DEV_NAMESPACE") != "" {
		return os.Getenv("DEV_NAMESPACE")
	}
	return util.PodNamespace
}
