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

In the kots application spec, modify the `statusInformers` option array type to include an optional `exclude` field with each entry.
Then, config template functions (e.g. ConfigOptionEquals) can be used in the `exclude` field value to indicate when/if to exclude the entry (informer) from the list.

## Detailed Design

First, modify the `statusInformers` option in the kots application spec to be an array of interfaces instead of strings to support both the old and the new format (explanation of new format below).

In the new format, each entry can be either a string (same as old format), or an object that consists of two fields, a string `resource` field, and a boolstring `exclude` field.
The `resource` field is required, but the `exclude` field is optional.

Second, when creating or updating an app version:

1- Parse the kots application spec and read the `statusInformers` array.
2- Check the type for each entry in the array, if not a string, parse in the new format.
3- Parse the `exclude` field as bool and exclude the entry from the array accordingly.
4- Convert the array back to the old format (array of strings) containing the names of the resources.
5- Update the `statusInformers` array in the kots application spec to be saved in the db for this version.

Now, when this version is deployed, the status informers to be executed will not contain the excluded status informers.
