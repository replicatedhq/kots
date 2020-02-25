# Contributing

Contributions are definitely welcome! We've published documentation on how to set up a local dev environment (or, least one way to do it!) here:

## Local Setup

The recommended configuration to build and run Kotsadm locally is to run everything on a Docker Desktop environment, with the local, Docker-supplied Kubernetes installation.

Required Software:
- [Skaffold](https://skaffold.dev) 0.29.0 or later
- [Kustomize](https://kustomize.io) 2.0 or later
- Kubernetes (Recommended to run microk8s)

## Running

To start, build and run all server components in the Kubernetes cluster with:

```
skaffold dev
```

Go code will not be rebuilt automatically.  Run `make kotsadm` to make the new binary and restart the pod.