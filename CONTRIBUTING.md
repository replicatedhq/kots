# Contributing

Contributions are definitely welcome! We've published documentation on how to set up a local dev environment (or, least one way to do it!) here:

## Local Setup

The recommended configuration to build and run Ship Cluster locally is to run everything on a Docker Desktop environment, with the local, Docker-supplied Kubernetes installation.

Required Software:
- [Skaffold](https://skaffold.dev) 0.29.0 or later
- [Kustomize](https://kustomize.io) 2.0 or later
- NodeJS 10.x with yarn
- Kubernetes (Recommended to run microk8s)

### Environment

Currently, Ship Cluster requires a GitHub app for authentication. This requirement is being removed, but while it exists, a fully configured GitHub app is required to run Ship Cluster locally. Create a GitHub App following the instructions [here](https://github.com/replicatedhq/ship-cluster/blob/master/docs/developer/github-app.md). After creating the app, you'll have to copy some of the settings locally:

Configure your local environment with your GitHub app credentials by adding them to these lines and exporting them locally.

```
export SHIP_CLUSTER_GITHUB_INSTALL_URL=
export SHIP_CLUSTER_GITHUB_CLIENT_ID=
export SHIP_CLUSTER_GITHUB_CLIENT_SECRET=
export SHIP_CLUSTER_GITHUB_INSTALLATION_ID=
export SHIP_CLUSTER_GITHUB_PRIVATE_KEY_PATH=
```

To configure your local environment, copy/paste the following scripts, once the above variables have been set. This will persist these changes across reboots.

```
echo "export SHIP_CLUSTER_GITHUB_CLIENT_ID=${SHIP_CLUSTER_GITHUB_CLIENT_ID}" \
  | tee -a ~/.profile > /dev/null
echo "export SHIP_CLUSTER_GITHUB_CLIENT_SECRET=${SHIP_CLUSTER_GITHUB_CLIENT_SECRET}" \
  | tee -a ~/.profile > /dev/null
echo "export SHIP_CLUSTER_GITHUB_INSTALLATION_ID=${SHIP_CLUSTER_GITHUB_INSTALLATION_ID}" \
  | tee -a ~/.profile > /dev/null
echo "export SHIP_CLUSTER_GITHUB_INSTALL_URL=${SHIP_CLUSTER_GITHUB_INSTALL_URL}" \
  | tee -a ~/.profile > /dev/null

mkdir -p kustomize/overlays/github/secrets
kubectl create secret generic github-app \
  --dry-run \
  --from-literal=client-id=${SHIP_CLUSTER_GITHUB_CLIENT_ID} \
  --from-literal=client-secret=${SHIP_CLUSTER_GITHUB_CLIENT_SECRET} \
  --from-literal=integration-id=${SHIP_CLUSTER_GITHUB_INSTALLATION_ID} \
  --from-literal=install-url=${SHIP_CLUSTER_GITHUB_INSTALL_URL} \
  -o yaml \
  > kustomize/overlays/github/secrets/github-app.yaml

kubectl create secret generic github-app-private-key \
  --dry-run \
  --from-file=private-key.pem=${SHIP_CLUSTER_GITHUB_PRIVATE_KEY_PATH} \
  -o yaml \
  > kustomize/overlays/github/secrets/github-private-key.yaml

```

## Running

To start, build and run all server components in the Kubernetes cluster with:

```
skaffold dev
```

