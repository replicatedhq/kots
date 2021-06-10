# Variadic Config Proposal

Vendors require the ability to dynamically create resources as part of install configuration.
One common use case is installing operators, where the customers need to create dynamic resources, like instances of an application or service, that are unknown until install time.
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

Vendors will leverage these new features as part of the Config Spec design. For these repeatable elements, the vendor will define either a target resource (e.g. a particular deployment file) that will need to be copied for each element of the array or a YAML path that will be cloned for the identified resource. This declarative approach was inspired by Kustomize patches. 

One note is that arrays are not used to store the Config values in any spec.
Using named keys rather that YAML arrays is intentional so that when an element is removed from an array, we can disambiguate whether an item was removed from all of the subsequent elements of the array being modified.

Usage examples provided in the Detailed Design section.

### `reapeatable` Config Items

The purpose of adding a `repeatable` attribute to Config Items is to add the capability *EXTEND* resources.

The existing Config Item concept will be augmented with a new property `repeatable` to indicated the value will be an array of values rather than a scalar. The value types will still inherit from the `type` field.

Config Items will also now include a `template` property to allow specifying the YAML document or sub-document to copy for this array of values.  

### `reapeatable` Config Groups

The purpose of adding a `repeatable` attribute to Config Groups is to add the ability to *COPY* collections of resources/config.

The existing Config Group concept will be augmented with a new property `repeatable` to indicated the values in each Config Item will be an array. 

Config Groups will also now include a `template` property to allow specifying the YAML document or sub-document to clone for each of these groups.  

## Detailed Design

While the design considered here is presented in an interleaved fashion, this proposal suggests that work be broken up in the following tasking:
1. Repeatable Config Items
1. Repeatable Config Groups

The first consideration is how the revised API will look to Vendors using these features in their application.

### Example Revised Kotskind Resources 

Vendors will use the revised Config Spec to define repeatable Config Groups and Items. 
Values are inserted by the Kots and returned as part of the API for creating the ConfigValues spec.
Below is a representative resource that will be used for several examples.

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
      minimumCount: 3       # NEW! Validation 
      templates:            # NEW! This desclares how the array of values will be used
        - apiVersion: v1    # By targeting a YAML path, we can clone this node for each element of the item.
          kind: Deployment
          name: my-deploy
          yamlPath: spec.template.spec.volumes[0].projected.sources
        - apiVersion: v1    # By targeting a resource file, we can clone the whole file for each element.
          kind: Secret
          prefix: secret-
      valuesByGroup:        # NEW! Returned to the API filled in from the CLI/console
        nginx_settings:
          static-file-<short guid>: "encoded file value one"
          static-file-<short guid>: "encoded file value two"
          static-file-<short guid>: "encoded file value three"

  # NEW! This is a repeatable Config Group
  - name: nginx             # ID
    title: Proxy Instances  # Group Friendly Name
    repeatable: true        # NEW! Tells the UI/Kots this is a group
    minimumCount: 1         # NEW! How many instances need to be created? Populates this many templates in the UI w/ defaults.
    instanceNames:             # NEW! Declares each instance of a group. The UI/CLI can generate this information.
    - nginx-<short guid 1>
    - nginx-<short guid 2>
    - nginx-<short guid 3>
    - nginx-<short guid 4>
    items: 
    - name: "port"
      type: "text"
      title: "Proxy Port"
      default: "", 
      templates:
      - apiVersion: v1    
        kind: Deployment
        prefix: proxy-
      - apiVersion: v1    
        kind: Service
        prefix: proxy-
      valuesByGroup:
        nginx-<short guid 1>:
            port-<short guid>: 80
        nginx-<short guid 2>:
            port-<short guid>: 443
        nginx-<short guid 3>:
            port-<short guid>: 8080
        nginx-<short guid 4>:
            port-<short guid>: 3000

  # Second Example of Repeatable Config Group
  - name: kafka
    title: Kafka Clusters
    repeatable: true
    repeatGroupName: Kafka Cluster
    minimumCount: 1
    templates:
    - apiVersion: kafka.banzaicloud.io/v1alpha1     # Specifies the resources that will be COPIED as part of the group
      kind: KafkaCluster
      prefix: kafka-cluster-
    # The names for each group name can be anything unique. They can be generated by the UI or made-up by the user
    instanceNames:             # NEW! Declares each instance of a group. The UI/CLI can generate this information.
    - kafka-<short guid 1>
    - kafka-<short guid 2>
    items: 
    - name: "name"
      type: "text"
      title: "Kafka Cluster Name"
      default: ""
      valuesByGroup:   
        kafka-<short guid 1>:     
          name: alpha
        kafka-<short guid 2>: 
          name: bravo
    # Combining both concepts
    - name: "brokers"
      type: "text"
      title: "Kafka Broker IDs"
      repeatName: Broker
      repeatable: true
      default: ""
      templates:
      - apiVersion: v1
        kind: kafka.banzaicloud.io/v1alpha1
        name: KafkaTopic
        yamlPath: spec.brokers[0]
      valuesByGroup:
        kafka-<short guid 1>: 
          broker-<short guid>: "broker A"
          broker-<short guid>: "broker B"
        kafka-<short guid 2>: 
          broker-<short guid>: "broker 1"
          broker-<short guid>: "broker 2"
          broker-<short guid>: "broker 3"
    - name: "topics"
      type: "text" 
      title: "Kafka Topics"
      repeatable: true
      repeatName: Topic
      default: "myTopic"
      minimumCount: 1
      templates:
      - apiVersion: kafka.banzaicloud.io/v1alpha1
        kind: KafkaTopic
        prefix: kafka-
      valuesByGroup:
        kafka-<short guid 1>: 
            topic-<short guid>: "topicA"
            topic-<short guid>: "topicB"
        kafka-<short guid 2>: 
            topic-<short guid>: "mytTopic" 
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
    <config-item-unique-name>:
      parent: <Group Unique Name>    # Nice to have?
      default: <value>
      value: <value> 
    # Example
    port-<short guid>:
      parent: nginx-<short guid 1>
      value: 80
    port-<short guid>:
      parent: nginx-<short guid 2>
      value: 443
    port-<short guid>:
      parent: nginx-<short guid 3>
      value: 8080
    port-<short guid>:
      parent: nginx-<short guid 4>
      value: 3000
