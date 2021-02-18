# Replicated

Kots was built to provide a great experience installing, configuring and updating Replicated apps.

To prepare the latest version of a Replicated app:

```shell
kubectl kots pull replicated://app-slug
```

Optionally, you can specify the version on the URL. This will download the version requested.

```shell
kubectl kots pull replicated://app-slug@v1.2.0
```

For disambiguation when there are multiple releases with the same name, use the sequence number:

```shell
kubectl kots pull replicated://app-slug#12
```

Some application channels require that they are specified on the URL:

```shell
kubectl kots pull replicated://app-slug/channel
```

For local testing, you can point to a directory that contains the extracted YAML:

```shell
kuebctl kots pull replicated://app-slug --local-path=./workdir
```
