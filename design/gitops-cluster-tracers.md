# GitOps Cluster Tracers

KOTS (Admin Console) does not have the ability to detect the currently deployed application 
version in a GitOps cluster workflow. This proposal adds tracers (annotations) to the
application pods so that it can be possible to detect that in the future.

## Goals

- Add annotations to application pods to help detect deployments in GitOps clusters in the future

## Non Goals

- Detect the currently deployed application version in a GitOps cluster

## Background

When customers decide to deploy their application via a GitOps workflow, they will
probably want to know which version of the app is currently deployed in the cluster,
which is currently not possible.

## High-Level Design

Add annotations for both the application slug and sequence to all of the application pods via Kustomize.
Those annotations will be applied to the downstream manifests before commiting the new changes.

## Detailed Design

When (re)writing the midstream in KOTS, the following will be injected into the kustomization.yaml file:

```
commonAnnotations:
  kots.io/app-slug: <app-slug>
  kots.io/app-sequence: <app-sequence>
```

the `<app-slug>` will be obtained from the license
the `<app-sequence>` will be added to the pull/rewrite options before calling kots.pull or kots.rewrite

The rest of the logic is already implemented. Before creating new commits, `kustomize build` is
executed to apply those changes to the downstream manifests.
