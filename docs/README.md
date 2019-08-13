# Kubernetes Off The Shelf (KOTS) Software

These are the docs for the kots open source application. If you are looking for the docs on using kots as a Go library, they are on [Godocs](https://godoc.org/github.com/replicatedhq/kots).

## Quick Start

```shell
krew install kots
kubectl kots download helm://stable/mysql@1.3.0
```

This will create a directory named `mysql` that contains 3 subdirectories:

```
|- mysql
   |- upstream
   |- base
   |- overlays
      |- midstream
```

### upstream

### base

### overlays

### overlays/midstream
