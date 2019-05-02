# Ship Operator

The Ship operator is a Kubernetes Operator designed to manage the Ship [watch](...) and [update](...) tasks. This operator is designed to be compatible with most workflows, and provides a declarative format to express and deploy the watch and update workflows for [Ship](https://github.com/replicatedhq/ship) applications.

## Deploying
To use the Ship Operator, deploy the controller to a cluster. Only one instance of the controller is required; the controller can manage any number of third party application watch/update events.

```shell
kubectl apply -f https://github.com...
```

## Application Configuration
The ship application can generate a custom resource that is compatible with this controller. After the `ship init` command is completed, run:

```shell
ship operator > ship-operator.yaml
```

This command will create a resource that can be edited and then deployed to the cluster to define the watch and update events.

## Ship Operator Resource
The Ship Operator enables simple, declarative workflows to be written and deployed to a Kubernetes cluster using `kubectl` or any other tool. The workflow can be integrated into a gitops workflow, or any other type of notification or continuous delivery process.

## How it works
When a `ShipWatch` resource is deployed to a cluster, the operaetor will create a deployment with a single pod. This deployment will get a copy of the ship `state.json` and start a watch. The operator keeps a cache of the `state.json` in a Secret, with an internally generated name. This allows the operator to share the state with any pod that it creates.

## Development
Run `make install` whenever the custom resource changes to apply the latest CRD to your cluster. Run `make githooks` to automate this task after every merge.
