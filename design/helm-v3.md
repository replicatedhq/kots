# Helm V3 Support

KOTS currently pulls and renders Charts using Helm V2.
This proposal is to add support for Helm V3.

## Goals

- Render Helm Charts using Helm V3

## Non Goals

- Any additional Helm functionality outside of the current pull and render workflows

## Background

Helm V3 has been stable since Nov 13, 2019.
Some Charts are using apiVerion v2 and additional Helm V3 Chart functionality.
KOTS does not currently use Helm V3 to render Charts.

## High-Level Design

A new property `spec.helmVersion` will be added to the `kots.io/beta` `HelmChart` spec which can be used to toggle the Helm version used to pull and render the Helm Chart.
This property will default to `v2` and can optionally be set to `v3`.

As Helm V3 is backwards compatible with V2, once there is evidence of success in various environments including real world scenarios, it is the intention to change this property to default to `v3`.

## Detailed Design

A new property `spec.helmVersion` will be added to the `kots.io/beta` `HelmChart` spec which can be used
to toggle the Helm version used to pull and render the Helm Chart.
This property will default to `v2` and can optionally be set to `v3`.
When unset or set to `v2` KOTS will use the `helm.sh/helm/v2` Go library to pull and render the Helm Chart.
The existing code path should remain the same.
When set to `v3` KOTS will use the `helm.sh/helm/v3` Go library to pull and render the Helm Chart.
