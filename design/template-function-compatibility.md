# KOTS and EC template-function compatibility

EC and KOTS register the union of their product template functions and Sprig
v3.3.0 functions. The checked-in inventory in
`pkg/template/testdata/function-contract.json` records internal Go implementation signatures for automated checks.
Shared rendering fixtures cover `repl{{ ... }}`, `{{repl ... }}`, and pipelines.
Unknown function names and invalid arguments still fail rendering.
Existing KOTS arithmetic is unchanged. Mixed signed/unsigned operands can fail
rendering; convert operands to the same type first. Integer division by zero
fails rendering. Floating-point division by zero produces signed infinity or NaN.

These functions are Replicated templates. Application Preflight v1beta3 and
native Helm chart templates continue to use Helm's own context; they are not
reinterpreted as Replicated templates.

## Added template functions

These signatures show the values available to template authors, following the
Replicated template-function documentation format. If an implementation reports
an error, template rendering stops and reports that failure; an error is not a
value available to the template.

### Added to KOTS

```go
func HelmValue(path string) string
func ReplicatedImageName(image string, noProxy ...bool) string
func ReplicatedImageRegistry(registry string, noProxy ...bool) string
```

### Added to EC

```go
func HasLocalRegistry() bool
func LocalImageName(remoteImageName string) string
func LocalRegistryAddress() string
func LocalRegistryHost() string
func LocalRegistryNamespace() string
func LocalRegistryImagePullSecret() string
func IdentityServiceEnabled() bool
func IdentityServiceClientID() string
func IdentityServiceClientSecret() string
func IdentityServiceName() string
func IdentityServicePort() string
func IdentityServiceRoles() map[string]interface{}
func KotsVersion() string
func KurlAll() string
func KurlString(yamlPath string) string
func KurlOption(yamlPath string) string
func KurlBool(yamlPath string) bool
func KurlInt(yamlPath string) int
func NodeCount() int
```

## Installation context and defaults

| Function or context | EC behavior | KOTS behavior |
| --- | --- | --- |
| `KotsVersion` | Empty string; EC does not install KOTS | Installed KOTS version |
| `IsKurl` | Always false | Actual installation type |
| `KurlAll`, `KurlString`, `KurlOption` | Empty string | Installer data, or empty string without kURL |
| `KurlBool`, `KurlInt` | False, integer 0 | Installer data, or typed defaults without kURL |
| Identity service functions | False, empty strings, empty roles map | Existing configured identity service |
| `Distribution` | `embedded-cluster` | Discovered cluster provider |
| `NodeCount` | Live node count when cluster access is available; otherwise 0 | Live node count, otherwise 0 |
| `KubernetesVersion` | Release Kubernetes version; `0.0.0+unknown` without context | Live version; `0.0.0+unknown` without context |
| `KubernetesMajorVersion`, `KubernetesMinorVersion` | Release version components, or empty strings | Live version components, or empty strings |
| `Lookup` | Empty map before cluster availability or when an object is absent; live errors propagate | Existing KOTS lookup behavior |
| `PrivateCACert` | ConfigMap name when a private CA is configured; otherwise empty | Existing KOTS CA ConfigMap context |
| `HelmValue` | Packaged chart defaults in the matching chart context; empty string outside chart context | Same |

`HelmValue` includes defaults from bundled dependencies and their aliases, with
parent defaults taking precedence. It does not read customer overrides or execute
Helm templates. Missing value paths produce empty strings; unreadable or malformed
chart archives produce errors. Chart context does not leak into later rendering.

Configuration defaults and generated-value persistence are a separate contract.
Registering `ConfigOptionIndex` does not add repeatable configuration support.

## Images and credentials

`LocalImageName` retains KOTS's routing decisions: use the local/customer registry
when configured; otherwise leave public references unchanged unless
`Application.spec.proxyPublicImages` is true, and route private images through
the configured Replicated domains. Access checks and malformed image references
can fail rendering; they are not converted to empty output.

