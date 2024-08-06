# E2E Tests

E2E tests are run in build-test workflow on pull_request event.

e2e_test.go uses Ginkgo to build a test suite from inventory.go and runs each test using testim/client.go or playwright/playwright.go

Tests are parallelized using Gingko's test focus. Each workflow definition in .github/workflows/build-test.yaml must define a `test-focus` parameter that matches the `Test Name` property defined in inventory.go. Each e2e test workflow skips all tests but what is defined in `test-focus`. Please colocate new playwright tests under the playwright comment with the other pw tests.

New tests should be written with playwright.

## Playwright

Playwright is the preferred testing framework for new tests moving forward. See playwright's [documentation](https://playwright.dev/docs/intro) for more information.

### Development environment

To install dependencies run:

```bash
cd e2e/playwright
npm ci 
npx playwright install --with-deps
```

Install the playwright extension in vscode if you've not already done so:

```bash
code --install-extension ms-playwright.playwright
```

### Adding a new test

To add a new test that you've already added in the kots-tests-app repo - do the following:

- Update `.github/workflows/build-test.yaml` to include the new test:

```
validate-change-channel:
    runs-on: ubuntu-20.04
    needs: [ enable-tests, can-run-ci, build-kots, build-kotsadm, build-e2e, build-kurl-proxy, build-migrations, push-minio, push-rqlite ]
    strategy:
      fail-fast: false
      matrix:
        cluster: [
          {distribution: kind, version: v1.28.0}
        ]
    steps:
      - name: Checkout
        uses: actions/checkout@v4
      - name: download e2e deps
        uses: actions/download-artifact@v4
        with:
          name: e2e
          path: e2e/bin/
      - run: docker load -i e2e/bin/e2e-deps.tar
      - run: chmod +x e2e/bin/*
      - name: download kots binary
        uses: actions/download-artifact@v4
        with:
          name: kots
          path: bin/
      - run: chmod +x bin/*
      - uses: ./.github/actions/kots-e2e
        with:
          test-focus: 'Change Channel'
          kots-namespace: 'change-channel'
          k8s-distribution: ${{ matrix.cluster.distribution }}
          k8s-version: ${{ matrix.cluster.version }}
          kots-dockerhub-username: '${{ secrets.E2E_DOCKERHUB_USERNAME }}'
          kots-dockerhub-password: '${{ secrets.E2E_DOCKERHUB_PASSWORD }}'
          // add any other parameters needed for the test
```

- Add the test to `e2e/inventory.go` , making sure the naming matches your kots-test-app and conforms to the naming convention of the other tests in the file:

```go
func NewChangeChannel() Test {
	return Test{
		ID:          "change-channel",
		Name:        "Change Channel",
		Namespace:   "change-channel",
		AppSlug:     "change-channel",
		UpstreamURI: "change-channel/automated",
	}
}
```

- Add a new inventory test entry to `e2e/e2e_test.go` to ensure it actually runs:

```go
Entry(nil, inventory.NewChangeChannel()),
```

- Create a new test directory in `e2e/playwright/tests` matching your test ID, with the corresponding test file:

```
$ tree e2e/playwright/tests/change-channel
e2e/playwright/tests/change-channel 
├── license.yaml  // a test specific license if needed
└── test.spec.ts  // the actual test file
```

- See `e2e/playwright/tests/shared` for test utility functions that can be used in your test for things like logging in or uploading a license.

## testim

Testim is our legacy testing framework. It is being phased out in favor of playwright.

### Development environment

To install dependencies run:

```bash
make kots
make -C e2e deps
npm install -g @testim/testim-cli
```

Set the testim access token:
```bash
export TESTIM_ACCESS_TOKEN=<my-testim-access-token>
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
    KOTSADM_IMAGE_TAG=24h
```

To run using a specific testim branch:
```bash
make e2e \
    TESTIM_BRANCH=$BRANCH_NAME
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
$ kubectl -n smoke-test port-forward svc/kotsadm 3000 --address=0.0.0.0
Forwarding from 0.0.0.0:3000 -> 3000
```

#### Requirements

*Currently, the admin console helm chart will not install on an M1 Macbook because of it's [node affinity](https://github.com/replicatedhq/kots-helm/blob/main/templates/kotsadm-deployment.yaml#L32-L35) rules*

1. [Docker](https://docs.docker.com/get-docker/)
