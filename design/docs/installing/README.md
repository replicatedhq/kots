# Installing kots

Kots is designed to be installed and run on your workstation. For a hosted, in-cluster version of kots, check out [kotsadm](https://github.com/replicatedhq/kotsadm).

The recommended installation path is to use [krew](https://krew.dev) on MacOS or Linux. While kots can function as a standalone application, krew's capabilities as a package manager create an easy way to install and upgrade across various platforms. When interacting with a kotsadm installation, kots can also use your kubecontext to find the kotsadm installation.

To install:

```shell
kubectl krew install kots
```

After installing, verify it's working with

```shell
kubectl kots version
```

For additional installation options, visit the [advanced installation options](https://github.com/replicatedhq/kots/blob/main/docs/installing/advanced.md) documentation.

