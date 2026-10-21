# KOTS and EC template-function compatibility

Add the three template functions missing from KOTS so application releases can share templates with EC v3.
Existing KOTS functions and registry upload layouts remain unchanged.

## Goals

- Add `HelmValue`, `ReplicatedImageName`, and `ReplicatedImageRegistry` to KOTS.
- Match EC template inputs and packaged-default lookup semantics while using KOTS's actual image destinations.
- Verify rendering through public template and release-rendering APIs.

## Non Goals

- Change existing KOTS functions, arithmetic, installer contexts, or image uploads.
- Add duplicate image uploads to make KOTS and EC paths identical.
- Modify EC v2 or make KOTS adopt EC v2-specific behavior.

## Background

EC and KOTS provide overlapping Replicated template functions, but EC v3 chart templates can use three functions that KOTS did not register.
The shared inventory in `pkg/template/testdata/function-contract.json` records Go implementation signatures, and shared rendering fixtures cover both Replicated delimiters and pipelines.
Native Helm templates and Troubleshoot's Helm-based contexts remain separate.

## High-Level Design

Register the missing functions alongside existing KOTS contexts.
Keep existing context functions ahead of explicit `Builder.Functs` entries; the new `HelmValue` chart override is the sole exception.
Resolve and cache the matching packaged chart defaults when `HelmValue` first executes.
Do not pre-parse documents to detect calls; normal KOTS rendering controls the delimiter sequence.
Cache resolution with a simple per-builder check, consistent with sequential template rendering.
Chart name and version may use other template functions but cannot use `HelmValue`, because the archive must be identified before its defaults are available.
Use the existing local registry settings and `imageutil.DestImage` for full local image references.

## Detailed Design

### Template-facing signatures

```go
func HelmValue(path string) string
func ReplicatedImageName(image string, noProxy ...bool) string
func ReplicatedImageRegistry(registry string, noProxy ...bool) string
```

These signatures describe template values rather than internal Go error returns.
Implementation errors stop rendering and include operation and manifest context.
Existing KOTS function behavior is preserved, including its existing error and default-value contracts.

### Packaged chart defaults

`HelmValue` reads packaged defaults for the matching chart, including bundled dependencies and dependency aliases, with parent defaults taking precedence.
This is a packaged-default lookup, matching EC, rather than native Helm's final coalesced values.
It does not propagate parent `global` values into dependency subtrees or evaluate dependency conditions and tags.
Defaults remain available for disabled dependencies and library charts; library charts are not excluded from Helm's values solely because of their type.
Native Helm applies its own dependency processing and global propagation later, when rendering the chart.
It does not read customer overrides or execute native Helm templates.
Dependencies are matched by name and version constraint, and multiple bundled sources for the same alias fail deterministically.
Chart context applies to the `kots.io` HelmChart manifest, matching EC's HelmChart manifest rendering, not to native Helm templates or their rendered output.
Native chart templates use Helm's `.Values`; Replicated `HelmValue` calls in the later base pass receive the ordinary empty fallback.
Outside a chart context or for a missing value path, it returns an empty string, matching Helm’s direct missing-value behavior.
Path lookup retains EC’s GJSON semantics rather than introducing a new path validator; characters such as `[` can occur in literal YAML keys.
An actual call inside a HelmChart requires a matching packaged archive; missing, unreadable, or malformed archives fail with chart name, version, and upstream manifest context.
A plain mention in a comment or string does not trigger archive lookup.
A call in an inactive branch also does not trigger archive lookup.
Each chart gets an independent builder so defaults do not leak into subsequent renders.
KOTS’s existing sequential Replicated delimiter passes remain unchanged, followed by native Helm rendering where applicable.
Legacy HelmChart discovery retains its existing whole-file eligibility check; this change does not discover additional charts hidden inside multi-document files.
Root-level chart files and files under a common chart directory are supported.

Paths use the existing EC GJSON syntax, optionally prefixed with `.Values` or a leading dot.
Dots separate nested keys; escape literal dots and query metacharacters with a backslash.
For a literal `log.level` key, a template can use a raw string argument:

