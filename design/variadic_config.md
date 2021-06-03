# OUSTANDING NOTES/QUESTIONS

Missing Pieces:
1. How to organize the file tree for these resources
1. How to make sure any template functions are valid Yaml.

Concerns Not Addressed:
`We don’t even know all the file upload combinations that may pop up and so we can’t really templatize every possible function. Even if we could the amount of possible combinations in the UI would be crazy.`
-> *Dan's comment*: sounds like they are really looking for assisted editing capabilities here.

Open Questions:
1. At what stage are config values rendered now?
  - Stored as part of upstream/userdata
  - Also stored in DB as part of `app-version`
1. Why /liveconfig?
  - Server-side validation/computation of templates
1. How to take something like a files and convert them into a config map
  - . Use YAML Multi-doc with repeat config items?
1. What to do about upstream, base, midstream, downstream
  - Upstream: holds the config values under `userdata/`
  - Base: ?
  - Midstream: ?
  - Downstream: ?
1. What would midstream look like for these resources?
1. Helm chart support?

# Variadic Config Proposal

Vendors require the ability to dynamically create resources as part of install configuration.
A common use case is installing operators, where the customers need to create dynamic resources, like instances of an application, that are unknown until install time.
They also need to be extend existing resources, like mounting _N_ files to a pod, where _N_ is not known at install time.
This proposal outlines a plan to support dynamic/variadic application configuration to facilitate dynamic resource creation. 

## Goals

Two Main Goals
1. Vendors can create "template" resources in the broadest sense; define once and they can be used _N_ times.
1. Vendors can extend resources with _N_ additional configuration properties, like environment variables or volume mounts.

## Non Goals

Vendor requests that were left out of scope of this proposal as future tasking:
* Having Kotsadm parse file(s) to gather config data, including variadic resources - this doesn't seem to be needed immediately by any customer. Files will just be base64 encoded and inserted using template functions.
* Nested Groups - template or otherwise, are not supported.
* Repeat File Dropzone: One dropzone that will create a repeated config values instead of clicking a "+" sign multiple times - I think this is a straightforward implementation following this proposal, so it is not covered for clarity.

## Background - TBD

One to two paragraphs of exposition to set the context for this proposal.

### Use Cases:

1. Template Resources example: create new Kafaka instance.
    * Customers can click "Add an Kafka" in the Kotsadm console and specify multi copies of configuration items for dynamic resource creation.
    * There will be some MVP validation of templates resources:
        * At least X instances of templated resources
        * Individual config item validation still works
1. Repeat Config Items examples: mounting config files to a container.
    * Customers can click something like a "+" next to an individual field to make it an array of values
    * Vendor can use the values to amend 
1. BOTH
    * Customers can still specify variadic config information using the CLI
    * Last-mile kustomization still works, or there is a technical path forward.

## High-Level Design 

Supporting the above customer use cases falls into two new feature additions for KOTS:
1. `templateGroups`
1. `repeatable` Config Items

### templateGroups

Template Groups are similar to the existing Config Groups concept, except this new category identified resources that can be cloned in ConfigValues or in the kotsadm UI. Instead of the relationshipt of groups->ConfigItems, 

Kotsadm will take the templateGroup values and implicitly copy and render new resources

### reape



## Detailed Design

A detailed design describing how the changes to the product should be made.

The names of types, fields, interfaces, and methods should be agreed on here, not debated in code review.
The same applies to changes in CRDs, YAML examples, and so on.

Ideally the changes should be made in sequence so that the work required to implement this design can be done incrementally, possibly in parallel.

### Repeat Config Item Manifest

```yaml
piVersion: apps/v1
kind: Deployment
metadata:
  name: my-deploy
  labels:
    kots.io/kotsadm: 'true'
    kots.io/backup: velero
spec:
  template:
    metadata:
      labels:
        kots.io/kotsadm: 'true'
        kots.io/backup: velero
      annotations:
        backup.velero.io/backup-volumes: backup
    spec:
      containers:
        - name: test
          volumeMounts:
            - mountPath: /backup
              name: backup
            - name: kubelet-client-cert
              mountPath: /etc/kubernetes/pki/kubelet
          env:
            repl {{  }}
            - name: DEX_PGPASSWORD
              valueFrom:
                secretKeyRef:
                  key: PGPASSWORD
                  name: kotsadm-dex-postgres
            - name: KOTSADM_LOG_LEVEL
              value: "debug"
            - name: DISABLE_SPA_SERVING
              value: "1"
            - name: KOTSADM_TARGET_NAMESPACE
              value: "test"
              valueFrom: ~

```

### App Archive

Rendered content in the app archive would look like the following. Note the files types are not mutually exclusive and could be overlapping. They are only for illustrative purposes.

* Upstream
    * Normal manifest files
    * Manifest files that utilize config options from a repeat item 
    * Manifest files that utilize config options from a template group
* Base
    * Rendered manifest files (including those with repeat config items)
    * Copy of manifest files 
* Overlays
    * 


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
      values:                       # values will get added by the UI
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

## Design Limitations

* Size of data in configmap/secrets can only hold so much data. No way to pass in an arbitrarily large file and have it passed as configuration.

## Testing

Write a summary of how this enhancement will be tested to ensure there are no regressions in the future.

## Alternatives Considered

Just clone files that are repeated.

Instead of using patch files to manage kustomization of the base, we could build a [custom generator in Go for kustomize](https://kubectl.docs.kubernetes.io/guides/extending_kustomize/) that takes in arbitrary templates and spits out the results directly.

## Security Considerations

If this proposal has an impact to the security of the product, its users, or data stored or transmitted via the product, they must be addressed here.


(Thanks to vmware-tanzu/velero for this design template)

## References

Kustomize Resources
1. [Generic Generator Discussion](https://github.com/kubernetes-sigs/kustomize/issues/126)
1. [JSON Path Example](https://github.com/yubessy/example-kustomize-cronjob-multiple-schedule)
