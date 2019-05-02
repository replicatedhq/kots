# Ship Cluster

Ship Cluster is a locally-installable application that manages [Helm Charts](https://helm.sh) and other Kubernetes applications in a workflow that's built for day-2 operations.

## Installing

To install Ship into your cluster, we recommend using Ship or Helm. If you use Helm, you'll be able to "unfork" the after installation, and migrate the install -> configure -> update process into a gitops workflow.

### Ship

```
brew install ship
ship init github.com/shipapps/ship-cluster
```

### Helm

```
helm install ship
```

## Contributing

For docs on setting up a dev environment to build and run Ship Cluster locally, read the [CONTRIBUTING](https://github.com/replicatedhq/ship-cluster/blob/master/CONTRIBUTING.md) guide.
