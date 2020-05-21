# Allow public images to be proxied through replicated registry

The goal is to reduce number of domains that need to be accessible through proxies from private networks.

## Goals

- All images will be pulled through proxy.replicated.com or registry.replicated.com.
- None of the other registries need to be accessible through firewalls.

## Non Goals

- There will be no magic anonymous access to registries via proxy.replicated.com:
    - Vendors will still need to connect to external registry with valid credentials even if all images are public.
    - `imagePullSecret` will be required in order to pull public images.
- Public images will not retain original public names on the private instance.  All image names will begin with `proxy.replicated.com`.
- This change will not apply to airgap builds and installs.  If instance has access to pubic registry, missing image from airgap bundle will still be pulled from the internet.

## Background

An end customer may desire to keep access to public internet minimal from a KOTS managed cluster.  Delivering public images to the cluster is the only part that requires access to domains other than replicated.com.

## High-Level Design

1. This will be an opt-in feature, enabled by including the new `proxyPublicImages` flag in the `kots.io/Application` spec.
1. A valid link to the external registry will need to be created by the vendor.
1. Public image will be pulled using valid credentials as if it were a private image.

## Detailed Design

The following field will be added to Application type:

```
type ApplicationSpec struct {
  ...
	ProxyPublicImages            bool            `json:"proxyPublicImages,omitempty"`
  ...
}
```

The call to `IsPrivateImage` function will skipped when `ProxyPublicImages` is set to `true`.  This will require modifying the following functions:
- `localImageName`
- `GetPrivateImages`
- `copyOneImage`
- `localImageName` (the template function used with `additionalImages` key)

## Alternatives Considered

An alternative to this method is to push images to private repos.  These can be hosted by any registry, including registry.replicated.com.  The downside of this method is the requirement that vendors modify their CI process or make currently public repos private.

## Security Considerations

There are no known impacts on security.
