package lint

import (
	"context"
	"encoding/base64"

	"github.com/pkg/errors"
	"github.com/replicatedhq/kots/pkg/lint/types"
	"github.com/replicatedhq/kots/pkg/lint/validators"
	log "github.com/sirupsen/logrus"
)

// LintOptions contains options for linting
type LintOptions struct {
	SkipNetworkChecks bool // Skip checks that require network (e.g., version validation)
	Verbose           bool // Show detailed progress for each validator
}

var (
	kurlLinter *validators.KurlLinter
)

func init() {
	kurlLinter = validators.NewKurlLinter()
}

// InitOPA initializes the OPA linting engine
// This must be called before running LintSpecFiles
func InitOPA() error {
	return validators.InitOPA()
}

// LintSpecFiles performs comprehensive linting on KOTS application spec files
// Returns lint results and a boolean indicating if linting completed all steps
func LintSpecFiles(ctx context.Context, specFiles types.SpecFiles, opts LintOptions) (*types.LintResult, error) {
	result := &types.LintResult{
		LintExpressions: []types.LintExpression{},
		IsComplete:      false,
	}

	// Unnest files (extract children from .tgz archives)
	unnestedFiles := specFiles.Unnest()

	tarGzFiles := types.SpecFiles{}
	yamlFiles := types.SpecFiles{}
	for _, file := range unnestedFiles {
		if file.IsTarGz() {
			tarGzFiles = append(tarGzFiles, file)
		}
		if file.IsYAML() {
			yamlFiles = append(yamlFiles, file)
		}
	}

	// Extract troubleshoot specs from ConfigMaps and Secrets, which may also be in Helm charts
	troubleshootSpecs := GetEmbeddedTroubleshootSpecs(ctx, yamlFiles)
	for _, tsSpec := range troubleshootSpecs {
		yamlFiles = append(yamlFiles, types.SpecFile{
			Name:            tsSpec.Name,
			Path:            tsSpec.Path,
			Content:         tsSpec.Content,
			DocIndex:        len(yamlFiles),
			AllowDuplicates: tsSpec.AllowDuplicates,
		})
	}

	// Also extract troubleshoot specs from .tgz files
	for _, tarGzFile := range tarGzFiles {
		content, err := base64.StdEncoding.DecodeString(tarGzFile.Content)
		if err != nil {
			log.Debugf("failed to base64 decode tarGz content: %v", err)
			continue
		}

		files, err := GetFilesFromChartReader(ctx, content)
		if err != nil {
			log.Debugf("failed to get files from tgz file %s: %v", tarGzFile.Name, err)
			continue
		}
		troubleshootSpecs := GetEmbeddedTroubleshootSpecs(ctx, files)
		for _, tsSpec := range troubleshootSpecs {
			yamlFiles = append(yamlFiles, types.SpecFile{
				Name:            tsSpec.Name,
				Path:            tsSpec.Path,
				Content:         tsSpec.Content,
				DocIndex:        len(yamlFiles),
				AllowDuplicates: tsSpec.AllowDuplicates,
			})
		}
	}

	// Step 1: YAML Syntax Validation
	if opts.Verbose {
		log.Info("Running validator 1/11: YAML Syntax...")
	}
	yamlLintExpressions := validators.ValidateYAML(yamlFiles)
	if opts.Verbose {
		log.Infof("  ✓ YAML Syntax: %d issue(s)", len(yamlLintExpressions))
	}

	// Step 2: OPA Non-Rendered Validation
	// Skip if YAML is invalid (can't parse)
	opaNonRenderedLintExpressions := []types.LintExpression{}
	if !hasErrors(yamlLintExpressions) {
		if opts.Verbose {
			log.Info("Running validator 2/11: OPA Non-Rendered...")
		}
		var err error
		opaNonRenderedLintExpressions, err = validators.ValidateOPANonRendered(yamlFiles)
		if err != nil {
			log.Warnf("OPA Non-Rendered validator failed: %v", err)
		}
		if opts.Verbose {
			log.Infof("  ✓ OPA Non-Rendered: %d issue(s)", len(opaNonRenderedLintExpressions))
		}
	} else if opts.Verbose {
		log.Info("Skipping validator 2/11: OPA Non-Rendered (invalid YAML)")
	}

	// Step 3: Template Rendering Validation
	// Skip if YAML is invalid (can't render)
	renderContentLintExpressions := []types.LintExpression{}
	renderedFiles := yamlFiles // Default to original files
	if !hasErrors(yamlLintExpressions) {
		if opts.Verbose {
			log.Info("Running validator 3/11: Template Rendering...")
		}
		var err error
		renderContentLintExpressions, renderedFiles, err = validators.ValidateRendering(yamlFiles)
		if err != nil {
			log.Warnf("Template Rendering validator failed: %v", err)
		}
		if opts.Verbose {
			log.Infof("  ✓ Template Rendering: %d issue(s)", len(renderContentLintExpressions))
		}
	} else if opts.Verbose {
		log.Info("Skipping validator 3/11: Template Rendering (invalid YAML)")
	}

	// Step 4: Rendered YAML Validity
	if opts.Verbose {
		log.Info("Running validator 4/11: Rendered YAML Validity...")
	}
	renderedYAMLLintExpressions := validators.ValidateRenderedYAML(renderedFiles)
	if opts.Verbose {
		log.Infof("  ✓ Rendered YAML Validity: %d issue(s)", len(renderedYAMLLintExpressions))
	}

	// Step 5: Helm Charts Validation
	if opts.Verbose {
		log.Info("Running validator 5/11: Helm Charts...")
	}
	// Use rendered files since the HelmChart custom resource might not have the right schema before rendering
	helmChartsLintExpressions, err := validators.ValidateHelmCharts(renderedFiles, tarGzFiles)
	if err != nil {
		log.Warnf("Helm Charts validator failed: %v", err)
		helmChartsLintExpressions = []types.LintExpression{}
	}
	if opts.Verbose {
		log.Infof("  ✓ Helm Charts: %d issue(s)", len(helmChartsLintExpressions))
	}

	// Step 6: KOTS Versions Validation (skip if offline mode)
	targetMinLintExpressions := []types.LintExpression{}
	if !opts.SkipNetworkChecks {
		if opts.Verbose {
			log.Info("Running validator 6/11: KOTS Versions...")
		}
		targetMinLintExpressions, err = validators.ValidateKOTSVersions(yamlFiles)
		if err != nil {
			log.Warn(errors.Wrap(err, "failed to lint target and min KOTS versions").Error())
		}
		if opts.Verbose {
			log.Infof("  ✓ KOTS Versions: %d issue(s)", len(targetMinLintExpressions))
		}
	} else if opts.Verbose {
		log.Info("Skipping validator 6/11: KOTS Versions (offline mode)")
	}

	// Step 7: Resource Annotations Validation
	if opts.Verbose {
		log.Info("Running validator 7/11: Resource Annotations...")
	}
	resourceAnnotationsLintExpressions, err := validators.ValidateAnnotations(renderedFiles)
	if err != nil {
		log.Warnf("Resource Annotations validator failed: %v", err)
		resourceAnnotationsLintExpressions = []types.LintExpression{}
	}
	if opts.Verbose {
		log.Infof("  ✓ Resource Annotations: %d issue(s)", len(resourceAnnotationsLintExpressions))
	}

	// Step 8: OPA Rendered Validation
	if opts.Verbose {
		log.Info("Running validator 8/11: OPA Rendered...")
	}
	opaRenderedLintExpressions, err := validators.ValidateOPARendered(renderedFiles, yamlFiles)
	if err != nil {
		log.Warnf("OPA Rendered validator failed: %v", err)
		opaRenderedLintExpressions = []types.LintExpression{}
	}
	if opts.Verbose {
		log.Infof("  ✓ OPA Rendered: %d issue(s)", len(opaRenderedLintExpressions))
	}

	// Step 9: Kubeval Validation
	if opts.Verbose {
		log.Info("Running validator 9/11: Kubeval (Kubernetes Schemas)...")
	}
	kubevalLintExpressions, err := validators.ValidateKubernetes(renderedFiles, yamlFiles)
	if err != nil {
		log.Warnf("Kubeval validator failed: %v", err)
		kubevalLintExpressions = []types.LintExpression{}
	}
	if opts.Verbose {
		log.Infof("  ✓ Kubeval: %d issue(s)", len(kubevalLintExpressions))
	}

	// Step 10: Kurl Installer Validation
	if opts.Verbose {
		log.Info("Running validator 10/11: Kurl Installer...")
	}
	installerLintExpressions, err := kurlLinter.ValidateKurlInstaller(yamlFiles)
	if err != nil {
		log.Warnf("Kurl Installer validator failed: %v", err)
		installerLintExpressions = []types.LintExpression{}
	}
	if opts.Verbose {
		log.Infof("  ✓ Kurl Installer: %d issue(s)", len(installerLintExpressions))
	}

	// Step 11: Embedded Cluster Validation (skip if offline mode)
	embeddedClusterLintExpressions := []types.LintExpression{}
	if !opts.SkipNetworkChecks {
		if opts.Verbose {
			log.Info("Running validator 11/11: Embedded Cluster...")
		}
		embeddedClusterLintExpressions, err = validators.ValidateEmbeddedCluster(yamlFiles)
		if err != nil {
			log.Warnf("Embedded Cluster validator failed: %v", err)
			embeddedClusterLintExpressions = []types.LintExpression{}
		}
		if opts.Verbose {
			log.Infof("  ✓ Embedded Cluster: %d issue(s)", len(embeddedClusterLintExpressions))
		}
	} else if opts.Verbose {
		log.Info("Skipping validator 11/11: Embedded Cluster (offline mode)")
	}

	// Collect all lint expressions
	allLintExpressions := []types.LintExpression{}
	allLintExpressions = append(allLintExpressions, yamlLintExpressions...)
	allLintExpressions = append(allLintExpressions, opaNonRenderedLintExpressions...)
	allLintExpressions = append(allLintExpressions, opaRenderedLintExpressions...)
	allLintExpressions = append(allLintExpressions, renderContentLintExpressions...)
	allLintExpressions = append(allLintExpressions, renderedYAMLLintExpressions...)
	allLintExpressions = append(allLintExpressions, helmChartsLintExpressions...)
	allLintExpressions = append(allLintExpressions, targetMinLintExpressions...)
	allLintExpressions = append(allLintExpressions, resourceAnnotationsLintExpressions...)
	allLintExpressions = append(allLintExpressions, kubevalLintExpressions...)
	allLintExpressions = append(allLintExpressions, installerLintExpressions...)
	allLintExpressions = append(allLintExpressions, embeddedClusterLintExpressions...)

	result.LintExpressions = allLintExpressions
	result.IsComplete = true

	return result, nil
}

// hasErrors returns true if any lint expressions are errors
func hasErrors(lintExpressions []types.LintExpression) bool {
	for _, lintExpression := range lintExpressions {
		if lintExpression.Type == "error" {
			return true
		}
	}
	return false
}
