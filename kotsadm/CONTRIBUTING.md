# Contributing

Contributions are definitely welcome! We've published documentation on how to set up a local dev environment (or, least one way to do it!) here:

## Local Setup

The recommended configuration to build and run kotsadm on a microk8s cluster, locally.

Required Software:
- [Skaffold](https://skaffold.dev) 0.29.0 or later
- [Kustomize](https://kustomize.io) 2.0 or later
- Kubernetes (Recommended to run microk8s)

## Running

Build Kotsadm go binary:

```
make kotsadm
```

Apply Schemahero CRDs

```
kubectl apply -f migrations/kustomize/base/schemahero.yaml
```

Next, you can build and run all server components in the Kubernetes cluster with:

```
skaffold dev
```

## Notes:
- Go code will not be rebuilt automatically.  Run `make kotsadm` again to make the new binary and restart the pod.
- After installing restic/velero, edit the restic daemonset to change the volume hostPath mount from:
      `/var/lib/kubelet/pods` to `/var/snap/microk8s/common/var/lib/kubelet/pods`

