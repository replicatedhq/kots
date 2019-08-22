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

To specify the channel to use (the license must be assigned to the requested channel):

```shell
kubectl kots pull replicated://app-slug/channel
```

