# Running in MicroK8s

It's possible to run Ship Cluster in Microk8s.

### Enable the registry

```
microk8s.enable registry
```

### Use the registry in your dev environment

Add a `--default-repo localhost:32000` to your Skaffold command. For example:

```
skaffold dev --profile github --default-repo localhost:32000
```
