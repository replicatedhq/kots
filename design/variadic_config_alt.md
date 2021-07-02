# Variadic Config Proposal

Vendors require the ability to dynamically create resources as part of install configuration.
One common use case is installing operators, where the customers need to create dynamic resources, like instances of an application or service, the number of which is unknown until install time.
They also need to be extend existing resources, like mounting _N_ files to a pod, where _N_ is not known until install time.
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
    * This may not have been requested but could have been implied by some vendor requests. 
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
1. Large Binary File Config Items 
    * Even though this was requested, the details of variadic config are considered a pre-requisite and this would need to be follow-on work.

## Background

Application configuration values are currently defined by vendors as static fields with basic scalar value types like integer, string and boolean (the file options can be treated as a special case of string). 
All fields must currently be defined ahead of time by the vendor.

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
1. Kots renders the upstream against the config values and also filters out any unnecessary files (e.g. preflight spec). This goes into the `base` directory along with a Kustomize file.
1. Midstream changes are applied.
1. Downstream changes are applied.
1. Completed manifests are sent to the operator to get deployed.

### Target Use Cases:

1. Template Resources example: create new Kafka instance.
    * Customers can click "Add a Kafka" in the Kotsadm console and specify multi copies of configuration items for dynamic resource creation.
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
1. "Repeatable Config Items" supported by to the [Config Item Schema](https://kots.io/reference/v1beta1/config/#items).
1. "Repeatable Config Groups" supported by to the [Config Group Schema](https://kots.io/reference/v1beta1/config/#groups).

Vendors will leverage these new features as part of the Config Spec design, and by using [Golang Text Templating](https://golang.org/pkg/text/template) syntax for repeated elements (`range`) in their yaml configuration and a number of new context methods. Not only will this explicitly document dynamically created resources, but it will provide a standardized convention as reference. 

Usage examples provided in the Detailed Design section.

### `reapeatable` Config Items

The purpose of adding a `repeatable` attribute to Config Items is to add the capability *EXTEND* resources.

The existing Config Item concept will be augmented with a new property `repeatable` to indicated the value will be an array of values rather than a scalar. The value types will still inherit from the `type` field.

To use these array values, a new method `ConfigOptionMap` will be added to the Replicated [Config Context](https://kots.io/reference/template-functions/config-context/) template functions to provide a `pipeline` output that can be used in conjunction with `range` in a Golang Text Tempalate to iterate over values.

### `reapeatable` Config Groups

The purpose of adding a `repeatable` attribute to Config Groups is to add the ability to *COPY* collections of resources/config.

The existing Config Group concept will be augmented with a new property `repeatable` to indicated the values in each Config Item will be an array. 
In the case where this item also has the repeatable attribute, values will be flatted into a array with a stride defined by the number of group instances.
Value names will also reflect both the index in the group and the index in the item.

To use these array values, a new method `ConfigGroupList` will be added to the Replicated [Config Context](https://kots.io/reference/template-functions/config-context/) template functions to provide a `pipeline` output that can be used in conjunction with `range` in a Golang Text Tempalate to iterate over values.

## Detailed Design

While the design considered here is presented in an interleaved fashion, this proposal suggests that work be broken up in the following tasking:
1. Repeatable Config Items
1. Repeatable Config Groups

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
    - name: "nginx_port"
      type: "text"
      title: "Nginx port"
      default: "80"
      value: ""

    # NEW!
    # This is a repeatable Config Item
    - name: "static_files"
      type: file
      title: "Static Assets"
      repeatable: true      # NEW! Tells the UI/Kots to expect an array
      minimumCount: 3       # NEW! Not sure if this is needed here but including for discussion
      repeatValues:         # NEW! Returned to the API filled in from the CLI/console
      - "encoded file value one"
      - "encoded file value two"

  # NEW! This is a repeatable group
  - name: nginx             # ID
    title: Proxy Instances  # Group Friendly Name
    repeatable: true        # NEW! Tells the UI/Kots this is a group
    groupName: Proxy        # NEW! UI Name associated with "Create another -----"
    groupPrefix: nginx      # NEW! Label all resources+values created with <prefix>-<cardnality>-resource
    minimumCount: 1         # NEW! How many instances need to be created? Populates this many templates in the UI w/ defaults.
    items: 
    - name: "port"
      type: "text"
      title: "Proxy Port"
      default: "", 
      templates:
      - name: secret-template
        template: envPatch.yaml
      repeatValues:         # values will get added by the UI
      - value: "80"
        id: "nginx-0-port" 
      - value: "443"
        id: "nginx-1-port"
      - value: "8080"
        id: "nginx-2-port"

  # Second Example of Repeatable Config Group
  - name: kafka
    title: Kafka Instances
    repeatable: true
    repeatGroupName: Kafka
    minimumCount: 1
    items: 
    - name: "hostname"
      type: "text"
      title: "Kafka Hostname"
      default: "kafka.default.local"
      repeatValues:
      - value: "kafka.one.local"
        id: "kafka-0-hostname" 
      - value: "kafka.two.local"
        id: "kafka-1-hostname"
    # Combining both concepts
    - name: "topic"
      type: "text" 
      title: "Kafka Default Topics"
      repeatable: true
      default: ""
      minimumCount: 1
      repeatValues:                 # values will get added by the UI
      - value: "topic A"            # Instance 1 has two topics, but instance 2 only has 1
        id: "kafka-0-topic-0" 
      - value: "topic B"
        id: "kafka-0-topic-1"
      - id: "kafka-1-topic-0"       # Assumes the default value; value property is optional
```

#### ConfigValues

The ConfigValues spec is rendered by Kots and stored as part of the applications release archive.
This will still be maintained as a flat list of values regardless of any new constructs.

**NOTE:** @Marc Why can't this be nested? Problem is that unlike statefulsets, order doesn't matter, i.e. what happens when I delete instance #3

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
    <prefix>-0-<name>:
      parent: <templateGroup.Name>    # Not sure if this is needed, but seems like useful information, plus disambiguates from values that accidentally use the same syntax.
      default: <value>
      value: <value> 
    # Example
    kafka-0-hostname:
      parent: kafka
      value: kafka-zero.corp.com
```

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
    <prefix>-0-<name>:
      parent: <templateGroup.Name>    # Not sure if this is needed, but seems like useful information, plus disambiguates from values that accidentally use the same syntax.
      default: <value>
      value: <value> 
    # Example
    kafka-name-0:
      parent: kafka
      groupId: 0
      value: alpha
    kafka-name-1:
      parent: kafka
      groupId: 1
      value: bravo
```

### Resource Templates

In addition to using the new syntax in the KOTS CRDS, they will template their resources using Golang Text Template `range` syntax like the following examples.
Comments are used to keep valid syntax for linting purposes and also provide explicit documentation of generated fields.

#### New ConfigContext Methods

| Method            | Input | Output | Purpose |
|-------------------|-------|--------|---------|
| ConfigOptionList  |       |        |         |
| ConfigOptionIndex |       |        |         |
| ConfigGroupList   |       |        |         |
|                   |       |        |         |
|                   |       |        |         |

#### Repeatable Config Item Usage

Mounting a bunch of secrets (files) to a container as config data.

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
          # repl{{ range $index, $value := ConfigOptionList "static_files" }}
          - secret:
            name: secret-repl{{ $index }} repl{{ end }}
# confingmap.yaml
# GENERATED CONTENT repl{{ range $index, $value := ConfigOptionList "static_files" }} 
---
apiVersion: v1
kind: Secret
metadata:
  name: secret-repl{{ $index }}
data:
  # property-like keys; each key maps to a simple value
  file: repl{{ $value }} repl{{ end }}
# END GENERATED
```

#### Repeatable Config Group Usage

`templateGroups` can leverage Golang Text Templates `template` syntax for defining templates inline in text files. It is possible using the `ParseGlob`, `Templates` and `Name` methods on the `Template` type to bulk parse these templates and associated them by name to the templateGroup name. 
KOTS will then use the array of N values provided to render the N copies of the template as a pre-processing step on the existing render process.
New template context functions will be needed to gather instance-specific template values. 
For this reason, these functions will need to be created for each instance of running the template.

Comments are used to keep valid syntax for linting purposes and also provide explicit documentation of generated fields.

As part of a separate pass or parsing, we could decide to use the standard Golang delimiters. 
We could also decide rather than using the following multi-doc YAML solution with a template named after the `templateGroup`, we could use multiple templates with a common prefix to identify the `templateGroup`.

**NOTE**: Maybe we don't need group.
```yaml
# GENERATED CONTENT repl{{ range $index, $group := ConfigGroupList "nginx"}}
# deployment.yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: nginx-repl{{ $index }} # Returns "nginx-0" for the first instance
  labels:
    app: example
    component: nginx-repl{{ $index }}
spec:
  template:
    spec:
      containers:
        - name: proxy
          image: nginx
          env:
          - name: NGINX_PORT
            value: {{repl ConfigOptionByIndex "nginx-port" $index }}  # Returns 80 for the first instance
---
# service.yaml
apiVersion: v1
kind: Service
metadata:
  name: nginx-repl{{ $index }}
  labels:
    app: example
    component: nginx-repl{{ $index }}
spec:
  type: LoadBalancer
  ports:
  - port: 80
    targetPort: {{repl ConfigOptionIndex "nginx-port" $index }}
  selector:
    app: example
    component: nginx-repl{{ $index }}
---
# END GENERATED CONTENT repl{{end}}
```

#### Combined Usage Example

```yaml
# GENERATED CONTENT repl{{ range $index, $group := ConfigGroupList "kafka"}}
# deployment.yaml
apiVersion: kafka.operator.io
kind: KafaCluster
metadata:
  name: kafka-repl{{ $index }}
  labels:
    app: example
    component: kafka-repl{{ $index }}
spec:
  template:
    spec:
      containers:
        - name: kafka
          image: kafka
          env:
          - name: HOSTNAME
            value: {{repl ConfigOptionTemplate "kafka-hostname" $index }}
          - name: DEFAULT_TOPICS
            value: # how to do a coma delimited list of results here?
---
# END GENERATED CONTENT repl{{end}}
```

### Revised Business Logic Overview

Additions where noted:
1. Customer passes in config values via CLI or UI
1. ConfigValue spec is saved to the `/userdata` folder along with upstream to the `upstream` directly
1. Kots renders the rest of `upstream` against the config values and also filters out any unnecessary files (e.g. preflight spec). 
This goes into the `base` directory along with a kustomize file.
    1. **NEW** New context methods will be applied to provide iteration over `repeatable` config items and groups.
1. Midstream changes are applied.
1. Downstream changes are applied.
1. Completed manifests are sent to the operator to get deployed.

## Design Limitations

1. Configmap/secrets can only hold 1MB of data. No way to pass in an arbitrarily large file and have it passed along as configuration.
    * This more than likely eliminates the possibility of storing binary files, which has been specifically requested.
1. No ability to bulk-patch resources before they are rendered. Can still use Kustomize targets to accomplish this.
1. The syntax is ugly and somewhat verbose. There will comment artifacts left in the base, midstream and downstream YAML files after rendering.

## Testing

Any template rendering based on this design should be refactored in such a way as to allow unit/integration testing of sample manifests against the expected API output. 
The following test cases are relevant:
1. Repeatable Config Items
    1. Happy Path w/ different various item types
    1. Rendering without any defined values
    1. Using ConfigOption and ConfigOptionList with fields that are (not) repeatable.
1. Repeatable Config Groups
    1. Happy path with templated resources
    1. No matching templated resources
    1. Multiple matching templated resources

Testim tests (both smoke tests and release acceptance tests) will be augmented along with teh QAKots application to test the new UI elements for both features.

At a future point we will need to add a test framework for the CLI (or augment the current acceptance tests) to test that configuration can be passed to kotsadm as part of an unattended install.

## Alternatives Considered

Support Full Helm Syntax.

<details>
  <summary>Annotations Approach</summary>
  
    ```yaml
    # deployment.yaml
    apiVersion: apps/v1
    kind: Deployment
    metadata:
    name: my-deploy
    annotation:
        kots.io/repeatableGroup/spec.template.volumes[secret-assets].sources: static_files  
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
            sources: {}
    # confingmap.yaml
    ---
    apiVersion: v1
    kind: Secret
    metadata:
    name: repl{{ $name }}
    labels:
        vendorA: true
    annotation:
        kots.io/repeatableItem: nginx-ports
        kots.io/repeatableItem: nginx-volumes
    data:
    # property-like keys; each key maps to a simple value
    file: repl{{ ConfigOptionValue <ID> }} 
    ```

    ```yaml
    # deployment.yaml
    apiVersion: apps/v1
    kind: Deployment
    metadata:
    name: nginx-repl{{$i}} # Returns "nginx-0" for the first instance
    labels:
        app: example
        component: nginx-repl{{$i}}
    spec:
    template:
        spec:
        containers:
            - name: proxy
            image: nginx
            env:
            - name: NGINX_PORT
                value: {{repl ConfigOptionTemplate "nginx-port" $i }}  # Returns 80 for the first instance
                value: {{repl ConfigOptionTemplate "nginx-port" $i }}  # Returns 80 for the first instance
    ---
    # service.yaml
    apiVersion: v1
    kind: Service
    metadata:
    name: repl{{ ConfigOptionTemplateName "nginx" }} 
    labels:
        app: example
        component: repl{{ ConfigOptionTemplateName "nginx" }}
    annotation:
    kots.io/repeatable: nginx
    spec:
    type: LoadBalancer
    ports:
    - port: 80
        targetPort: {{repl ConfigOptionTemplate "nginx-port" }}
    selector:
        app: example
        component: repl{{ ConfigOptionTemplateName "nginx" }}
    ---
    ```
</details>

<details>
  <summary>Sub Templates Approach</summary>
  
    ```yaml
    # deployment.yaml
    apiVersion: apps/v1
    kind: Deployment
    metadata:
    name: my-deploy
    annotation:
    kots.io/repeatableGroup/spec.template.volumes[secret-assets].sources: static_files
    kots.io/repeatableGroup/spec.template.volumes[secret-assets].sources: static_files
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
            sources: {}
            # END GENERATED 
    # confingmap.yaml
    ---
    apiVersion: v1
    kind: Secret
    metadata:
    name: repl{{ $name }}
    labels:
        vendorA: true
    annotation:
    kots.io/repeatableItem: nginx-ports
    kots.io/repeatableItem: nginx-volumes
    data:
    # property-like keys; each key maps to a simple value
    file: repl{{ ConfigOptionValue <ID> }} 
    ```

    ```yaml
    # GENERATED CONTENT repl{{ $i, $group := range ConfigOptionRepeatableGroup "nginx"}}
    # deployment.yaml
    apiVersion: apps/v1
    kind: Deployment
    metadata:
    name: nginx-repl{{$i}} # Returns "nginx-0" for the first instance
    labels:
        app: example
        component: nginx-repl{{$i}}
    spec:
    template:
        spec:
        containers:
            - name: proxy
            image: nginx
            env:
            - name: NGINX_PORT
                value: {{repl ConfigOptionTemplate "nginx-port" $i }}  # Returns 80 for the first instance
                value: {{repl ConfigOptionTemplate "nginx-port" $i }}  # Returns 80 for the first instance
    ---
    # service.yaml
    apiVersion: v1
    kind: Service
    metadata:
    name: repl{{ ConfigOptionTemplateName "nginx" }} 
    labels:
        app: example
        component: repl{{ ConfigOptionTemplateName "nginx" }}
    annotation:
    kots.io/repeatable: nginx
    spec:
    type: LoadBalancer
    ports:
    - port: 80
        targetPort: {{repl ConfigOptionTemplate "nginx-port" }}
    selector:
        app: example
        component: repl{{ ConfigOptionTemplateName "nginx" }}
    ---
    # END GENERATED CONTENT repl{{end}}
    ```
</details>


Using the Golang Text Template functionality has it's flaws as far as keeping valid YAML syntax and puts a lot of knowledge burden on the vendor. Some alternative considered here were:
    1. Having KOTS Use very basic search/parse capabilities to look for Config Items that were members of a template group, which would implicitly copy any resource using them N times.
    This wasn't proposed because it seemed like it would implement a brute-force search of each file for every possible templateGroup config item.
    1. As a more obscure solution, we could build a [custom generator in Go for kustomize](https://kubectl.docs.kubernetes.io/guides/extending_kustomize/) that takes in arbitrary templates and spits out the results directly. This didn't seem to have too many advantages over using the standard go tooling, but would have require more complexity to manage in KOTS.
    1. Having `ConfigContext` methods that returned valid YAML and/or JSON was also discussed, but this would require passing in templates as arguments for something complicated like rendering a whole configmap.

## Security Considerations

As configuration is already part of the app definition, this proposal doesn't anticipate any changes to security posture.

Because the resources can be generated or extended dynamically, it's expected that the honnus is on the vendor to ensure this doesn't not open any vulnerabilities in their application.

## References

Kustomize Resources
1. [Golang Text/Template Package](https://golang.org/pkg/text/template)
1. [Generic Generator Discussion](https://github.com/kubernetes-sigs/kustomize/issues/126)
1. [JSON Path Example](https://github.com/yubessy/example-kustomize-cronjob-multiple-schedule)

(Thanks to vmware-tanzu/velero for this design template)
