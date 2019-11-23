# Kubernetes Off The Shelf (KOTS) Software

## CLI

### `kots install`
The `install` command is the recommended way to learn KOTS. Executing the `install` command will install an application and the [kotsadm](https://github.com/replicatedhq/kotsadm) Admin Console to an existing Kubernetes cluster. This command supports installing Helm charts (without Tiller), standard Kubernetes applications and also Replicated KOTS apps.

Try installing the Replicated sample app ([Sentry Pro Example](https://github.com/replicatedhq/kots-sentry/)) by first installing KOTS on your workstation
```
curl https://kots.io/install | bash
```

and running the following command:
```
kubectl kots install sentry-pro
```

Set a namespace for the admin console and the application components to be installed, and provide a password for the admin console. After this command completes, the kotsadm Admin Console will be running in your cluster, listening on port :8800 on a ClusterIP service in the namespace you deployed the application to. By default this is exposed to your workstation using kubectl port-forward, but you could set up an ingress/load balancer of your own.

Visit http://localhost:8800 to access the Admin Console, enter the password.

Download the [sample license](https://kots.io/sample-license) for Sentry Pro & upload it to the console. You'll then be presented with configuration settings, preflight checks and other application options.

If you terminate your terminal session, the port-forward will also terminate. To access the admin console again, just run:
```
kubectl kots admin-console --namespace sentry-pro
```


### `kots pull`
The `pull` command will create a local directory set up so you can create Kustomize-friendly patches and then use kubectl to deploy to a cluster yourself. The pull command will not add the admin console to a cluster or install anything in your cluster.

```
kubectl kots pull sentry-pro --namespace sentry-pro
kubectl apply -k ./sentry-pro/overlays/midstream
```

### `kots upload`
The `upload` command will upload a directory with an upstream, base and overlays directory to a kotsadm server.

```
kubectl kots upload ~/sentry-pro
```

### `kots download`
The `download` command will download an application YAML from a kotsadm server. This is especially useful when paired with `upload` (above) to iterate on and make changes to an application.

```
kubectl kots download [--namespace] [app-slug]
```

The app-slug argument is optional. If there is more than 1 application in the specified namespace, kots will prompt for which one to download.
