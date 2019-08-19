# Helm

Kots has built-in, full support for installing and configuring Helm charts.

To prepare the latest version of a Helm chart from the "stable" repo:

```shell
kubectl kots pull helm://stable/mysql
```

Optionally, you can specify the Chart version on the URL.

```shell
kubectl kots pull helm://stable/mysql@1.3.0
```

You can pass Helm arguments on the CLI.

```shell
kubectl kots pull helm://stable/mysql --set myusqlPassword=password
```

For charts that are not in the "stable" repo, you can specify the repo name and use the `--repo` flag to provide the repo uri.

```shell
kubectl kots pull helm://elastic/elasticsearch --repo https://helm.elastic.co
```

And you can combine all of these options together, if needed.

```shell
kubectl kots pull helm://elastic/elasticsearch --repo https://helm.elastic.co --set imageTag=7.2.0
```
