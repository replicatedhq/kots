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

Two Main Business-Driver Goals
1. Vendors can create "template" resources in the broadest sense; define resources once and they can be used _N_ times as needed by their application.
1. Vendors can extend resources with _N_ additional configuration properties, like environment variables or volume mounts.

Additional Technical Goals
1. Maintain last mile kustomization of of all resources.

## Non Goals

Vendor requests that were left out of scope of this proposal as future tasking:
1. Having Kotsadm parse file(s) to gather config data, including variadic resources
    * I don't think this was requested but could have been implied by some vendor requests. 
    * Vendors and customers still interact with config fields the same way through the CLI or UI, although there will be options to create dynamic fields.
    * Individual Files can still be base64 encoded and inserted into resources using template functions.
1. Nested Groups 
    * Template or otherwise, are not supported as part of this proposal.
1. Glob File Dropzone Widget
    * What is it: one dropzone that will create config values for a collection of files instead of clicking a "+" sign multiple times
    * This might have been implied by various customer usage cases (I just want to dump some files here and mount them to a container) 
    * I think this is a straightforward implementation following this proposal, so it is not covered for clarity.
1. Dynamic Preflights
    * Because resources can be created dynamically, preflights may be valuable if they could be modified based on the planned size of deployment.
    * This is not included in this scope. Any modification to preflights would need to be submitted as an independent proposal.

## Background

Application configuration values are currently defined by vendors as static fields with basic scalar value types like integer, string and boolean (the file options can be treated as a special case of string). 
All fields must currently be defined ahead of time.

KOTS currently uses resources with the following hierarchy:
1. **Config Spec** - This top-level resource defines the static fields available to configure the application.
It is defined by the vendor.
_A Config Spec w/ values populated is used as request format to change config values._
    1. **Config Group** - Defines collections of config items for navigation and bulk hide/show manipulation.
        1. **Config Item** - The individual scalar fields defined in a group (and optionally their values).
1. **ConfigValues Spec** - This top-level resource is _rendered by kots_ after the configuration is defined by the user into the app upstream archive under /upstream/userdata. 
It is a flat list of field names and values w/o any group mapping.
    1.  **Config Value** - The name, value and default value of each config item.

Examples of these are provided inline in the Detailed Design section of the proposal.

The current configuration pipeline works as follows:
1. Customer passes in config values via CLI or UI
1. ConfigValue spec is saved to the `/userdata` folder along with upstream to the `upstream` directly
1. Kots renders the upstream against the config values and also filters out any unnecessary files (e.g. preflight spec). This goes into the `base` directory along with a kustomize file.
1. Midstream changes are applied.
1. Downstream changes are applied.
1. Completed manifests are sent to the operator to get deployed.

### Target Use Cases:

1. Template Resources example: create new Kafka instance.
    * Customers can click "Add an Kafka" in the Kotsadm console and specify multi copies of configuration items for dynamic resource creation.
    * There will be some MVP validation of templates resources:
        * At least X instances of templated resources
        * Individual config item validation still works
1. Repeat Config Items examples: mounting config files to a container.
    * Customers can click something like a "+" next to an individual field to make it an array of values
    * Vendor can use the values to amend 
1. BOTH
    * Customers can still specify variadic config information using the CLI
    * Last-mile Kustomization still works, or there is a technical path forward.

## High-Level Design 

