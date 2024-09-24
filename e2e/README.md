# E2E Tests

E2E tests are run in build-test workflow on pull_request event.

e2e_test.go uses Ginkgo to build a test suite from inventory.go and runs each test using testim/client.go or playwright/playwright.go

Tests are parallelized using Gingko's test focus. Each workflow definition in .github/workflows/build-test.yaml must define a `test-focus` parameter that matches the `Test Name` property defined in inventory.go. Each e2e test workflow skips all tests but what is defined in `test-focus`. Please colocate new Playwright tests under the Playwright comment with the other pw tests.

New tests should be written with Playwright.

## Adding a new test

Playwright is the preferred testing framework for new tests moving forward. See playwright's [documentation](https://playwright.dev/docs/intro) for more information.

Install Playwright dependencies:

```bash
cd e2e/playwright
npm ci 
npx playwright install --with-deps
```

Install the Playwright extension in VSCode if you've not already done so:

```bash
code --install-extension ms-playwright.playwright
```

To add a new test that you've already added in the [kots-tests-app repo](https://github.com/replicatedhq/kots-test-apps) - do the following:

- Update `.github/workflows/build-test.yaml` to include the new test. You can copy an existing pw entry like `validate-change-channel` and update the test-focus, kots-namespace, and any other parameters needed for the test.
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

## Running tests

For testIM tests, set the testim access token:
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

To build and run with ttl.sh images:

```bash
make all-ttl.sh
make e2e \
    KOTSADM_IMAGE_REGISTRY=ttl.sh \
    KOTSADM_IMAGE_NAMESPACE=$USER \
    KOTSADM_IMAGE_TAG=24h
```

To run using a specific testIM branch:
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
