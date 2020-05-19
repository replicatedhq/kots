# Config Vars

Config option values are currently rendered through our template engine individually.
There is no shared state between config items.
Attempts have been made to work around this by adding [globals](https://github.com/replicatedhq/kots/commit/8bec22f66f6422fd4e25cdff25b7eebcd99be434#diff-b82706ecb35ffdea1e9f78aa454d2ec8R45) to KOTS.
This proposal is an attempt to allow the vendor defined complex, reusable variables for input to config items.

## Goals

- Config vars should be able to solve the shared state problem
- Config vars should be able to be referenced from the `groups` section using template functions
- Config vars should be able to be regenerated when a property of the license or config changes

## Non Goals

- 

## Questions

- Do we need to have the option to encrypt config vars?

## Background

See top section.

## High-Level Design

A new property `spec.vars` will be added to the `kots.io/v1beta1.Config` spec in which a vendor can
define variables for reuse in the `groups` section of the spec.
Config vars can be referenced in the `groups` spec through a template function `ConfigVar [id]`.
Config vars will be regenerated when the `genid` property changes.
The config vars result will be stored in the `kots.io/v1beta1.ConfigValues` spec.

## Detailed Design

### Defining Vars

Config vars can be defined using the vars section of the spec.
Choose one option below:

#### Option 1

Option 1 makes use of template functions.
The `value` property must render to parseable YAML.
In the example below the sprig function `genSelfSignedCert` returns a struct which is rendered to a
YAML object with properties `cert` and `key`.

```yaml
apiVersion: kots.io/v1beta1
kind: Config
metadata:
  name: sentry-enterprise
spec:
  vars:
    - name: selfSignedCert
      genid: '{{repl ConfigOption "hostname" | sha256sum }}'
      values: |
        {{repl- $cert := genSelfSignedCert (ConfigOption "hostname") nil nil 365 -}}
        cert: {{repl $cert.Cert }}
        key: {{repl $cert.Key }}
```

#### Option 2

Option 2 makes use of predefined functions.
In the example below the property `genSelfSignedCert` is a function that returns two properties
`cert` and `key`.

```yaml
apiVersion: kots.io/v1beta1
kind: Config
metadata:
  name: sentry-enterprise
spec:
  vars:
    - name: selfSignedCert
      genid: '{{repl ConfigOption "hostname" | sha256sum }}'
      genSelfSignedCert:
        cn: '{{repl ConfigOption "hostname" }}'
        ips: ~
        alternateDNS: ~
        daysValid: 365
```

### Referencing Vars

Config vars can be referenced using the template function `ConfigVar [id]`.
See the example below:

```yaml
  groups:
    - name: tls
      title: TLS
      items:
        - name: hostname
          title: Hostname
          type: text
        - name: tls_key
          title: TLS Key
          type: textarea
          value: |
            {{repl ConfigVar "selfSignedCert" "key" | b64dec }}
        - name: tls_cert
          title: TLS Cert
          type: textarea
          value: |
            {{repl ConfigVar "selfSignedCert" "cert" | b64dec }}
```

### Regenerating Vars

Config vars will be regenerated when the `genid` property of the var changes.
This property can be templated.
See the example below:

```yaml
  vars:
    - name: selfSignedCert
      genid: '{{repl ConfigOption "hostname" | sha256sum }}'
      ...
```

### Stored Value

The config vars result will be stored in the `kots.io/v1beta1.ConfigValues` spec.

```yaml
apiVersion: kots.io/v1beta1
kind: ConfigValues
metadata:
  name: sentry-enterprise
spec:
  values:
    selfSignedCert.key:
      value: '-----BEGIN RSA PRIVATE KEY-----\n...'
    selfSignedCert.cert:
      value: '-----BEGIN RSA PRIVATE KEY-----\n...'
```
