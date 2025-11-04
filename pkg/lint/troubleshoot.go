package lint

import (
	"context"
	"fmt"
	"path"
	"strings"

	"github.com/replicatedhq/kots/pkg/lint/types"
	troubleshootscheme "github.com/replicatedhq/troubleshoot/pkg/client/troubleshootclientset/scheme"
	"github.com/replicatedhq/troubleshoot/pkg/constants"
	log "github.com/sirupsen/logrus"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

var decoder runtime.Decoder

func init() {
	_ = v1.AddToScheme(troubleshootscheme.Scheme) // for secrets and configmaps
	decoder = troubleshootscheme.Codecs.UniversalDeserializer()
}

// GetEmbeddedTroubleshootSpecs extracts troubleshoot specs from ConfigMaps and Secrets
func GetEmbeddedTroubleshootSpecs(ctx context.Context, specsFiles types.SpecFiles) types.SpecFiles {
	tsSpecs := types.SpecFiles{}

	for _, specFile := range specsFiles {
		troubleshootSpecs := findTroubleshootSpecs(ctx, specFile.Content)
		for _, tsSpec := range troubleshootSpecs {
			tsSpecs = append(tsSpecs, types.SpecFile{
				Name:            path.Join(specFile.Name, tsSpec.Name),
				Path:            specFile.Name,
				Content:         tsSpec.Content,
				AllowDuplicates: tsSpec.AllowDuplicates,
			})
		}
	}

	return tsSpecs
}

// findTroubleshootSpecs extracts troubleshoot specs from ConfigMap and Secret specs
func findTroubleshootSpecs(ctx context.Context, fileData string) types.SpecFiles {
	tsSpecs := types.SpecFiles{}

	srcDocs := strings.Split(fileData, "\n---\n")
	for _, srcDoc := range srcDocs {
		obj, _, err := decoder.Decode([]byte(srcDoc), nil, nil)
		if err != nil {
			log.Debugf("failed to decode raw spec: %s", srcDoc)
			continue
		}

		switch v := obj.(type) {
		case *v1.ConfigMap:
			specs := getSpecFromConfigMap(v, fmt.Sprintf("%s-", v.Name))
			tsSpecs = append(tsSpecs, specs...)
		case *v1.Secret:
			specs := getSpecFromSecret(v, fmt.Sprintf("%s-", v.Name))
			tsSpecs = append(tsSpecs, specs...)
		}
	}

	return tsSpecs
}

func getSpecFromConfigMap(cm *v1.ConfigMap, namePrefix string) types.SpecFiles {
	possibleKeys := []string{
		constants.SupportBundleKey,
		constants.RedactorKey,
		constants.PreflightKey,
		constants.PreflightKey2,
	}

	specs := types.SpecFiles{}
	for _, key := range possibleKeys {
		str, ok := cm.Data[key]
		if ok {
			specs = append(specs, types.SpecFile{
				Name:            namePrefix + key,
				Content:         str,
				AllowDuplicates: true,
			})
		}
	}

	return specs
}

func getSpecFromSecret(secret *v1.Secret, namePrefix string) types.SpecFiles {
	possibleKeys := []string{
		constants.SupportBundleKey,
		constants.RedactorKey,
		constants.PreflightKey,
		constants.PreflightKey2,
	}

	specs := types.SpecFiles{}
	for _, key := range possibleKeys {
		data, ok := secret.Data[key]
		if ok {
			specs = append(specs, types.SpecFile{
				Name:            namePrefix + key,
				Content:         string(data),
				AllowDuplicates: true,
			})
		}

		str, ok := secret.StringData[key]
		if ok {
			specs = append(specs, types.SpecFile{
				Name:            namePrefix + key,
				Content:         str,
				AllowDuplicates: true,
			})
		}
	}

	return specs
}

// GetFilesFromChartReader extracts files from a Helm chart archive
func GetFilesFromChartReader(ctx context.Context, content []byte) (types.SpecFiles, error) {
	// The content is the raw .tgz bytes, we need to parse it as a tar.gz

	// Create a temporary SpecFile to use SpecFilesFromTarGz
	tempFile := types.SpecFile{
		Name:    "temp.tgz",
		Path:    "temp.tgz",
		Content: string(content), // SpecFilesFromTarGz will handle both base64 and raw
	}

	files, err := types.SpecFilesFromTarGz(tempFile)
	if err != nil {
		return nil, err
	}

	return files, nil
}
