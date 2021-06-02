# OUSTANDING NOTES/QUESTIONS

* At what stage are config values rendered now?
  - Stored as part of upstream/userdata
  - Also stored in DB as part of `app-version`
* Why /liveconfig?
  - Server-side validation/computation of templates
* What to do about upstream, base, midstream, downstream
  - Upstream: holds the config values under `userdata/`
  - Base: ?
  - Midstream: ?
  - Downstream: ?
* What would midstream look like for these resources?
* Helm chart support?

# Variadic Config Proposal

Vendors require the ability to dynamically create resources as part of install configuration.
A common use case is installing operators, where the customers need to create dynamic resources, like instances of an application, that are unknown until install time.
This proposal outlines a plan to support dynamic/variadic application configuration to facilitate dynamic resource creation. 

## Goals

* Vendors can create "template" resources in the broadest sense; define once and they can be used _N_ times.
* Customers can click "Add an XXX" in the Kotsadm console and specify additional configuration items for dynamic resources.
* There needs to be some MVP validation of resources:
  * At least X resources
  * Individual config item validation still works
* Customers can specify variadic config information using the CLI
* Last-mile kustomization still works, or there is a technical path forward.

## Non Goals

Vendor requests that were left out of scope of this proposal as future tasking:
* Having Kotsadm parse file(s) to gather config data, including variadic resources.

## Background - TBD

One to two paragraphs of exposition to set the context for this proposal.

## High-Level Design -TBD 

One to two paragraphs that describe the high level changes that will be made to implement this proposal.

## Detailed Design

A detailed design describing how the changes to the product should be made.

The names of types, fields, interfaces, and methods should be agreed on here, not debated in code review.
The same applies to changes in CRDs, YAML examples, and so on.

Ideally the changes should be made in sequence so that the work required to implement this design can be done incrementally, possibly in parallel.

### Revised Kotskind Resources

#### Config
```yaml
apiVersion: kots.io/v1beta1 
kind: Config 
metadata: 
  creationTimestamp: null 
  name: config-sample 
spec: 
  # UNCHANGED
  groups: 
  - name: nginx_settings 
    title: Nginx Configs 
    description: Config to serve as an example for creating your own
    items: 
    - name: "nginx_port",
      type: "text", 
      title: "Nginx port", 
      default: "80", 
      value: ""
    - <config items here>
  # NEW!
  templateGroups:
  - name: kafka_template    # Template ID
    title: Kafka Instances  # Group Friendly Name
    groupName: Kafka        # UI Name associated with "Create another -----"
    groupPrefix: Kafka      # Label all resources+values created with <prefix>-resource-<cardnality>
    minimumCount: 1         # How many instances need to be created? Populates this many templates in the UI w/ defaults.
    resources:
        # Do we indicate the resource(s) being templated, or can it be implicit based on usage.
    templateItems: 
    - name: "hostname",
      type: "text", 
      title: "Kafka Hostname", 
      default: "kafka.default.local", 
      values:
      - value: "kafka.one.local"
        id: "kafka-hostname-0" 
      - value: "kafka.two.local"
        id: "kafka-hostname-1"
      - <item values here>
    - <template config items here>
```

#### ConfigValues
```yaml
apiVersion: kots.io/v1beta1 
kind: ConfigValues 
metadata: 
  creationTimestamp: null 
  name: qa-kots 
spec: 
  values: 
    # EXISTING BEHAVIOR
    <name>:
      default: <value>
      value: <value> 
    # Example
    a_templated_text: 
      default: h6IVctWRdVBhflnQkImZybQUUkBjHQuAHj9QWFfBnFEOrf2CqBlkc70F22lMNHug 
      value: GPyocL_6XLb4uCcvPhmoYKtnlWMX3mIHzopzUediHzRs1SenEpmJQi6fJqHDV6MX 
    ...
    # NEW!
    # Values for a template
    <prefix>-<name>-0:
      parent: <templateGroup.Name>    # Not sure if this is needed, but seems like useful information, plus disambiguates from values that accidentally use the same syntax.
      default: <value>
      value: <value> 
    # Example
    kafka-hostname-0:
      parent: <templateGroup.Name>
      value: kafka-zero.corp.com
```

### API Requests

TBD - THIS IS THE CURRENT FUNCTIONALITY

`POST /api/v1/app/{appSlug}/config`

```json
configGroups: [{name: "nginx_settings", title: "Nginx Configs",…},…]
0: {
    items: [{name: "nginx_port", type: "text", title: "Nginx port", default: "80", value: ""}]
    name: "nginx_settings"
    title: "Nginx Configs"
},
1: {name: "example_settings", title: "My Example Config",…}
createNewVersion: false
sequence: 0
```

`GET /api/v1/app/{appSlug}/config/{sequence}`

### Kustomization / File Tree View

TBD

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

## References

Kustomize Resources
1. [Generic Generator Discussion](https://github.com/kubernetes-sigs/kustomize/issues/126)
1. [JSON Path Example](https://github.com/yubessy/example-kustomize-cronjob-multiple-schedule)
