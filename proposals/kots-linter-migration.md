# PRD: Migrate KOTS Linters from kots-lint Service to KOTS Repo

**Author:** Engineering Team  
**Status:** Draft  
**Created:** 2025-10-30  
**Target Release:** TBD

---

## Executive Summary

This PRD outlines the migration of KOTS linting functionality from the standalone `kots-lint` service (currently hosted at `lint.replicated.com`) into the main `kots` repository. This will enable direct CLI access via `replicated kots lint` and reduce infrastructure dependencies.

---

## Background

### Current State

The `kots-lint` repository is a standalone Go HTTP service that provides 4 distinct linting endpoints:

1. **Main KOTS Linter** (`POST /v1/lint`) - Comprehensive KOTS application validation
2. **Builders Linter** (`POST /v1/builders-lint`) - Helm chart validation for Replicated Builders
3. **Enterprise Linter** (`POST /v1/enterprise-lint`) - Custom OPA policy validation
4. **Troubleshoot Linter** (`POST /v1/troubleshoot-lint`) - Troubleshoot spec validation

**Current Architecture:**
```
Vendor → tar cvf - app/ | curl -XPOST --data-binary @- https://lint.replicated.com/v1/lint
                                    ↓
                          kots-lint service (HTTP API)
                                    ↓
                            Returns lint results
```

### Problems with Current Approach

1. **Network dependency** - Requires internet connectivity and service availability
2. **Latency** - HTTP round-trip adds overhead
3. **Deployment complexity** - Separate service to maintain and deploy
4. **Limited offline usage** - Can't lint in airgapped environments
5. **CLI gap** - No direct `replicated` CLI command for linting

---

## Goals

### Primary Goals

1. - [ ] Migrate **Main KOTS Linter** logic into `kots` repository
2. - [ ] Create `replicated kots lint` CLI command
3. - [ ] Maintain 100% feature parity with current linter
4. - [ ] Enable offline linting (no network required)
5. - [ ] Preserve all existing validation rules and behaviors

### Secondary Goals

