# Developing in GitHub Codespaces

This is still early and probably could use some optimization and stabilization. But if you want to create a dev environmnet in GitHub Codespaces, here's a guide to help you.

First, create a new Codespace at github.com/codespaces. I choose a 16 core, 32 GB RAM codespace for this experiment.

## One time installation of depenedencies

Once you've opened the new codespace, run:

```
./hack/dev/codespace.sh
```

This will do a few things:

1. Install a couple of apt dependencies that we need
2. Install Kind and create a local Kubernetes cluster
3. Install a Docker registry (distribution) and connect it to the Kind network
4. Install some more deps (krew, schemahero, skaffold, kustomize)


## Make the base images

```
make cache
```

## Start the dev environment

```
skaffold dev
```

This will use Skaffold's user port forarding because NodePorts are not supported in kind (at least not in the same way).

## What's Nexts?

The one time setup should not be required. There's a beta feature of Codespaces to pre-build images. Let's explore that.

We should remove the NodePort definitions from the dev overlays and use the Skaffold port forwarding exclusively to make this simpler.