```

### Resource Templates

None of the existing Replicated ConfigContext methods change for consumers, with the correct array value being selected by KOTS automatically when making copies of YAML documents or sub-documents. 

There are a couple new methods added.

| Method                | Input                     | Output                       | Purpose                                     |
|-----------------------|---------------------------|------------------------------|---------------------------------------------|
| RepeatableConfigGroupName | Config Item Name (string) | Group Instance Name (string) | Grab the Group Instance Name for this value |
| RepeatableConfigOptionName | Config Item Name (string) | Config Item Unique Name (string) | Grab the Unique Name for this Config Item element |


#### Repeatable Config Item Usage

Mounting a bunch of secrets (files) to a container as config data.
The expected output is one deployment that references three dynamically created secrets.

Note that these files were specific in the above Config Spec as applying to the Repeatable Config Item `static_files`.

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
          env:
          volumeMounts:
            - mountPath: /var/www/html
              name: secret-assets
              readOnly: true
      volumes:
      - name: secret-assets
        projected:
          sources:
            - secret:
              name: repl{{ RepeatableConfigOptionName "static_files" }}
```
```yaml
# secret.yaml
apiVersion: v1
kind: Secret
metadata:
  name: repl{{ RepeatableConfigOptionName "static_files" }}
data:
  file: repl{{ ConfigOption "static_files" }}
```

#### Repeatable Config Group Usage

Repeatable Config Groups are designed to work much the same way with resource targeting.

The expected output based on the Config Spec provided above would be 4 matching deployments and services.

```yaml
# deployment.yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: proxy-repl{{ $RepeatableConfigGroupName "port" }}
  labels:
    app: example
    component: repl{{ $RepeatableConfigGroupName "port" }}
spec:
  template:
    spec:
      containers:
        - name: proxy
          image: nginx
          env:
          - name: NGINX_PORT
            value: {{repl ConfigOption "port" }}  # Returns 80 for the first instance
```
```yaml
# service.yaml
apiVersion: v1
kind: Service
metadata:
  name: proxy-repl{{ $RepeatableConfigGroupName "port"  }}
  labels:
    app: example
    component: repl{{ $RepeatableConfigGroupName "port"  }}
spec:
  type: LoadBalancer
  ports:
  - port: 80
    targetPort: {{repl ConfigOption "port" }}
  selector:
    app: example
    component: repl{{ $RepeatableConfigGroupName "port"  }}
```

#### Combined Usage Example

This is a thought exercise around using an operator that has both Repeatable Config Items and Groups. 

The expected output is 2 KafkaCluster CRs. They each have a different number of brokers. One cluster has 2 KafkaTopic CRs and the other only has 1.