Supporting the above customer use cases falls into two new feature additions for KOTS:
1. `templateGroups` added to the [Config specification](https://kots.io/reference/v1beta1/config/).
1. `repeatable` attribute added to [Config Items](https://kots.io/reference/v1beta1/config/#items).

Vendors will leverage these new features as part of the Config Spec design, and by using [Golang Text Templating](https://golang.org/pkg/text/template) syntax for repeated elements (`range`) and sub-template definitions (`template`) in their yaml configuration. Not only will this explicitly document dynamically created resources, but it will provide a standardized convention as reference. 

Usage examples provided in the Detailed Design section.

### `reapeatable` Config Items

The purpose of adding a `repeatable` attribute to config items is to add the capability *EXTEND* resources.

The existing Config Item concept will be augmented with a new property `repeatable` to indicated the value will be an array of values rather than a scalar. The value types will still inherit from the `type` field.

To use these array values, a new method `ConfigOptionMap` will be added to the Replicated [Config Context](https://kots.io/reference/template-functions/config-context/) template functions to provide a `pipeline` output that can be used in conjunction with `range` in a Golang Text Tempalate to iterate over values.

### templateGroups

The purpose of adding `templateGroups` in the Config spec is to add the ability to *COPY* collections of resources/config.

Template Groups are similar to the existing Config Groups concept, except this new category identified resources that can be cloned in ConfigValues or in the kotsadm UI. They have addition properties like name prefixes and validation. 

Instead of the relationship of `Groups` having `ConfigItems`, `TemplateConfigGroups` have `TemplateItems`. The distinction is that TemplateItems can have an array of values while Config Items can only have a single value. When rendered into a ConfigValue spec by KOTS all of the values are flatten into a single list, the only difference being the naming convention and a `parent` field to point back to the template group.

YAML files rendered as part of of a `templateGroup` are declated using the Golang notation for declaring nested templates (`{{define <NAME>}} <content> {{end}}`.
Explicit declaration will not only streamline business logic but provide additional debugging context. 

## Detailed Design

While the design considered here is presented in an interleaved fashion, this proposal suggests that work be broken up in the following tasking:
1. Repeatable Config Items
1. templateGroups
1. UI optimizations (Glob File Dropzone)

The first consideration is how the revised API will look to Vendors using these features in their application.

### Example Revised Kotskind Resources 

Vendors will use the revised Config Spec to define templateGroups and repeatable Config Items. 
Values are inserted by the Kots and returned as part of the API for creating the ConfigValues spec.
Below is a representative resource.

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

    # NEW!
    # This is a repeat Config Item
    - name: "static_files"
      type: file
      title: "Static Assets"
      repeatable: true      # Tells the UI/Kots to expect an array
      minimumCount: 3       # Not sure if this is needed here but including for discussion
      repeatValues:         # Returned to the API filled in from the CLI/console    
      - "encoded file value one"
      - "encoded file value two"
    - <more config items here>

  # NEW!
  templateGroups:
  - name: nginx             # Template ID
    title: Proxy Instances  # Group Friendly Name
    groupName: Proxy        # UI Name associated with "Create another -----"
    groupPrefix: nginx      # Label all resources+values created with <prefix>-resource-<cardnality>
    minimumCount: 1         # How many instances need to be created? Populates this many templates in the UI w/ defaults.
    templateItems: 
    - name: "port",
      type: "text", 
      title: "Proxy Port", 
      default: "", 
      values:               # values will get added by the UI
      - value: "80"
        id: "nginx-0-port" 
      - value: "443"
        id: "nginx-1-port"
      - value: "8080"
        id: "nginx-2-port"
  - name: kafka_template    
    title: Kafka Instances  
    groupName: Kafka        
    groupPrefix: kafka      
    minimumCount: 1         
    templateItems: 
    - name: "hostname",
      type: "text", 
      title: "Kafka Hostname", 
      default: "kafka.default.local", 
      values:
      - value: "kafka.one.local"
        id: "kafka-0-hostname" 
      - value: "kafka.two.local"
        id: "kafka-1-hostname"
    # Combining both concepts
    - name: "topic",
      type: "text", 
      title: "Kafka Default Topics", 
      default: "", 
      repeatable: true              # Tells the UI/Kots to expect an array
      minimumCount: 1       
      values:                       # values will get added by the UI
      - value: "topic A"            # Instance 1 has two topic, but instance 2 only has 1
        id: "kafka-0-topic-0" 
      - value: "topic B"
        id: "kafka-0-topic-1"
      - value: "topic C"
        id: "kafka-1-topic-0"
    - <template config items here>
```

#### ConfigValues

The ConfigValues spec is rendered by Kots and stored as part of the applications release archive.
This will still be maintained as a flat list of values regardless of any new constructs.

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

### Resource Templates

In addition to using the new syntax in the KOTS CRDS, they will template their resources using Golang Text Template `range` syntax like the following examples.
Comments are used to keep valid syntax for linting purposes and also provide explicit documentation of generated fields.

#### Repeatable Config Item Usage

```yaml
# deployment.yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: my-deploy
spec:
  template:
    spec:
      containers:
        - name: test
          image: httpd
          volumeMounts:
            - mountPath: /var/www/html
              name: secret-assets
              readOnly: true

      volumes:
      - name: secret-assets
        projected:
          sources:
          # GENERATED CONTENT repl{{ range $name := ConfigOptionMap "static_files" }}
          - secret:
            name: repl{{ $name }} repl{{ end }}
          # END GENERATED 
---
# confingmap.yaml
# GENERATED CONTENT repl{{ range $name, $value := ConfigOptionMap "static_files" }} 
apiVersion: v1
kind: Secret
metadata:
  name: repl{{ $name }}
data:
  # property-like keys; each key maps to a simple value
  file: repl{{ $value }} repl{{ end }}
# END GENERATED
```

#### templateGroup Config Usage

In addition to using the new syntax in the KOTS CRDS, they will template their resources using Golang Text Template `range` syntax like the following examples.
Comments are used to keep valid syntax for linting purposes and also provide explicit documentation of generated fields.

```yaml
# GENERATED CONTENT repl{{ template nginx . }} 
# deployment.yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: my-deploy
spec:
  template:
    spec:
      containers:
        - name: proxy
          image: nginx
          env:
          - name: NGINX_PORT
            value: "{ { repl HasLocalRegistry } }"
---
# service.yaml
apiVersion: v1
kind: Service
metadata:
  name: example-nginx
  labels:
    app: example
    component: nginx
spec:
  type: LoadBalancer
  ports:
  - port: 80
  selector:
    app: example
    component: nginx
# repl{{end}}
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

### API Requests

TBD - THIS IS THE CURRENT FUNCTIONALITY

`POST /api/v1/app/{appSlug}/config`

`GET /api/v1/app/{appSlug}/config/{sequence}`

### Kustomization / File Tree View

TBD

## Design Limitations

* Configmap/secrets can only hold 1MB of data. No way to pass in an arbitrarily large file and have it passed along as configuration.
    * This more than likely eliminates the possibility of storing binary files.

* No ability to bulk-patch resources before they are rendered. Can still use Kustomize targets to accomplish this.

There is an ugly artifact of using comments 


## Testing

Any template rendering based on this design should be refactored in such a way as to allow unit/integration testing of sample manifests against the expected API output. 
The following test cases are relevant:
1. Repeatable Config Items
    1. Happy Path w/ different various item types
    1. Rendering without any defined values
    1. Using ConfigOption and ConfigOptionList with fields that are (not) repeatable.
1. templateGroup Configs
    1. Happy path with templated resources
    1. No matching templated resources
    1. Multiple matching templated resources

Testim tests (both smoke tests and release acceptance tests) will be augmented along with teh QAKots application to test the new UI elements for both features.

At a future point we will need to add a test framework for the CLI (or augment the current acceptance tests) to test that configuration can be passed to kotsadm as part of an unattended install.

## Alternatives Considered

TBD

Just clone files that are repeated.

Implicit collection of resources.

Instead of using patch files to manage kustomization of the base, we could build a [custom generator in Go for kustomize](https://kubectl.docs.kubernetes.io/guides/extending_kustomize/) that takes in arbitrary templates and spits out the results directly.

## Security Considerations

As configuration is already part of the app definition, this proposal doesn't anticipate any changes to security posture.

Because the resources can be generated or extended dynamically, it's expected that the honnus is on the vendor to ensure this doesn't not open any vulnerabilities in their application.

(Thanks to vmware-tanzu/velero for this design template)

## References

Kustomize Resources
1. [Golang Text/Template Package](https://golang.org/pkg/text/template)
1. [Generic Generator Discussion](https://github.com/kubernetes-sigs/kustomize/issues/126)
1. [JSON Path Example](https://github.com/yubessy/example-kustomize-cronjob-multiple-schedule)
