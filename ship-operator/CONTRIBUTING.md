Contributing
=============

Build & Run the Project
------------------------

### Prerequisites

Ensure you have (min versions to be added):

- `go`
- `docker`
- a kubernetes cluster (docker for desktop is a good development environment)
- a ship application (just the state.json)

### Manager

A single instance of the manager should be running. This can run outside of the cluster by running

```
make run
```

### Controller

The manager will create a new instance of the controller for each custom resource (kind: ShipWatch) deployed. The controller is a command in this project (`ship-operator manager` and `ship-operator controller`). As of now, the controller is required to run as a container image in the cluster. To build:

```
make docker-build
```

## Sample custom resource

A sample custom resource is found in `/kustomize/base/config/samples/ship_v1beta1_shipwatch.yaml`. This is a good sample resource to test with and validate an environment. Before deploying this custom resource, you should create a secret that contains 3 keys:  state.json, key.pem and github.token.

```
---
apiVersion: v1
kind: Secret
metadata:
  name: sample-app-state
type: Opaque
data:
  state.json: <base64 encoded replicated ship state.json file>
  key.pem: <base64 encoded private key that has write access to the repo specified in the custom resource>
  github.token: <base64 encoded github personal access token that can create a pull request to the repo specified>
```