1. - [ ] Improve linting performance (eliminate HTTP overhead)
2. - [ ] Better error messages with file paths and line numbers
3. - [ ] Support progressive linting (don't fail-fast on warnings)
4. - [ ] Add structured output formats (JSON, YAML, table)

### Non-Goals

1. ❌ Migrating Builders, Enterprise, or Troubleshoot linters (out of scope for phase 1)
2. ❌ Changing validation rules or adding new checks
3. ❌ Maintaining backward compatibility with HTTP API
4. ❌ Deprecating `lint.replicated.com` immediately (will be done separately)

---

## Success Criteria

- [ ] `replicated kots lint <directory>` command works locally
- [ ] All 11 validation steps from kots-lint are preserved
- [ ] 100% of current test cases pass
- [ ] Documentation updated with migration guide
- [ ] Performance: Linting completes in < 5 seconds for typical apps
- [ ] Zero breaking changes for existing KOTS users

---

## Proposed Solution

### High-Level Architecture

```
Vendor → replicated kots lint ./my-app
              ↓
      KOTS CLI (local)
              ↓
      pkg/lint package (new)
              ↓
      Returns lint results (stdout)
```

### Directory Structure

```
kots/
├── cmd/kots/cli/
│   └── lint.go                    # NEW: CLI command
├── pkg/lint/
│   ├── lint.go                    # NEW: Main linter entrypoint
│   ├── validators/
│   │   ├── yaml.go                # YAML syntax validation
│   │   ├── opa.go                 # OPA policy validation
│   │   ├── render.go              # Template rendering validation
│   │   ├── kubeval.go             # Kubernetes schema validation
│   │   ├── helmchart.go           # Helm chart validation
│   │   ├── version.go             # KOTS version validation
│   │   ├── annotations.go         # KOTS annotations validation
│   │   ├── informers.go           # Status informers validation
│   │   ├── kurl.go                # Kurl installer validation
│   │   └── embeddedcluster.go     # EC installer validation
│   ├── rego/
│   │   ├── nonrendered.rego       # Copied from kots-lint
│   │   ├── rendered.rego          # Copied from kots-lint
│   │   └── builders.rego          # Copied from kots-lint (future)
│   ├── schema/
│   │   ├── kubernetes/            # Copied from kots-lint
│   │   ├── kots/                  # Copied from kots-lint
│   │   ├── troubleshoot/          # Copied from kots-lint
│   │   └── embeddedcluster/       # Copied from kots-lint
│   ├── types.go                   # LintExpression, SpecFile types
│   ├── loader.go                  # Load files from tar/directory
│   └── formatter.go               # Output formatting
└── pkg/kotsutil/                  # Existing - may need extensions
```

---

## Detailed Design

### Phase 1: Core Migration (Main KOTS Linter Only)

#### - [x] 1.1 File Loader ✅

**Purpose:** Load YAML/Helm files from filesystem or tar archive

**Source:** `kots-lint/pkg/domain/spec.go`

**Key Functions:**
```go
// Load files from directory
func LoadFromDirectory(path string) (SpecFiles, error)

// Load files from tar archive  
func LoadFromTar(reader io.Reader) (SpecFiles, error)

// Unnest children files (from .tgz)
func (fs SpecFiles) Unnest() SpecFiles

// Separate multi-doc YAML files
func (fs SpecFiles) Separate() (SpecFiles, error)
```

**Migration Notes:**
- Copy `SpecFile` and `SpecFiles` types
- Preserve `DocIndex` for multi-doc YAML tracking
- Keep `AllowDuplicates` flag for extracted KotsKinds

---

#### - [x] 1.2 Validation Pipeline ✅

**Purpose:** Execute 11 validation steps in order

**Source:** `kots-lint/pkg/kots/lint.go::LintSpecFiles()`

**Pipeline Order** (fail-fast on errors):

1. **YAML Syntax** → `lintIsValidYAML()`
2. **OPA Non-Rendered** → `lintWithOPANonRendered()`
3. **Render Templates** → `lintRenderContent()`
4. **Rendered YAML Validity** → `lintRenderedFilesYAMLValidity()`
5. **Helm Charts** → `lintHelmCharts()`
6. **KOTS Versions** → `lintTargetMinKotsVersions()`
7. **Resource Annotations** → `lintResourceAnnotations()`
8. **OPA Rendered** → `lintWithOPARendered()`
9. **Kubeval** → `lintWithKubeval()`
10. **Kurl Installer** → `kurlLinter.LintKurlInstaller()`
11. **Embedded Cluster** → `ec.LintEmbeddedClusterVersion()`

**Implementation:**
```go
func LintSpecFiles(ctx context.Context, files SpecFiles, opts LintOptions) (*LintResult, error) {
    result := &LintResult{}
    
    // Step 1: YAML Syntax
    if errs := validators.ValidateYAML(files); hasErrors(errs) {
        result.Errors = errs
        return result, nil  // Fail-fast
    }
    
    // Step 2: OPA Non-Rendered
    if errs := validators.ValidateOPANonRendered(files); hasErrors(errs) {
        result.Errors = errs
        return result, nil
    }
    
    // Step 3-11: Continue pipeline...
    
    result.IsComplete = true
    return result, nil
}
```

---

#### - [x] 1.3 Validator Implementations ✅

##### - [x] 1.3.1 YAML Syntax Validator ✅

**Source:** `kots-lint/pkg/kots/lint.go::lintIsValidYAML()`

```go
package validators

func ValidateYAML(files SpecFiles) []LintExpression {
    var expressions []LintExpression
    
    for _, file := range files {
        if !file.IsYAML() {
            continue
        }
        
        decoder := yaml.NewDecoder(bytes.NewReader([]byte(file.Content)))
        decoder.SetStrict(true)
        
        for {
            var doc interface{}
            err := decoder.Decode(&doc)
            
            if err == io.EOF {
                break
            }
            
            if err != nil {
                expressions = append(expressions, LintExpression{
                    Rule:    "invalid-yaml",
                    Type:    "error",
                    Path:    file.Path,
                    Message: err.Error(),
                    // Extract line number from error
                })
                break
            }
        }
    }
    
    return expressions
}
```

**Dependencies:**
- `gopkg.in/yaml.v2` (already in kots repo)

---

##### - [x] 1.3.2 OPA Validator ✅

**Source:** `kots-lint/pkg/kots/lint.go::lintWithOPANonRendered()` and `lintWithOPARendered()`

```go
package validators

import (
    _ "embed"
    "github.com/open-policy-agent/opa/rego"
)

//go:embed ../rego/nonrendered.rego
var nonRenderedRegoContent string

//go:embed ../rego/rendered.rego
var renderedRegoContent string

var (
    nonRenderedQuery *rego.PreparedEvalQuery
    renderedQuery    *rego.PreparedEvalQuery
)

func InitOPA() error {
    // Prepare non-rendered query
    q, err := rego.New(
        rego.Query("data.kots.spec.nonrendered.lint"),
        rego.Module("nonrendered.rego", nonRenderedRegoContent),
    ).PrepareForEval(context.Background())
    if err != nil {
        return err
    }
    nonRenderedQuery = &q
    
    // Prepare rendered query
    // ... similar
    
    return nil
}

func ValidateOPANonRendered(files SpecFiles) ([]LintExpression, error) {
    separated, err := files.Separate()
    if err != nil {
        return nil, err
    }
    
    results, err := nonRenderedQuery.Eval(context.Background(), rego.EvalInput(separated))
    if err != nil {
        return nil, err
    }
    
    return opaResultsToLintExpressions(results, files)
}
```

**Dependencies:**
- `github.com/open-policy-agent/opa` (NEW)

**Migration Steps:**
1. Copy `kots-lint/pkg/kots/rego/*.rego` files to `pkg/lint/rego/`
2. Embed rego files using `//go:embed`
3. Initialize OPA queries at package init or CLI startup
4. Copy `opaResultsToLintExpressions()` helper function

---

##### - [x] 1.3.3 Template Renderer Validator ✅

**Source:** `kots-lint/pkg/kots/lint.go::lintRenderContent()`

```go
package validators

func ValidateTemplateRendering(files SpecFiles) ([]LintExpression, SpecFiles, error) {
    var expressions []LintExpression
    
    separated, err := files.Separate()
    if err != nil {
        return nil, nil, err
    }
    
    // Find and validate Config
    config, path, err := separated.FindAndValidateConfig()
    if err != nil {
        expressions = append(expressions, LintExpression{
            Rule:    "config-is-invalid",
            Type:    "error",
            Path:    path,
            Message: err.Error(),
        })
        return expressions, nil, nil
    }
    
    // Build template context
    builder, err := GetTemplateBuilder(config)
    if err != nil {
        return nil, nil, err
    }
    
    // Render each file
    renderedFiles := SpecFiles{}
    for _, file := range separated {
        rendered, err := file.RenderContent(builder)
        if err != nil {
            // Check if it's a RenderTemplateError with line info
            expressions = append(expressions, LintExpression{
                Rule:    "unable-to-render",
                Type:    "error",
                Path:    file.Path,
                Message: err.Error(),
                // Extract line number if available
            })
            continue
        }
        
        file.Content = string(rendered)
        renderedFiles = append(renderedFiles, file)
    }
    
    return expressions, renderedFiles, nil
}
```

**Dependencies:**
- `pkg/template` (already exists in KOTS)
- May need to expose `FindAndValidateConfig()` from kotsutil

---

##### - [x] 1.3.4 Kubeval Validator ✅

**Source:** `kots-lint/pkg/kots/lint.go::lintWithKubeval()`

```go
package validators

import "github.com/instrumenta/kubeval/kubeval"

func ValidateKubernetes(renderedFiles, originalFiles SpecFiles, schemaDir string) ([]LintExpression, error) {
    var expressions []LintExpression
    
    config := kubeval.Config{
        SchemaLocation:    fmt.Sprintf("file://%s", schemaDir),
        Strict:            true,
        KubernetesVersion: "1.33.3",  // Match kots-lint version
    }
    
    for _, renderedFile := range renderedFiles {
        config.FileName = renderedFile.Path
        
        results, err := kubeval.Validate([]byte(renderedFile.Content), &config)
        if err != nil {
            // Handle "schema not found" vs actual errors
            expressions = append(expressions, LintExpression{
                Rule:    "kubeval-error",
                Type:    determineType(err),
                Path:    renderedFile.Path,
                Message: err.Error(),
            })
            continue
        }
        
        // Process validation results
        for _, result := range results {
            for _, validationError := range result.Errors {
                // Map error to line number in original file
                expressions = append(expressions, LintExpression{
                    Rule:    validationError.Type(),
                    Type:    "warn",
                    Path:    renderedFile.Path,
                    Message: validationError.Description(),
                    // Find line number in original file
                })
            }
        }
    }
    
    return expressions, nil
}
```

**Dependencies:**
- `github.com/instrumenta/kubeval` (NEW)

**Migration Steps:**
1. Copy entire `kubernetes_json_schema/` directory to `pkg/lint/schema/`
2. Bundle schemas in binary (embed or copy at build time)
3. Extract to temp directory at runtime or use embedded FS

---

##### - [x] 1.3.5 Helm Chart Validator ✅

**Source:** `kots-lint/pkg/kots/lint.go::lintHelmCharts()` and `pkg/kots/helm.go`

```go
package validators

import (
    "helm.sh/helm/v3/pkg/chart/loader"
    "helm.sh/helm/v3/pkg/chartutil"
    "helm.sh/helm/v3/pkg/engine"
)

func ValidateHelmCharts(renderedFiles, tarGzFiles SpecFiles) ([]LintExpression, error) {
    var expressions []LintExpression
    
    // Find all HelmChart manifests
    helmCharts := findAllHelmCharts(renderedFiles)
    
    // Check each HelmChart has matching .tgz
    for _, hc := range helmCharts {
        found, err := archiveExists(tarGzFiles, hc)
        if err != nil {
            return nil, err
        }
        if !found {
            expressions = append(expressions, LintExpression{
                Rule:    "helm-archive-missing",
                Type:    "error",
                Message: fmt.Sprintf("Missing .tgz for chart %s:%s", hc.Name, hc.Version),
            })
        }
    }
    
    // Check each .tgz has matching HelmChart manifest
    for _, tgz := range tarGzFiles {
        if !tgz.IsTarGz() {
            continue
        }
        
        found, err := helmChartExists(helmCharts, tgz)
        if err != nil {
            return nil, err
        }
        if !found {
            expressions = append(expressions, LintExpression{
                Rule:    "helm-chart-missing",
                Type:    "error",
                Message: fmt.Sprintf("Missing HelmChart manifest for %s", tgz.Path),
            })
        }
    }
    
    return expressions, nil
}

func RenderHelmChart(tgzReader io.Reader) (SpecFiles, error) {
    // Load chart using Helm libraries
    chart, err := loader.LoadArchive(tgzReader)
    if err != nil {
        return nil, err
    }
    
    // Render templates in lint mode
    eng := new(engine.Engine)
    eng.LintMode = true  // Ignore required/fail functions
    
    renderedTemplates, err := eng.Render(chart, values)
    if err != nil {
        return nil, err
    }
    
    // Convert to SpecFiles
    // ...
}
```

**Dependencies:**
- `helm.sh/helm/v3` (already in kots)

---

##### - [x] 1.3.6 Version Validator ✅

**Source:** `kots-lint/pkg/kots/lint.go::lintTargetMinKotsVersions()`

```go
package validators

func ValidateKOTSVersions(files SpecFiles) ([]LintExpression, error) {
    var expressions []LintExpression
    
    separated, err := files.Separate()
    if err != nil {
        return nil, err
    }
    
    // Find LintConfig to check if validation is disabled
    lintConfig := findLintConfig(separated)
    tvDisabled := isRuleDisabled(lintConfig, "non-existent-target-kots-version")
    mvDisabled := isRuleDisabled(lintConfig, "non-existent-min-kots-version")
    
    for _, file := range separated {
        // Parse Application manifest
        if !isKotsApplication(file) {
            continue
        }
        
        targetVersion, minVersion := extractVersions(file)
        
        if targetVersion != "" && !tvDisabled {
            exists, err := kotsVersionExists(targetVersion)
            if err != nil {
                return nil, err
            }
            if !exists {
                expressions = append(expressions, LintExpression{
                    Rule:    "non-existent-target-kots-version",
                    Type:    "error",
                    Path:    file.Path,
                    Message: "Target KOTS version not found",
                })
            }
        }
        
        // Similar for minVersion
    }
    
    return expressions, nil
}

func kotsVersionExists(version string) (bool, error) {
    url := fmt.Sprintf("https://api.github.com/repos/replicatedhq/kots/releases/tags/%s", version)
    
    resp, err := http.Get(url)
    if err != nil {
        return false, err
    }
    defer resp.Body.Close()
    
    return resp.StatusCode == 200, nil
}
```

**Dependencies:**
- `net/http` (stdlib)
- Environment variable `GITHUB_API_TOKEN` for rate limits

**Notes:**
- Cache version checks in memory to avoid repeated API calls
- Make GitHub API calls optional (skip if no network)

---

##### - [x] 1.3.7 Annotations Validator ✅

**Source:** `kots-lint/pkg/kots/lint.go::lintResourceAnnotations()`

```go
package validators

func ValidateAnnotations(files SpecFiles) ([]LintExpression, error) {
    var expressions []LintExpression
    
    separated, err := files.Separate()
    if err != nil {
        return nil, err
    }
    
    for _, file := range separated {
        annotations := extractAnnotations(file)
        
        for key, value := range annotations {
            switch key {
            case "kots.io/creation-phase", "kots.io/deletion-phase":
                // Must be integer between -9999 and 9999
                parsed, err := strconv.ParseInt(value, 10, 64)
                if err != nil {
                    expressions = append(expressions, LintExpression{
                        Rule:    "deployment-phase-annotation",
                        Type:    "error",
                        Path:    file.Path,
                        Message: fmt.Sprintf("%s must be an integer", key),
                    })
                } else if parsed < -9999 || parsed > 9999 {
                    expressions = append(expressions, LintExpression{
                        Rule:    "deployment-phase-annotation",
                        Type:    "error",
                        Path:    file.Path,
                        Message: fmt.Sprintf("%s must be between -9999 and 9999", key),
                    })
                }
                
            case "kots.io/wait-for-properties":
                // Must be valid JSONPath
                if err := validateJSONPath(value); err != nil {
                    expressions = append(expressions, LintExpression{
                        Rule:    "wait-for-properties-annotation",
                        Type:    "error",
                        Path:    file.Path,
                        Message: fmt.Sprintf("Invalid JSONPath: %v", err),
                    })
                }
            }
        }
    }
    
    return expressions, nil
}
```

**Dependencies:**
- `k8s.io/client-go/util/jsonpath` (already in kots)

---

##### - [x] 1.3.8 Status Informers Validator ✅

**Source:** `kots-lint/pkg/kots/rego/rendered.rego`

**Note:** This is implemented in OPA Rego, not Go. The validation happens in `ValidateOPARendered()`.

The rego file checks:
1. Format: `[namespace/]kind/name`
2. Existence: Does the referenced resource exist in the YAML?

No separate Go validator needed - handled by OPA.

---

##### - [x] 1.3.9 Kurl Installer Validator ✅

**Source:** `kots-lint/pkg/kurl/lint.go`

```go
package validators

import kurllint "github.com/replicatedhq/kurlkinds/pkg/lint"

func ValidateKurlInstaller(files SpecFiles) ([]LintExpression, error) {
    var expressions []LintExpression
    
    linter := kurllint.New()
    
    separated, err := files.Separate()
    if err != nil {
        return nil, err
    }
    
    for _, file := range separated {
        if !file.IsYAML() {
            continue
        }
        
        output, err := linter.ValidateMarshaledYAML(context.Background(), file.Content)
        if err != nil {
            if err == kurllint.ErrNotInstaller {
                continue  // Not a Kurl installer, skip
            }
            return nil, err
        }
        
        for _, out := range output {
            expressions = append(expressions, LintExpression{
                Rule:    fmt.Sprintf("kubernetes-installer-%s", out.Type),
                Type:    "error",
                Path:    file.Path,
                Message: out.Message,
            })
        }
    }
    
    return expressions, nil
}
```

**Dependencies:**
- `github.com/replicatedhq/kurlkinds/pkg/lint` (NEW)

---

##### - [x] 1.3.10 Embedded Cluster Validator ✅

**Source:** `kots-lint/pkg/ec/lint.go`

```go
package validators

func ValidateEmbeddedCluster(files SpecFiles) ([]LintExpression, error) {
    var expressions []LintExpression
    
    separated, err := files.Separate()
    if err != nil {
        return nil, err
    }
    
    for _, file := range separated {
        if !isEmbeddedClusterConfig(file) {
            continue
        }
        
        version := extractECVersion(file)
        
        if version == "" {
            expressions = append(expressions, LintExpression{
                Rule:    "ec-version-required",
                Type:    "error",
                Path:    file.Path,
                Message: "Embedded Cluster version is required",
            })
            continue
        }
        
        // Check if version exists on GitHub
        exists, preRelease, err := ecVersionExists(version)
        if err != nil {
            return nil, err
        }
        
        if !exists {
            expressions = append(expressions, LintExpression{
                Rule:    "non-existent-ec-version",
                Type:    "error",
                Path:    file.Path,
                Message: "Embedded Cluster version not found",
            })
        } else if preRelease {
            expressions = append(expressions, LintExpression{
                Rule:    "non-existent-ec-version",
                Type:    "error",
                Path:    file.Path,
                Message: "Embedded Cluster version is a pre-release",
            })
        }
    }
    
    return expressions, nil
}

func ecVersionExists(version string) (exists bool, preRelease bool, err error) {
    url := fmt.Sprintf("https://api.github.com/repos/replicatedhq/embedded-cluster/releases/tags/%s", version)
    
    resp, err := http.Get(url)
    if err != nil {
        return false, false, err
    }
    defer resp.Body.Close()
    
    if resp.StatusCode == 404 {
        return false, false, nil
    }
    
    var release struct {
        PreRelease bool `json:"prerelease"`
    }
    json.NewDecoder(resp.Body).Decode(&release)
    
    return true, release.PreRelease, nil
}
```

**Dependencies:**
- `net/http` (stdlib)

---

#### - [x] 1.4 CLI Command (Test/Development) ✅

**Location:** `cmd/kots/cli/lint.go` (for testing in kots repo)

**Note:** This is a development/testing CLI. Final production CLI will be in the `replicated` CLI repository.

```go
package cli

import (
    "github.com/spf13/cobra"
    "github.com/replicatedhq/kots/pkg/lint"
)

func LintCmd() *cobra.Command {
    var (
        outputFormat string  // table, json, yaml
        failOnWarn   bool
        skipNetworkChecks bool
    )
    
    cmd := &cobra.Command{
        Use:   "lint [path]",
        Short: "Lint a KOTS application",
        Long: `Lint validates a KOTS application for errors and warnings.
        
The path can be:
  - A directory containing YAML files
  - A tar archive
  - If omitted, uses current directory`,
        Args: cobra.MaximumNArgs(1),
        RunE: func(cmd *cobra.Command, args []string) error {
            path := "."
            if len(args) > 0 {
                path = args[0]
            }
            
            // Load files
            files, err := lint.LoadFiles(path)
            if err != nil {
                return err
            }
            
            // Run linter
            result, err := lint.LintSpecFiles(cmd.Context(), files, lint.LintOptions{
                SkipNetworkChecks: skipNetworkChecks,
            })
            if err != nil {
                return err
            }
            
            // Format output
            if err := lint.PrintResult(result, outputFormat); err != nil {
                return err
            }
            
            // Exit code
            if result.HasErrors() {
                return errors.New("linting failed with errors")
            }
            if failOnWarn && result.HasWarnings() {
                return errors.New("linting failed with warnings")
            }
            
            return nil
        },
    }
    
    cmd.Flags().StringVarP(&outputFormat, "output", "o", "table", "Output format (table, json, yaml)")
    cmd.Flags().BoolVar(&failOnWarn, "fail-on-warn", false, "Exit with error on warnings")
    cmd.Flags().BoolVar(&skipNetworkChecks, "offline", false, "Skip checks that require network")
    
    return cmd
}
```

**Integration:**
Add to `cmd/kots/main.go`:
```go
rootCmd.AddCommand(cli.LintCmd())
```

---

#### - [x] 1.5 Supporting Utilities ✅

##### - [x] 1.5.1 Line Number Extraction ✅

**Source:** `kots-lint/pkg/util/text.go`

Key functions to copy:
```go
// GetLineNumberFromYamlPath - Find line number for a YAML path (e.g., "spec.replicas")
func GetLineNumberFromYamlPath(content, yamlPath string, docIndex int) (int, error)

// GetLineNumberFromMatch - Find line number for a text match
func GetLineNumberFromMatch(content, match string, docIndex int) (int, error)

// TryGetLineNumberFromValue - Extract line number from error message
func TryGetLineNumberFromValue(value string) (int, error)

// GetLineNumberForDoc - Get starting line for a specific document in multi-doc YAML
func GetLineNumberForDoc(content string, docIndex int) (int, error)
```

These are critical for mapping validation errors to specific line numbers in the original files.

---

##### - [x] 1.5.2 Troubleshoot Spec Extraction ✅

**Source:** `kots-lint/pkg/kots/troubleshoot.go`

```go
// GetEmbeddedTroubleshootSpecs - Extract Preflight/SupportBundle specs from ConfigMaps/Secrets
func GetEmbeddedTroubleshootSpecs(ctx context.Context, files SpecFiles) SpecFiles
```

This extracts troubleshoot specs that are embedded in ConfigMaps/Secrets (common pattern for Helm charts).

---

#### - [x] 1.6 Output Formatting ✅

**Location:** `pkg/lint/formatter.go`

```go
package lint

type OutputFormatter interface {
    Format(result *LintResult) (string, error)
}

// Table format (default)
type TableFormatter struct{}

func (f *TableFormatter) Format(result *LintResult) (string, error) {
    // Use tabwriter or similar
    // Output:
    //   RULE                    TYPE    PATH              LINE  MESSAGE
    //   invalid-yaml            error   deployment.yaml   42    unexpected end of file
    //   helm-archive-missing    error                           Missing .tgz for nginx:1.2.3
}

// JSON format
type JSONFormatter struct{}

func (f *JSONFormatter) Format(result *LintResult) (string, error) {
    return json.MarshalIndent(result, "", "  ")
}

// YAML format
type YAMLFormatter struct{}

func (f *YAMLFormatter) Format(result *LintResult) (string, error) {
    return yaml.Marshal(result)
}
```

---

## Migration Plan

### Phase 1: Foundation (Week 1-2) ✅ COMPLETE

**Tasks:**
1. - [x] Create `pkg/lint` directory structure
2. - [x] Copy and adapt type definitions (`SpecFile`, `LintExpression`, etc.)
3. - [x] Implement file loader (`LoadFromDirectory`, `LoadFromTar`)
4. - [x] Copy utility functions (line number extraction, YAML helpers)
5. - [x] Copy JSON schemas to `pkg/lint/schema/` (78MB)
6. - [x] Copy Rego files to `pkg/lint/rego/` (4 files)
7. - [x] Add dependencies to `go.mod`:
   - `github.com/open-policy-agent/opa` ✅ (already present v1.9.0)
   - `github.com/instrumenta/kubeval` ✅ (added v0.0.0-20190918223246-8d013ec9fc56)
   - `github.com/replicatedhq/kurlkinds/pkg/lint` ✅ (already present v1.5.0)
   - `github.com/mitchellh/mapstructure` ✅ (added v1.5.0)
8. - [x] Write unit tests for loader and utilities (22 tests, all passing)

**Deliverable:** ✅ Foundation code that can load files and extract line numbers

**Completion Date:** 2025-10-30

---

### Phase 2: Validators (Week 3-4) ✅ COMPLETE

**Tasks:**
1. - [x] Implement YAML syntax validator (`yaml.go` - 122 lines)
2. - [x] Implement OPA validators (non-rendered + rendered) (`opa.go` - 168 lines)
3. - [x] Implement template rendering validator (`render.go` - 308 lines)
4. - [x] Implement Kubeval validator (`kubeval.go` - 104 lines)
5. - [x] Implement Helm chart validator (`helm.go` - 173 lines)
6. - [x] Implement version validator (`versions.go` - 165 lines)
7. - [x] Implement annotations validator (`annotations.go` - 132 lines)
8. - [x] Implement Kurl installer validator (`kurl.go` - 56 lines)
9. - [x] Implement Embedded Cluster validator (`embeddedcluster.go` - 144 lines)
10. - [ ] Write comprehensive unit tests for each validator

**Deliverable:** ✅ All 9 validators implemented (1,372 lines) - Tests pending Phase 5

**Completion Date:** 2025-10-30

---

### Phase 3: Pipeline Integration (Week 5) ✅ COMPLETE

**Tasks:**
1. - [x] Implement main `LintSpecFiles()` pipeline (`lint.go` - 220 lines)
2. - [x] Integrate all validators in correct order (11 steps with proper fail-fast)
3. - [x] Implement fail-fast logic (steps 1-8 fail-fast, 9-11 collect warnings)
4. - [x] Add context cancellation support (context passed through pipeline)
5. - [ ] Write integration tests with real KOTS apps (pending Phase 5)
6. - [ ] Test against `kots-lint/example/` files (pending Phase 5)
7. - [ ] Ensure 100% output parity with kots-lint service (pending Phase 5)

**Deliverable:** ✅ Complete linting pipeline with all validators integrated

**Completion Date:** 2025-10-30

---

### Phase 4: CLI Command (Week 6) ✅ COMPLETE

**Note:** The CLI in this repo (`cmd/kots/cli/lint.go`) is for **testing and development only**. The final production CLI will be integrated into the `replicated` CLI repository, which will import `github.com/replicatedhq/kots/pkg/lint`.

**Tasks:**
1. - [x] Implement test CLI command in `cmd/kots/cli/lint.go` (177 lines)
2. - [x] Add output formatters (table, JSON, YAML)
3. - [x] Add CLI flags (`--output`, `--fail-on-warn`, `--offline`)
4. - [x] Implement progress indicators (using logger.ActionWithSpinner)
5. - [x] Add error handling and user-friendly messages
6. - [ ] Write CLI tests (pending Phase 5)
7. - [x] Document integration plan for `replicated` CLI repo (added to PRD)

**Deliverable:** ✅ Working test CLI command for development/validation

**Completion Date:** 2025-10-30

**Future:** Integrate into `replicated` CLI as `replicated kots lint` (separate task)

---

### Phase 5: Testing & Documentation (Week 7)

**Tasks:**
1. - [ ] Port all test cases from kots-lint
2. - [ ] Add edge case tests
3. - [ ] Performance testing and optimization
4. - [ ] Write user documentation
5. - [ ] Write migration guide for vendors
6. - [ ] Update `replicated` CLI README
7. - [ ] Create example applications for testing

**Deliverable:** Production-ready linter with complete documentation

---

### Phase 6: Release (Week 8)

**Tasks:**
1. - [ ] Beta release to select vendors
2. - [ ] Gather feedback and fix issues
3. - [ ] Final testing on real-world apps
4. - [ ] Update replicated.com documentation
5. - [ ] Announce to vendor community
6. - [ ] Plan deprecation timeline for lint.replicated.com

**Deliverable:** GA release of `replicated kots lint`

---

## Testing Strategy

### Unit Tests

Each validator must have comprehensive unit tests:

```go
func TestValidateYAML(t *testing.T) {
    tests := []struct{
        name        string
        input       SpecFiles
        wantErrors  int
        wantRule    string
    }{
        {
            name: "valid yaml",
            input: SpecFiles{{Content: "apiVersion: v1\nkind: Pod"}},
            wantErrors: 0,
        },
        {
            name: "invalid yaml - syntax error",
            input: SpecFiles{{Content: "apiVersion: v1\nkind: Pod\n  bad indent"}},
            wantErrors: 1,
            wantRule: "invalid-yaml",
        },
        // ... more cases
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            result := ValidateYAML(tt.input)
            if len(result) != tt.wantErrors {
                t.Errorf("got %d errors, want %d", len(result), tt.wantErrors)
            }
        })
    }
}
```

---

### Integration Tests

Test complete linting pipeline with real applications:

```go
func TestLintRealApplication(t *testing.T) {
    tests := []struct{
        name           string
        appPath        string
        wantErrors     int
        wantWarnings   int
        wantComplete   bool
    }{
        {
            name:         "example app",
            appPath:      "testdata/example-app",
            wantErrors:   0,
            wantWarnings: 0,
            wantComplete: true,
        },
        {
            name:         "app with helm chart",
            appPath:      "testdata/helm-app",
            wantErrors:   0,
            wantComplete: true,
        },
        {
            name:         "app with errors",
            appPath:      "testdata/invalid-app",
            wantErrors:   3,
            wantComplete: false,
        },
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            files, _ := LoadFromDirectory(tt.appPath)
            result, err := LintSpecFiles(context.Background(), files, LintOptions{})
            require.NoError(t, err)
            
            assert.Equal(t, tt.wantComplete, result.IsComplete)
            assert.Equal(t, tt.wantErrors, result.ErrorCount())
            assert.Equal(t, tt.wantWarnings, result.WarningCount())
        })
    }
}
```

---

### Compatibility Tests

Ensure output matches kots-lint service:

```go
func TestOutputParity(t *testing.T) {
    // For each test case, lint with both old and new linter
    // Compare results
    
    appDir := "testdata/sample-app"
    
    // New linter
    files, _ := LoadFromDirectory(appDir)
    newResult, _ := LintSpecFiles(context.Background(), files, LintOptions{})
    
    // Old linter (call HTTP API)
    oldResult := callKotsLintService(appDir)
    
    // Compare
    assert.Equal(t, oldResult.ErrorCount(), newResult.ErrorCount())
    assert.Equal(t, oldResult.WarningCount(), newResult.WarningCount())
    
    // Compare specific rules
    for _, oldExpr := range oldResult.Expressions {
        found := false
        for _, newExpr := range newResult.Expressions {
            if oldExpr.Rule == newExpr.Rule && oldExpr.Path == newExpr.Path {
                found = true
                break
            }
        }
        assert.True(t, found, "Missing lint expression: %s", oldExpr.Rule)
    }
}
```

---

### Performance Tests

Ensure linting completes quickly:

```go
func BenchmarkLintLargeApp(b *testing.B) {
    files, _ := LoadFromDirectory("testdata/large-app")
    
    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        LintSpecFiles(context.Background(), files, LintOptions{})
    }
}

func TestLintPerformance(t *testing.T) {
    files, _ := LoadFromDirectory("testdata/typical-app")
    
    start := time.Now()
    _, err := LintSpecFiles(context.Background(), files, LintOptions{})
    duration := time.Since(start)
    
    require.NoError(t, err)
    assert.Less(t, duration, 5*time.Second, "Linting took too long")
}
```

---

## Dependencies

### New Dependencies to Add

```
go.mod additions:

require (
    github.com/open-policy-agent/opa v0.60.0
    github.com/instrumenta/kubeval v0.16.1
    github.com/replicatedhq/kurlkinds latest
    github.com/mitchellh/mapstructure v1.5.0  // For OPA result parsing
)
```

### Existing Dependencies to Use

- `helm.sh/helm/v3` - Already in kots
- `gopkg.in/yaml.v2` - Already in kots
- `k8s.io/client-go` - Already in kots
- `github.com/replicatedhq/kotskinds` - Already in kots
- `github.com/replicatedhq/troubleshoot` - Already in kots

---

## Schema Management

### Schema Files

The `kubernetes_json_schema/schema/` directory contains 1,500+ JSON schema files (280 MB). 

**Options:**

#### Option 1: Embed Schemas in Binary
```go
//go:embed schema/**/*.json
var schemaFS embed.FS
```

**Pros:**
- Single binary, no external files needed
- Works offline

**Cons:**
- Large binary size (~280 MB increase)

#### Option 2: Download Schemas at Runtime
```go
func EnsureSchemas() (string, error) {
    cacheDir := filepath.Join(os.UserCacheDir(), "kots", "schemas")
    if !exists(cacheDir) {
        // Download from GitHub releases
        downloadSchemas(cacheDir)
    }
    return cacheDir, nil
}
```

**Pros:**
- Smaller binary
- Schemas can be updated independently

**Cons:**
- Requires network on first run
- More complex

#### Option 3: Bundle Schemas Separately (Recommended)

```bash
# Release artifacts:
replicated-linux-amd64          # ~50 MB
replicated-schemas.tar.gz       # ~30 MB (compressed)
```

First run:
```bash
$ replicated kots lint ./app
Downloading schemas... (one-time setup)
Schemas cached to ~/.cache/kots/schemas
```

**Pros:**
- Reasonable binary size
- Optional for offline scenarios
- Can be pre-installed

**Cons:**
- Extra artifact to manage

**Recommendation:** Use Option 3 for MVP, consider Option 1 for embedded cluster scenarios

---

## Error Handling

### Network Failures

Some validators require network access (GitHub API checks). Handle gracefully:

```go
func ValidateKOTSVersions(files SpecFiles, opts LintOptions) ([]LintExpression, error) {
    if opts.SkipNetworkChecks {
        // Skip GitHub API calls
        return []LintExpression{}, nil
    }
    
    exists, err := kotsVersionExists(version)
    if err != nil {
        if isNetworkError(err) {
            // Warn but don't fail
            return []LintExpression{{
                Rule: "version-check-skipped",
                Type: "info",
                Message: "Could not verify KOTS version (network unavailable)",
            }}, nil
        }
        return nil, err
    }
    // ...
}
```

---

### Partial Failures

If one validator fails catastrophically, others should still run:

```go
func LintSpecFiles(ctx context.Context, files SpecFiles, opts LintOptions) (*LintResult, error) {
    result := &LintResult{}
    
    // Even if validator returns error, collect what we can
    if exprs, err := ValidateYAML(files); err != nil {
        log.Warn("YAML validator error: %v", err)
        // Add generic error but continue
        result.AddError(LintExpression{
            Rule: "linter-error",
            Type: "error",
            Message: fmt.Sprintf("Validator failed: %v", err),
        })
    } else {
        result.Add(exprs...)
    }
    
    // Continue with other validators...
}
```

---

## Backwards Compatibility

### For Vendors

**No breaking changes:**
- Existing `lint.replicated.com` API remains unchanged
- Vendors can continue using `curl -XPOST ...` workflow
- New CLI is additive, not replacing anything

### For KOTS Runtime

This migration only affects **lint-time validation**, not runtime behavior:
- No changes to how KOTS deploys applications
- No changes to KOTS API server
- No changes to kotsadm

---

## Production Integration (Post Phase 6)

After completing Phases 1-6 in the `kots` repository, the linting functionality will be integrated into the production `replicated` CLI:

### Integration Plan

**Repository:** `github.com/replicatedhq/replicated` (separate repo)

**Integration Steps:**
1. Import `github.com/replicatedhq/kots/pkg/lint` package
2. Create `replicated kots lint` command that calls `lint.LintSpecFiles()`
3. Add to `replicated` CLI command tree
4. Test with real vendor workflows
5. Deprecate test CLI from `kots` repo (optional - can keep for dev purposes)

**Command Path:**
```
replicated/
├── cmd/
│   └── kots/
│       └── lint.go       # Production CLI
```

**Usage:**
```bash
replicated kots lint ./my-app
```

**Benefits of this approach:**
- ✅ `kots` repo contains the **core linting logic** (pkg/lint)
- ✅ `replicated` CLI provides the **user interface**
- ✅ Test CLI in `kots` repo for development/validation
- ✅ Clean separation of concerns

---

## Future Enhancements (Out of Scope)

These are **not** part of the initial migration but can be added later:

1. **Auto-fix suggestions** - Suggest fixes for common errors
2. **Custom rules** - Allow vendors to add their own lint rules
3. **IDE integration** - VSCode extension for real-time linting
4. **CI/CD integration** - GitHub Action for automated linting
5. **Incremental linting** - Only lint changed files
6. **Watch mode** - Continuously lint on file changes
7. **Migrate other linters** - Builders, Enterprise, Troubleshoot linters

---

## Risks & Mitigations

| Risk | Impact | Mitigation |
|------|--------|------------|
| **Missing edge cases** | Linter produces different results than kots-lint | Comprehensive compatibility tests comparing outputs |
| **Performance regression** | Linting takes too long | Benchmarking, profiling, optimize hot paths |
| **Large binary size** | 280 MB schema files in binary | Bundle schemas separately, download on first run |
| **OPA dependency conflicts** | OPA has many transitive deps | Use `go mod tidy`, vendoring if needed |
| **GitHub API rate limits** | Version checks fail for many users | Cache results, allow skipping with `--offline` flag |
| **Schema version drift** | Kubernetes schemas become outdated | Document schema update process, automate via CI |

---

## Success Metrics

### Before Launch
- [ ] 100% test coverage on validators
- [ ] All kots-lint test cases pass
- [ ] Parity testing with 50+ real vendor applications
- [ ] Linting completes in < 5 seconds for typical app
- [ ] Zero regressions reported by beta testers

### After Launch (3 months)
- [ ] 80% of vendors using `replicated kots lint` instead of curl
- [ ] < 5 bugs reported
- [ ] Average lint time < 3 seconds
- [ ] 90% positive feedback from vendors

---

## Documentation Requirements

### User Documentation

1. **Getting Started Guide**
   ```bash
   # Install replicated CLI
   curl https://... | bash
   
   # Lint your app
   replicated kots lint ./my-app
   
   # View output in JSON
   replicated kots lint ./my-app --output json
   ```

2. **Migration Guide** for vendors currently using `lint.replicated.com`
   - Before: `tar cvf - app/ | curl ...`
   - After: `replicated kots lint app/`
   - Feature comparison
   - Deprecation timeline

3. **Linting Rules Reference**
   - Complete list of all lint rules
   - Explanation of each rule
   - How to fix common errors
   - How to disable specific rules (via LintConfig)

### Developer Documentation

1. **Architecture Doc** - How the linter works internally
2. **Adding New Rules** - How to add custom validation rules
3. **Testing Guide** - How to test the linter
4. **Debugging Guide** - Troubleshooting lint failures

---

## Appendix

### Complete List of Lint Rules

From kots-lint source code analysis:

**YAML & Syntax:**
- `invalid-yaml` - YAML syntax errors
- `invalid-rendered-yaml` - YAML invalid after template rendering

**Template Rendering:**
- `unable-to-render` - Template rendering failed
- `config-is-invalid` - Config spec is invalid

**Helm Charts:**
- `helm-archive-missing` - HelmChart manifest without .tgz
- `helm-chart-missing` - .tgz file without HelmChart manifest

**KOTS Versions:**
- `non-existent-target-kots-version` - targetKotsVersion doesn't exist
- `non-existent-min-kots-version` - minKotsVersion doesn't exist

**Annotations:**
- `deployment-phase-annotation` - Invalid creation/deletion phase
- `wait-for-properties-annotation` - Invalid wait-for-properties JSONPath

**Status Informers:**
- `invalid-status-informer-format` - Wrong format
- `nonexistent-status-informer-object` - Referenced object doesn't exist

**Kubeval:**
- `kubeval-schema-not-found` - No schema for resource type
- `kubeval-error` - Schema validation error
- Various field-specific errors from Kubernetes schemas

**Installers:**
- `kubernetes-installer-{type}` - Kurl installer errors
- `ec-version-required` - Embedded Cluster version missing
- `non-existent-ec-version` - EC version doesn't exist or is pre-release

**OPA Rules (100+ rules in rego files):**
- Secret detection rules
- Required spec rules (Application, Config, etc.)
- Best practice rules
- Many more in `kots-spec-opa-nonrendered.rego` and `kots-spec-opa-rendered.rego`

---

### Example Usage

```bash
# Basic usage
$ replicated kots lint ./my-kots-app
✓ YAML syntax valid
✓ No hardcoded secrets found
✓ Templates rendered successfully
✓ Helm charts validated
✓ KOTS versions verified
✓ Kubernetes schemas valid
✓ All checks passed!

# With errors
$ replicated kots lint ./my-kots-app
✗ YAML syntax errors found:
  deployment.yaml:42 - unexpected end of file
  
✗ Helm chart errors found:
  HelmChart nginx:1.2.3 missing corresponding .tgz file
  
Linting failed with 2 errors

# JSON output
$ replicated kots lint ./my-kots-app --output json
{
  "lintExpressions": [
    {
      "rule": "invalid-yaml",
      "type": "error",
      "path": "deployment.yaml",
      "message": "unexpected end of file",
      "positions": [{"start": {"line": 42}}]
    }
  ],
  "isLintingComplete": false
}

# Offline mode (skip GitHub API checks)
$ replicated kots lint ./my-kots-app --offline
ℹ Skipping KOTS version verification (offline mode)
✓ All offline checks passed!
```

---

## Questions & Answers

### Q: Why not keep the HTTP service?
**A:** The service requires network connectivity, adds latency, and creates operational overhead. Local linting is faster, works offline, and simplifies the vendor workflow.

### Q: Will the HTTP API be deprecated?
**A:** Not immediately. We'll maintain it during a transition period (6-12 months) while vendors migrate to the CLI.

### Q: What about Builders/Enterprise/Troubleshoot linters?
**A:** Those are out of scope for Phase 1. We may migrate them later if there's demand.

### Q: How do we keep schemas up to date?
**A:** We'll document the schema update process and potentially automate it via CI to pull latest schemas from upstream repos.

### Q: What if vendors need custom lint rules?
**A:** Future enhancement. For now, they can use LintConfig to disable unwanted rules. Custom rules can be added via OPA in a future version.

---

## Approval

- [ ] Engineering Lead
- [ ] Product Manager
- [ ] Technical Writer
- [ ] QA Lead

---

**End of PRD**

