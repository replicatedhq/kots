# Contributing

Contributions are definitely welcome! We've published documentation on how to set up a local dev environment (or, least one way to do it!) here:

## Local Setup

The recommended configuration to build and run kotsadm on a microk8s cluster, locally.

Required Software:
- [Skaffold](https://skaffold.dev) 1.10.1 or later
- [Kustomize](https://kustomize.io) 2.0 or later
- Kubernetes (Recommended to run microk8s)

## Running

Build Kotsadm go binary:

```
GOOS=linux make kotsadm
```

Next, you can build and run all server components in the Kubernetes cluster with:

```
skaffold dev
```

## Notes:
- Go code will not be rebuilt automatically.  Run `make kotsadm` again to make the new binary and restart the pod.
- After installing restic/velero, `kubectl -n velero edit daemonset restic` to change the volume hostPath mount from:
      `/var/lib/kubelet/pods` to `/var/snap/microk8s/common/var/lib/kubelet/pods`

