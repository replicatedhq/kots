# Kubernetes Off The Shelf (KOTS) Software

## Installing

kots is provided as a kubectl plugin using [krew](https://krew.dev). To install kots:

```shell
kubectl krew install kots
```

## CLI

### `kubectl kots install`
The `install` command is the recommended way to learn kots. Executing the `install` command will install an application and the [kotsadm](https://github.com/replicatedhq/kotsadm`) Admin Console to an existing Kubernetes cluster. This command supports installing Helm charts (without Tiller), standard Kubernetes applications and also Replicated apps.

To try it, just choose a helm chart ([Elasticsearch](https://github.com/elastic/helm-charts/tree/master/elasticsearch)) and run the following command:

```
kubectl kots install helm://elastic/elasticsearch --repo https://helm.elastic.co --namespace elasticsearch
```

After this command completes, the kotsadm Admin Console will be running in your cluster, listening on port :8800 on a ClusterIP service in the namespace you deployed the application to. You can connect to this using kubectl port-forward, or set up an ingress/load balancer of your own.

```
kubectl port-forward --namespace elasticsearch svc/kotsadm 8800:8800
```

And now visit http://localhost:8800 to set the Elasticsearch Admin Console.


### `kubectl kots pull`
The `pull` command will create a local directory set up so you can create Kustomize-friendly patches and then use kubectl to deploy to a cluster yourself. The pull command will not add the admin console to a cluster or install anything in your cluster.

```
kubectl kots pull helm://elastic/elasticsearch --repo https://helm.elastic.co --namespace elasticsearch
kubectl apply -k ./elasticsearch/overlays/midstream
```

### `kubectl kots upload`
The `upload` command will upload a directory with an upstream, base and overlays directory to a kotsdm server.

```
kubectl kots upload ~/mysql
```

### `kubectl kots configure`
The `configure` command will start a local browser-based configuration screen for Replicated apps.

### `kubectl kots watch`
