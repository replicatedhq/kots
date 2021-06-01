# NOTES/QUESTIONS

* Where are config values rendered now?
* What to do about base, midstream, downstream
  * What would midstream look like?
* Helm chart support?

# Variadic Config Proposal

One to two sentences that describes the goal of this proposal.
The reader should be able to tell by the title, and the opening paragraph, if this document is relevant to them.

_Note_: The preferred style for design documents is one sentence per line.
*Do not wrap lines*.
This aids in review of the document as changes to a line are not obscured by the reflowing those changes caused.

## Goals

* Customers can click "Add an XXX" in the Kotsadm
* MVP validation of Variadic resources:
  * At least X resources
  * Individual resource validation still works
* Customers can specify variadic config information using the CLI


## Non Goals

Vendor requests that were left out of scope of this proposal as future tasking:
* Having Kotsadm parse file(s) to gather config data, including variadic resources.

## Background

One to two paragraphs of exposition to set the context for this proposal.

## High-Level Design

One to two paragraphs that describe the high level changes that will be made to implement this proposal.

### ProposedConfig Resource 

## Detailed Design

A detailed design describing how the changes to the product should be made.

The names of types, fields, interfaces, and methods should be agreed on here, not debated in code review.
The same applies to changes in CRDs, YAML examples, and so on.

Ideally the changes should be made in sequence so that the work required to implement this design can be done incrementally, possibly in parallel.

### Validation

* Minimum/Maxmimum
* Individual resource validation
* Hidden/Is Enabled?

## Testing

Write a summary of how this enhancement will be tested to ensure there are no regressions in the future.

## Alternatives Considered

If there are alternative high level or detailed designs that were not pursued they should be called out here with a brief explanation of why they were not pursued.

## Security Considerations

If this proposal has an impact to the security of the product, its users, or data stored or transmitted via the product, they must be addressed here.


(Thanks to vmware-tanzu/velero for this design template)
