package template

import (
	"fmt"
	"github.com/distribution/reference"
	"github.com/replicatedhq/kots/pkg/docker/registry"
	registrytypes "github.com/replicatedhq/kots/pkg/docker/registry/types"
	"github.com/replicatedhq/kots/pkg/imageutil"
	kotsv1beta1 "github.com/replicatedhq/kotskinds/apis/kots/v1beta1"
	"strings"
	"text/template"
)

type RegistryMode string

const (
	RegistryModeProxy  RegistryMode = "proxy"
	RegistryModeAirgap RegistryMode = "airgap"
	RegistryModeBYO    RegistryMode = "byo"
)

type RegistryInfo struct{ RegistryAddress, BYORegistryAddress, ProxyDomain, RegistryDomain string }
type imageCtx struct {
	mode     RegistryMode
	appSlug  string
	registry RegistryInfo
}

func (ctx ConfigCtx) replicatedImageContext() (imageCtx, error) {
	info, err := registry.GetRegistryProxyInfo(ctx.license, &kotsv1beta1.Installation{Spec: InstallationSpecFromVersionInfo(ctx.VersionInfo)}, ctx.app)
	if err != nil {
		return imageCtx{}, err
	}
	image := imageCtx{mode: RegistryModeProxy, appSlug: ctx.AppSlug, registry: RegistryInfo{ProxyDomain: info.Proxy, RegistryDomain: info.Registry}}
	if ctx.hasLocalRegistry() {
		image.mode = RegistryModeBYO
		image.registry.BYORegistryAddress = ctx.localRegistryAddress()
	}
	return image, nil
}
func (ctx ConfigCtx) replicatedImageName(image string, np ...bool) (string, error) {
	c, err := ctx.replicatedImageContext()
	if err != nil {
		return "", err
	}
	return c.replicatedImageName(image, np...)
}
func (ctx ConfigCtx) replicatedImageRegistry(registry string, np ...bool) (string, error) {
	c, err := ctx.replicatedImageContext()
	if err != nil {
		return "", err
	}
	return c.replicatedImageRegistry(registry, np...)
}

// newFunctionDefaults is added before KOTS contexts, preserving their behavior.
type newFunctionDefaults struct{}

func (newFunctionDefaults) FuncMap() template.FuncMap {
	return template.FuncMap{"HelmValue": func(string) (string, error) { return "", nil }}
}

// noProxy returns true if the first element of the variadic noProxy slice is true.
func noProxy(noProxy []bool) bool {
	return len(noProxy) > 0 && noProxy[0]
}

// replicatedImageName rewrites a full image reference for the current environment.
// Use this when a chart stores the entire image in one field (e.g. "ghcr.io/org/app:v1").
//
// In online mode the image is routed through the Replicated proxy. If the image
// already uses the configured proxy domain it is returned as-is. If it uses the
// default proxy.replicated.com but a custom domain is configured, the domain is swapped.
// Pass noProxy=true for public images that should not be proxied.
// In airgap mode the image is always rewritten to the local registry.
// Local registry images use KOTS's flattened upload layout. The noProxy hint
// applies only to online proxy operation.
func (ctx imageCtx) replicatedImageName(image string, np ...bool) (string, error) {
	if image == "" {
		return "", nil
	}
	if _, err := reference.ParseNormalizedNamed(image); err != nil {
		return "", fmt.Errorf("parse image reference %q: %w", image, err)
	}

	if ctx.mode == RegistryModeAirgap {
		if ctx.registry.RegistryAddress == "" {
			return "", fmt.Errorf("registry address is required")
		}
		return localRegistryImageRef(image, ctx.registry.RegistryAddress)
	}

	if ctx.mode == RegistryModeBYO {
		if ctx.registry.BYORegistryAddress == "" {
			return "", fmt.Errorf("BYO registry address is required")
		}
		return localRegistryImageRef(image, ctx.registry.BYORegistryAddress)
	}

	if noProxy(np) {
		return image, nil
	}
	if ctx.registry.ProxyDomain == "" || ctx.appSlug == "" {
		return "", fmt.Errorf("proxy domain and app slug are required")
	}
	// If the image already uses the configured proxy domain, return unchanged.
	if matchesDomain(image, ctx.registry.ProxyDomain) {
		return image, nil
	}
	// If the image uses the default proxy domain but a custom domain is
	// configured, swap the domain prefix.
	if matchesDomain(image, "proxy.replicated.com") {
		return strings.Replace(image, "proxy.replicated.com", ctx.registry.ProxyDomain, 1), nil
	}
	// Normalize the image reference so that Docker Hub images always include
	// docker.io/library (e.g. "nginx" → "docker.io/library/nginx"). The proxy
	// requires the upstream registry in the path to route correctly.
	named, err := reference.ParseNormalizedNamed(image)
	if err != nil {
		return "", fmt.Errorf("parse image reference %q: %w", image, err)
	}
	// named.String() returns the fully qualified reference including tag/digest if present
	return fmt.Sprintf("%s/proxy/%s/%s", ctx.registry.ProxyDomain, ctx.appSlug, named.String()), nil
}

