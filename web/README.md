# Web
change

This is the Admin Console web site. It's a react site that interacts with the api directory in this repo.

### Building

CI/CD is set up using Buildkite. On commit to master, two new images will be created:

kotsadm/kotsadm-web:alpha and replicated/kotsadm-web:alpha. These are the same, but 2 exceptions: 1) the env/*.js file that's included and the nginx config. Basically the kotsadm/ image is very configurable, and the replicated/ image is pretty static for the multi-tenant version hosted at www.replicated.com.

When a git tag is made, tagged images (not :alpha) are created.

## CI / CD checks

The **build-web** job will fail if unit tests fail or there are formatting or linting issues. Run these command locally and resolve issues to pass CI checks.

### All CI tests

```
yarn test

```

** This will run all of the `test:*` commands concurrently. Recommend using to determine if there is an error. Use the individual commands for debugging since the logging output is easier to read than the output from `concurrently`.

### Unit tests

```
yarn test:unit
```

### Linting and formatting

```
yarn format:fix
```

### Typechecking
```
yarn test:typecheck
```

