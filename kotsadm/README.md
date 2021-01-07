# Kotsadm

Kotsadm is an installable admin console for managing Kubernetes Off-The-Shelf (KOTS) software. Kotsadm provides a nextgen admin experience for any KOTS application, designed to meet the needs of a wide spectrum of enterprise IT admins, from a “click-to-deploy” model to “automated operations”.

Once deployed, Kotsadm gives administrators the ability to get an application configured, [installed](https://kots.io/kotsadm/installing/installing-a-kots-app/) and [updated](https://kots.io/kotsadm/updating/updating-kots-apps/) using step-through configuration, and automated preflight checks. 

For advanced cluster operators, we recommend setting up the [GitOps](https://kots.io/kotsadm/gitops/single-app-workflows/) & [internal registry](https://kots.io/kotsadm/registries/self-hosted-registry/) integrations to move away from click-to-deploy and instead leverage an Kotsadm's automated operations.

## Distributing a KOTS application
Software vendors can [package their Kubernetes applications](https://kots.io/vendor/) or [Helm charts](https://kots.io/vendor/helm/using-helm-charts) as a KOTS application in order to distribute the application to cluster operators. Kotsadm can serve as the whitelabeled (or multi-app), in-cluster admin console for any KOTS application.

## Kots

Kots CLI is a kubectl plugin to help manage Kubernetes Off-The-Shelf software. See the documentation for more information on [Kots CLI](https://kots.io/kots-cli/getting-started/).

## Installing

For more information on installing Kotsadm locally see https://kots.io/kotsadm/installing/installing-a-kots-app/.

## Contributing

For docs on setting up a dev environment to build and run kotsadm locally, read the [CONTRIBUTING](https://github.com/replicatedhq/kots/blob/master/kotsadm/CONTRIBUTING.md) guide.

# Community

For questions about using KOTS & Kotsadm, there's a [Replicated Community](https://help.replicated.com/community) forum, and a [#kots channel in Kubernetes Slack](https://kubernetes.slack.com/channels/kots).