// replicatedImageRegistry rewrites a registry host for the current environment.
// Use this when a chart splits image references into separate registry and repository
// fields (e.g. registry: "ghcr.io", repository: "org/app").
//
// In online mode the registry is rewritten to the proxy path
// (e.g. "ghcr.io" → "proxy.myapp.com/proxy/myapp/ghcr.io"). If the input already
// uses the configured proxy domain it is returned as-is. If it uses the default
// proxy.replicated.com but a custom domain is configured, the domain is swapped.
// Pass noProxy=true for public images that should not be proxied.
// In airgap mode the local registry address is always returned.
// In BYO mode the BYO registry address (which may include a namespace path) is
// always returned; the original upstream registry is dropped because the
// customer's registry is the only source charts pull from.
func (ctx imageCtx) replicatedImageRegistry(registry string, np ...bool) (string, error) {
	switch ctx.mode {
	case RegistryModeAirgap:
		if ctx.registry.RegistryAddress == "" {
			return "", fmt.Errorf("registry address is required")
		}
		return ctx.registry.RegistryAddress, nil
	case RegistryModeBYO:
		if ctx.registry.BYORegistryAddress == "" {
			return "", fmt.Errorf("BYO registry address is required")
		}
		return ctx.registry.BYORegistryAddress, nil
	default: // RegistryModeProxy
		if noProxy(np) {
			return registry, nil
		}
		if ctx.registry.ProxyDomain == "" || ctx.appSlug == "" {
			return "", fmt.Errorf("proxy domain and app slug are required")
		}
		// If the input already uses the configured proxy domain, return unchanged.
		if matchesDomain(registry, ctx.registry.ProxyDomain) {
			return registry, nil
		}
		// If the input uses the default proxy domain (proxy.replicated.com) but a
		// custom domain is configured, swap the domain prefix.
		if matchesDomain(registry, "proxy.replicated.com") {
			return strings.Replace(registry, "proxy.replicated.com", ctx.registry.ProxyDomain, 1), nil
		}
		return fmt.Sprintf("%s/proxy/%s/%s", ctx.registry.ProxyDomain, ctx.appSlug, registry), nil
	}
}

// matchesDomain returns true if value equals the domain or starts with domain + "/".
func matchesDomain(value, domain string) bool {
	return value == domain || strings.HasPrefix(value, domain+"/")
}

func (ctx imageCtx) replicatedRegistryDomain() string {
	return ctx.registry.RegistryDomain
}

func (ctx imageCtx) replicatedProxyDomain() string {
	return ctx.registry.ProxyDomain
}

// localRegistryImageRef uses the same destination layout as KOTS image uploads.
func localRegistryImageRef(image, address string) (string, error) {
	if address == "" {
		return "", fmt.Errorf("registry address is required")
	}
	host, namespace, _ := strings.Cut(strings.TrimSuffix(address, "/"), "/")
	return imageutil.DestImage(registrytypes.RegistryOptions{Endpoint: host, Namespace: namespace}, image)
}
