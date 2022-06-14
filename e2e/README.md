# E2E Tests

E2E tests are run in build-test workflow on pull_request event.

## Development environment

To install dependencies run:

```bash
make kots
make -C e2e deps
npm install -g @testim/testim-cli
```

The entire suite can be run with the command:

```bash
make e2e
```

To run an individual test run:

```bash
make e2e \
    FOCUS="Change License"
```

To build and run with ttl.sh images run:

```bash
make all-ttl.sh
make e2e \
    KOTSADM_IMAGE_REGISTRY=ttl.sh \
    KOTSADM_IMAGE_NAMESPACE=$USER \
    KOTSADM_IMAGE_TAG=12h
```

To run against the okteto dev environment run:

*Note when using an existing cluster you must focus the suite on a single test*

```bash
okteto context use https://replicated.okteto.dev
make e2e \
    FOCUS="Change License" \
    EXISTING_KUBECONFIG=${KUBECONFIG:-$HOME/.kube/config}
```

To skip cluster teardown in order to debug issues:

*Note the namespace may be specific to the test*

```bash
$ make e2e \
    FOCUS="Change License" \
    SKIP_TEARDOWN=1
...
    To set kubecontext run:
      export KUBECONFIG="$(k3d kubeconfig merge kots-e2e3629427925)"
    To delete cluster run:
      k3d cluster delete kots-e2e3629427925
$ export KUBECONFIG="$(k3d kubeconfig merge kots-e2e3629427925)"
$ kubectl -n smoke-test port-forward svc/kotsm 3000 --address=0.0.0.0
Forwarding from 0.0.0.0:3000 -> 3000
```

### Requirements

1. [Docker](https://docs.docker.com/get-docker/)
