# Conditional Status Informers

Currently, KOTS does not support having conditional application status informers.
This proposal makes it possible to have a status informer disappear/appear if a specific config option is set to a certain value.

## Goals

- Add the ability to exclude specific status informers based on the applications's config values

## Non Goals

- Make the application status informers support additional kinds/objects.

## Background

For example, many vendors have an embedded vs. external database option.
They have a status informer set up on the database object, but this only works when using the embedded option.
Many of them have requested the ability to have conditional status informers so that the status informer can be excluded when selecting the external db option.

## High-Level Design

Add the ability to render/template the `statusInformers` array in the kots application spec (`kind: Application`).
This way, config template functions (e.g. ConfigOptionEquals) can be used to decide whether or not an informer should be included/applied.

## Detailed Design

## Notes:
- Getting the list of status informers happens during the app deploy loop in TypeScript.
- The list of informers is then sent through a socket to the operator to be applied.

1- To add the ability to exclude specific status informers, the `statusInformers` array will be rendered before it's sent to the operator.
2- This way, conditional template functions can be used as an entry in the `statusInformers` array in the kots application spec.
3- The template function will then resolve to either a valid status informer entry, or not (an empty string `""` for example).
4- Invalid status informers (entries) will be excluded from the array.
