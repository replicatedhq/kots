# ocistore

The OCIStore uses an OCI-compatible image registry for persistent storage.
In addition, this store stores some cache in Kubernetes objects (secrets, configmaps).

To use this locally:

```
export KOTSSTORE=oci
skaffold dev
```

## Kubernetes Objects

To enable this store to function quickly, some data is stored in the cluster. 
This store has been designed to have a fixed number of configmaps and secrets per application stored, and the number will not scale with the number of versions of an app, time that an application has been running, or any other metric that's not controlled by the end user.
Activity on an application will not increase the number of objects stored in the cluster.

| Type | Name / Identifier | Description |
|------|-------------------|-------------|
| ConfigMap | `kotsadm-apps` | List of all apps installed |
| ConfigMap | `kotsadm-downstreams` | List of all "downstreams" |
| ConfigMap | `kotsadm-appdownstreams` | Lookup / relationship between apps and downstreams |
| Secret | `kotsadm-sessions | List of all active user sessions |
| ConfigMap | `kotsadm-clusters` | List of all clusters/downstreams |
| Secret | `kotsadm-clustertokens` | Lookup from deploy token to cluster id |
| Secret | `kotsadm-registry` | The details for a locally configured image registry |

## Dev tips

Resetting all state: `k delete configmap -l owner=kotsadm && k delete secret -l owner=kotsadm`

This shows up in the log when bootstrapping: `failed to list installed apps for downstream: failed to get app downstreams list configmap: failed to create configmap: configmaps \"kotsadm-appdownstreams\" already exists`:  Yes, there's a bug.  it's a race condition but it's not causing an error.
