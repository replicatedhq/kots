# YAML File Errors

When the output of a rendered file results in invalid YAML, the resource is not included in base and not deployed.
Additionally, the file name and error is included in the kots.io/v1beta1.Installer yaml (see the example below).
The error often references a line number in the rendered content which is inaccessible to the user.
This proposal attempts to make the rendered output available to the user so that it is easier to debug the issue.

## Goals

- Surface the rendered YAML content to the user when unmarshalling fails

## Non Goals

- Attempt to automate fixing any common errors such as unescaped quotes or line breaks
- Any additional UI work. This will just focus on surfacing information in the release

## Design

Add a new directory at the root of the file tree `yamlErrors` that will include the raw content of all rendered output for yaml files that failed unmarshalling.

An example installation.yaml file:

```yaml
apiVersion: kots.io/v1beta1
kind: Installation
metadata:
  name: sentry-enterprise
spec:
  channelName: Stable
  knownImages:
  - image: docker.io/sentry-enterprise/app:1.0
    isPrivate: true
  releaseNotes: |
    Release 1.0.3
  updateCursor: "3"
  versionLabel: 1.0.3
  yamlErrors:
  - error: 'yaml: line 26: did not find expected key'
    path: deployment.yaml
```

An example file tree:

```
> upstream
> base
> overlays
v yamlErrors
  _index.yaml
  deployment.yaml
```

yamlErrors/\_index.yaml:

```yaml
yamlErrors:
- error: 'yaml: line 26: did not find expected key'
  path: deployment.yaml
```