```yaml
apiVersion: kafka.banzaicloud.io/v1alpha1
kind: KafkaCluster
metadata:
    labels:
        controller-tools.k8s.io: "1.0"
    name: kafka-cluster-repl{{ $ConfigOption name }}
spec:
    headlessServiceEnabled: true
    brokers:
    - id: repl{{ $ConfigOptionGroupName broker }}
      brokerConfigGroup: "default"
      brokerConfig:
        # Right now this cannot be templated separately for each kafka cluster
        envs:
          - name: +CLASSPATH
            value: "/opt/kafka/libs/dev/*:"
          - name: CLASSPATH+
            value: ":/opt/kafka/libs/extra-jars/*"
        # Neither can this
        brokerIngressMapping:
          - "ingress-az1"
    ...
```
```yaml
apiVersion: kafka.banzaicloud.io/v1alpha1
kind: KafkaTopic
metadata:
    name: kafka-repl{{ ConfigOption topic }}
spec:
    clusterRef:
        name: kafka-cluster-repl{{ ConfigOption name }}
    name: repl{{ ConfigOption topic }}
    partitions: 1
    replicationFactor: 1
```

### Revised Business Logic Overview

Additions where noted:
1. Customer passes in config values via CLI or UI
1. ConfigValue spec is saved to the `/userdata` folder along with upstream to the `upstream` directly
1. Kots renders the rest of `upstream` against the config values and also filters out any unnecessary files (e.g. preflight spec). 
This goes into the `base` directory along with a kustomize file.
    1. **NEW** First identify repeat groups and iterate through them
    1. **NEW** Inside that loop, identify any repeat items and render the YAML nodes
    1. **NEW** Complete render of other repl functions
    1. **NEW** Copy the file with a unique ID from the group or item into base
    1. **NEW** Repeat evaluation for simple repeat elements
    1. Render everything else.
1. Midstream changes are applied.
1. Downstream changes are applied.
1. Completed manifests are sent to the operator to get deployed.

## Design Limitations

1. This currently doesn't include any nested groups. This will likely be needed at a future point to support complex CRDs.
1. Configmap/secrets can only hold 5MB/1MB of data, respectively. No way to pass in an arbitrarily large file and have it passed along as configuration.
    * This more than likely eliminates the possibility of storing binary files, which has been specifically requested.
1. No ability to bulk-patch resources before they are rendered. Can still use Kustomize targets to accomplish this.

## Testing

Any template rendering based on this design should be refactored in such a way as to allow unit/integration testing of sample manifests against the expected API output. 

Testim tests (both smoke tests and release acceptance tests) will be augmented along with teh QAKots application to test the new UI elements for both features.

At a future point we will need to add a test framework for the CLI (or augment the current acceptance tests) to test that configuration can be passed to kotsadm as part of an unattended install.

## Alternatives Considered

1. Use Go Templating and new pipeline ConfigContext functions instead of targeting resources in the Config File like kustomize.
    1. Probably more intuitive for helm users.
    1. Potentially restricting and requires the use of comments to produce valid YAML.
    1. Philosophically different from the usage of Kustomize.
1. Having KOTS Use very basic search/parse capabilities to look for Config Items that were members of a template group, which would implicitly copy any resource using them N times.
This wasn't proposed because it seemed like it would implement a brute-force search of each file for every possible templateGroup config item.
1. As a more obscure solution, we could build a [custom generator in Go for kustomize](https://kubectl.docs.kubernetes.io/guides/extending_kustomize/) that takes in arbitrary templates and spits out the results directly. This didn't seem to have too many advantages over using the standard go tooling, but would have require more complexity to manage in KOTS.
1. Having `ConfigContext` methods that returned valid YAML and/or JSON was also discussed, but this would require passing in templates as arguments for something complicated like rendering a whole configmap.

## Security Considerations

As configuration is already part of the app definition, this proposal doesn't anticipate any changes to security posture.

Because the resources can be generated or extended dynamically, it's expected that the onus is on the vendor to ensure this doesn't not open any vulnerabilities in their application.

## References

Kustomize Resources
1. [Golang Text/Template Package](https://golang.org/pkg/text/template)
1. [Generic Generator Discussion](https://github.com/kubernetes-sigs/kustomize/issues/126)
1. [JSON Path Example](https://github.com/yubessy/example-kustomize-cronjob-multiple-schedule)

(Thanks to vmware-tanzu/velero for this design template)
