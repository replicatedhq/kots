# Redaction Reports

Support bundles are sanatized with user-provided redactors.
Statistics on what redactors were activated and where should be provided.

## Goals

- Users can see what redactors were activated, and what lines they affected.

## Non Goals

- Users cannot see what, specifically, was redacted.
- If multiple redactors triggered on the same line, users cannot see what redaction is attributable to which redactor.

## Background

Users create custom redactors for support bundles, and it can be difficult to validate that they are functioning.
Redaction reports allow users to see that their redactors are having an effect.

## High-Level Design

Troubleshoot is modified to collect information on what redactions were applied, and where.
This information is then returned when generating a support bundle via the API, or POSTed with the completed bundle if generated from the CLI.
Kots stores this in postgres, and makes it available to the UI via a REST api.

## Detailed Design

A detailed design describing how the changes to the product should be made.

The names of types, fields, interfaces, and methods should be agreed on here, not debated in code review.
The same applies to changes in CRDs, YAML examples, and so on.

Ideally the changes should be made in sequence so that the work required to implement this design can be done incrementally, possibly in parallel.

## Alternatives Considered

If there are alternative high level or detailed designs that were not pursued they should be called out here with a brief explanation of why they were not pursued.

## Security Considerations