The returned path must match the product's actual registry uploads. KOTS uses
its flattened layout (`<registry>/<namespace>/<image>`); EC preserves repository
paths (`<registry>/<namespace>/<repository>/<image>`). For example,
`docker.io/library/nginx:1` in a customer registry becomes
`registry:5000/customer/nginx:1` in KOTS and
`registry:5000/customer/library/nginx:1` in EC. Neither upload path publishes
duplicate images to force identical function output.

`ReplicatedImageName` uses that same product-specific local destination.
`ReplicatedImageRegistry` returns the configured registry address and repository
namespace in local mode; a split repository field must match the product's
upload layout. A registry-only function cannot flatten a separate repository
field, so use `ReplicatedImageName` for a complete portable image reference.
Both functions proxy online images by default and accept the optional `noProxy`
boolean. `noProxy` applies only to online operation. `LocalImageName` instead
uses the public/private routing decision described above.

`HasLocalRegistry` is false for proxy-only online operation.
`LocalRegistryHost` includes the port; `LocalRegistryNamespace` is a repository
path, not a Kubernetes namespace. `LocalRegistryAddress` joins those values.

`LocalRegistryImagePullSecret` returns **base64-encoded dockerconfigjson**, not a
Secret name. Online credentials authenticate to the configured Replicated domains;
local/customer credentials authenticate to the registry authority without its
repository path. `ImagePullSecretName` returns `<app-slug>-registry`. EC ensures
this Secret in the application, additional, and chart namespaces alongside private
CA ConfigMaps. Containerd authentication continues to work independently. Pods,
Jobs, and operator-created workloads can reference this Secret in those namespaces.
Workloads in other namespaces must arrange credentials there as well.

## Recovery bootstrap

Recovery registers the common surface using bootstrap context. Saved config and
license/version metadata are unavailable before metadata restore. Executed
saved-state lookups return typed defaults and are recorded. Recovery chart
validation rejects those lookups before provisioning. An inactive branch can
contain a saved-state function without blocking recovery. Recovery charts should
use bootstrap inputs and packaged chart defaults for their infrastructure.
Backup/Restore policy uses the ordinary builder after saved metadata is available,
including the actual `LocalImageName` behavior.

## Release requirements and version floor

This contract targets the first future KOTS and EC releases containing these
changes. No already shipped binary gains new functions. A numeric version floor
must be assigned to the actual published artifacts; it cannot be inferred from a
source checkout or a planned release number.

The local EC v2 source reference at `c2071e2462e29611b1726576d58d8eb9df127097`
(latest reachable tag `2.19.13+k8s-1.34`) pins Admin Console
`1.132.0-ec.3` in `pkg/addons/adminconsole/static/metadata.yaml`. That reference
predates this contract and must not be advertised as supporting the new functions.
It is not an inventory of all supported EC v2 releases.

Before releasing an application that uses the new functions:

1. Record the first released KOTS and EC versions containing the contract.
2. For each supported EC v2 installer, inspect its embedded release metadata and
   check the actual bundled KOTS binary against the inventory and fixtures. Require
   an intermediate KOTS/EC v2 update when that binary lacks the new functions.
3. Verify real workload pulls, including Jobs and operator-created Pods, online,
   airgap and with customer registries. The local registry tests verify authenticated
   upload/download and digest preservation; they do not certify Kubernetes admission
   behavior or every customer registry.
4. Exercise recovery bootstrap and a post-migration backup/restore with the actual
   release artifacts. Keep application Preflight v1beta2/v1beta3 packaging distinct.

## Maintaining the contract

Run the existing `make agent/test-unit` entry point from EC. When checking
both repositories, also compare the function inventory, shared rendering and
license fixtures, packaged-default handling, and image tests in `pkg/template`.
A future function or Sprig change must update both repositories and the contract
before release.
