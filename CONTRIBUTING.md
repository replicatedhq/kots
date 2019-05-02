# Contributing

Contributions are definitely welcome! We've published documentation on how to set up a local dev environment (or, least one way to do it!) here:

## Local Setup

The recommended configuration to build and run Ship Cluster locally is to run everything on a Docker Desktop environment, with the local, Docker-supplied Kubernetes installation.

Required Software:
- Docker Desktop 2.0.0.2 or later
  - Enable Kubernetes in Docker Desktop
- [Skaffold](https://skaffold.dev) 0.28.0 or later
- [Kustomize](https://kustomize.io) 2.0 or later
- NodeJS 8.x

### Environment

Currently, Ship Cluster requires a GitHub app for authentication. This requirement is being removed, but while it exists, a fully configured GitHub app is required to run Ship Cluster locally. Create a GitHub App following the instructions here.

Configure your local environment with your GitHub app credentials by:

```
export SHIP_CLUSTER_GITHUB_INSTALL_URL=https://github.com/apps/<YOUR-APP-NAME>
export SHIP_CLUSTER_GITHUB_CLIENT_ID=<YOUR-CLIENT-ID>

echo "export SHIP_CLUSTER_GITHUB_INSTALL_URL=${SHIP_CLUSTER_GITHUB_INSTALL_URL}" | tee -a ~/.bash_profile > /dev/null
echo "export SHIP_CLUSTER_GITHUB_CLIENT_ID=${SHIP_CLUSTER_GITHUB_CLIENT_ID}" | tee -a ~/.bash_profile > /dev/null
```

Finally, create 2 Kubernetes secret in the kustomize/overlays/github/secrets directory:

```
kubectl create secret generic github-app \
  --dry-run \
  --from-literal=client-id=<YOUR-CLIENT-ID> \
  --from-literal=client-secret=<YOUR-CLIENT-SECRET> \
  --from-literal=integration-id=<YOUR-INSTALLATION-ID> \
  -o yaml \
  > kustomize/overlays/github/secrets/github-app.yaml

kubectl create secret generic github-app-private-key \
  --dry-run \
  --from-file=private-key.pem=/path/to/github/app/private-key.pem \
  -o yaml \
  > kustomize/overlays/github/secrets/github-private-key.yaml

```

## Running

To start, build and run all server components in the Kubernetes cluster with:

```
skaffold dev --profile github
```

In a seperate terminal window, run the web UI with:

```
cd web
make serve
```
