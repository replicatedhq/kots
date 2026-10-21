package template

import (
	"fmt"
	"strings"

	"github.com/pkg/errors"

	"github.com/distribution/reference"
	"github.com/replicatedhq/kots/pkg/docker/registry"
	registrytypes "github.com/replicatedhq/kots/pkg/docker/registry/types"
	"github.com/replicatedhq/kots/pkg/imageutil"
	kotsv1beta1 "github.com/replicatedhq/kotskinds/apis/kots/v1beta1"
)

// replicatedImageRegistryInfo preserves KOTS's direct registry and upstream endpoints.
func (ctx ConfigCtx) replicatedImageRegistryInfo() (*registry.RegistryProxyInfo, error) {
	info, err := registry.GetRegistryProxyInfo(ctx.license, &kotsv1beta1.Installation{Spec: InstallationSpecFromVersionInfo(ctx.VersionInfo)}, ctx.app)
	if err != nil {
		return nil, errors.Wrap(err, "get registry proxy info")
	}
	return info, nil
}

// replicatedImageName returns a full image reference using the installation's upload layout.
func (ctx ConfigCtx) replicatedImageName(image string, noProxy ...bool) (string, error) {
	if image == "" {
		return "", nil
	}
	named, err := reference.ParseNormalizedNamed(image)
	if err != nil {
		return "", errors.Wrapf(err, "parse image reference %q", image)
	}
	if ctx.hasLocalRegistry() {
		destination, err := imageutil.DestImage(registrytypes.RegistryOptions{Endpoint: ctx.localRegistryHost(), Namespace: ctx.localRegistryNamespace()}, image)
		if err != nil {
			return "", errors.Wrapf(err, "compute local registry destination for image %q", image)
		}
		return destination, nil
	}
	if len(noProxy) > 0 && noProxy[0] {
		return image, nil
	}
	info, err := ctx.replicatedImageRegistryInfo()
	if err != nil {
		return "", errors.Wrapf(err, "resolve registry routing for image %q", image)
	}
	if matchesDomain(image, info.Registry) {
		return image, nil
	}
	if info.Registry != "" && matchesDomain(image, info.Upstream) {
		return strings.Replace(image, info.Upstream, info.Registry, 1), nil
	}
	proxy := info.Proxy
	if proxy == "" {
		return "", errors.Errorf("resolve proxy routing for image %q: replicated proxy domain is required", image)
	}
	if ctx.AppSlug == "" {
		return "", errors.Errorf("resolve proxy routing for image %q: application slug is required", image)
	}

	if matchesDomain(image, proxy) {
		return image, nil
	}
	if matchesDomain(image, "proxy.replicated.com") {
		return strings.Replace(image, "proxy.replicated.com", proxy, 1), nil
	}
	return fmt.Sprintf("%s/proxy/%s/%s", proxy, ctx.AppSlug, named.String()), nil
}

// replicatedImageRegistry returns a registry prefix; separate repositories must match the upload layout.
func (ctx ConfigCtx) replicatedImageRegistry(registry string, noProxy ...bool) (string, error) {
	// Match EC: local routing replaces the supplied prefix entirely, so the
	// unused argument is neither parsed nor validated in this mode.
	if ctx.hasLocalRegistry() {
		return ctx.localRegistryAddress(), nil
	}
	if registry == "" {
		return "", nil
	}
	if _, err := reference.ParseNormalizedNamed(registry + "/image"); err != nil {
		return "", errors.Wrapf(err, "parse registry reference %q", registry)
	}
	if len(noProxy) > 0 && noProxy[0] {
		return registry, nil
	}
	info, err := ctx.replicatedImageRegistryInfo()
	if err != nil {
		return "", errors.Wrapf(err, "resolve registry routing for registry %q", registry)
	}
	if matchesDomain(registry, info.Registry) {
		return registry, nil
	}
	if info.Registry != "" && matchesDomain(registry, info.Upstream) {
		return strings.Replace(registry, info.Upstream, info.Registry, 1), nil
	}
	proxy := info.Proxy
	if proxy == "" {
		return "", errors.Errorf("resolve proxy routing for registry %q: replicated proxy domain is required", registry)
	}
	if ctx.AppSlug == "" {
		return "", errors.Errorf("resolve proxy routing for registry %q: application slug is required", registry)
	}

	if matchesDomain(registry, proxy) {
		return registry, nil
	}
	if matchesDomain(registry, "proxy.replicated.com") {
		return strings.Replace(registry, "proxy.replicated.com", proxy, 1), nil
	}
	return fmt.Sprintf("%s/proxy/%s/%s", proxy, ctx.AppSlug, registry), nil
}

func matchesDomain(value, domain string) bool {
	return domain != "" && (value == domain || strings.HasPrefix(value, domain+"/"))
}
