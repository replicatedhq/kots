# Helm V3 Support

KOTS currently supports Helm V2 (apiVersion v1) Charts.
This proposal is to add support for Helm V3 (apiVersion v2) Charts.

## Goals

- Support for Helm apiVersion v2 Charts

## Non Goals

- Any additional Helm functionality outside of the current pull and render workflows

## Background

Helm V3 has been stable since Nov 13, 2019.
Some Charts are using apiVerion v2 and additional Helm V3 Chart functionality.
KOTS does not currently support Helm apiVersion v2 Charts.

## High-Level Design

A new property `spec.helmVersion` will be added to the `kots.io/beta` `HelmChart` spec which can be used
to toggle the Helm version used to pull and render the Helm Chart.
This property will default to `v2` and can optionally be set to `v3`.

## Detailed Design

A new property `spec.helmVersion` will be added to the `kots.io/beta` `HelmChart` spec which can be used
to toggle the Helm version used to pull and render the Helm Chart.
This property will default to `v2` and can optionally be set to `v3`.
When unset or set to `v2` KOTS will use the `helm.sh/helm/v2` Go library to pull and render the Helm Chart.
The existing code path should remain the same.
When set to `v3` KOTS will use the `helm.sh/helm/v3` Go library to pull and render the Helm Chart.
When KOTS encounters an apiVersion v2 chart it will automatically use Helm V3 libraries to pull and render the chart.