```yaml
logLevel: 'repl{{ HelmValue `.Values.log\.level` }}'
```

An unescaped `.Values.log.level` addresses the nested `log` → `level` value instead.
Retaining this distinction preserves existing EC template behavior rather than guessing between literal and nested keys.

### Image routing and credentials

`ReplicatedImageName` rewrites a full image reference through the configured Replicated proxy online.
`noProxy=true` preserves the original online reference.
Local registry installations, including airgap installations, use `imageutil.DestImage`, the same destination calculation as the existing `LocalImageName` function and KOTS uploads.
The full configured registry namespace is preserved, including multiple path segments.

KOTS flattens the upstream repository path while EC preserves it.
For example, `docker.io/library/nginx:1` becomes `registry:5000/customer/nginx:1` in KOTS and `registry:5000/customer/library/nginx:1` in EC.
Neither product uploads duplicate aliases to force identical output.

`ReplicatedImageRegistry` returns the configured registry prefix in local mode and a proxy prefix online.
Local mode ignores the supplied registry argument entirely, including malformed prefixes, because the configured local prefix replaces it; this matches EC.
An empty online registry argument returns an empty string; locally it returns the configured registry prefix, matching EC v3.
A separate repository field must already match KOTS's flattened upload layout.
A registry-only function cannot rewrite a separate repository field, so use `ReplicatedImageName` when a complete portable reference is needed.
The new functions resolve registry configuration through the same existing KOTS registry policy used by `LocalImageName`.
References already served by the configured Replicated registry stay direct; upstream registry references are mapped to the custom registry domain before considering the proxy.
They do not create credentials or upload images.

### Existing function compatibility

Existing KOTS arithmetic is unchanged.
Mixed signed/unsigned operands can fail rendering; convert operands to the same numeric type first.
Integer division by zero fails rendering, while floating-point division follows Go's infinity and NaN behavior.
Existing installer, identity, license, Kubernetes, and private-CA contexts retain their implementations.
The EC side supplies product-specific defaults for unsupported capabilities; this KOTS change does not replace existing KOTS contexts with those defaults.

## Testing

Use `NewBuilder`, `Builder.WithHelmChart`, and `Builder.String` to verify the new functions through the public template surface.
Use `RenderKotsKinds` to verify matching archive selection, separate chart contexts, real function calls versus literal mentions, and contextual failures.
Cover bundled dependencies and aliases, root-level and prefixed archives, malformed archives, missing paths, escaped literal keys, nested local registry namespaces, online bypass, and existing local-image parity.
Unit tests avoid live Kubernetes credential lookup by leaving the builder namespace empty when no cluster-dependent behavior is under test.
Run `go test ./pkg/template ./pkg/base` from the KOTS checkout.
Check shared inventories and fixtures against EC before release.
Numeric release floors must identify the first published KOTS and EC artifacts containing these changes rather than an assumed source-checkout version.

## Alternatives Considered

Changing KOTS upload paths or adding duplicate image aliases was rejected because image references must match each product's actual uploads.
Treating dotted keys as literal without escaping was rejected because it would change EC's existing GJSON path contract and make nested paths ambiguous.
Finding `HelmValue` by raw substring was rejected because comments and plain strings would spuriously require a packaged chart.
Changing existing KOTS functions to match EC defaults was rejected because the story adds missing functions and does not redefine existing KOTS behavior.

## Security Considerations

Packaged defaults originate in the vendor's release archive and remain untrusted input to parsing.
The reader does not extract files to the host filesystem or execute native chart templates.
Malformed archives and metadata fail with contextual errors, and excessive dependency recursion is rejected.
Archive size is not artificially limited per dependency because existing valid releases can contain large dependency assets.
Only defaults required for this context are retained; lookup does not access customer overrides, cluster Secrets, or network registries.

Image routing uses existing KOTS registry configuration and produces references consistent with actual uploads.
The new image functions do not return credentials or log credential values.
The split registry/repository limitation must remain documented so templates do not accidentally reference a repository that KOTS never uploaded.
