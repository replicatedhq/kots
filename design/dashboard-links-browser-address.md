# Enable Dashboard Buttons on connections that are not port forwarded

Currently, any vendor-defined dashboard button that are included in a KOTS application (https://kots.io/vendor/config/dashboard-buttons/) always refer to "localhost".
This doesn't work when accessing the cluster over a connection that isn't "localhost", for example, when using the embeded kURL cluster.
The proposal here is to udpate the implementation of this button to be compatible with all environments.

## Goals

- The dashboard buttons defined in https://kots.io/vendor/config/dashboard-buttons/ should link to the same hostname/IP that the Admin Console is being accessed from
- Enabling the KOTS application to specify the protocol to use

## Non Goals

- Enabling portfowarding or firewall configuration for these buttons
- Enabling TLS termination for these buttons
- Enabling ingress for these buttons

## Background

Currently, the dashboard buttons redirect to localhost.
If a KOTS application adds a button to expose a linked service on port 80, the link will be to http://localhost:80.
Not all use of the Admin Console is over "localhost", and these buttons only work if the Admin Console is being accessed over localhost.

This design will change the hostname to the browser-supplied hostname.
Additionally, this design will allow the KOTS manifest to change the protocol from http to https, with a default of http.

## High-Level Design

When the React frontend creates the button resource, it should use the hostname/ip from the browser, not localhost.
Currently, these are read directly from the API (https://github.com/replicatedhq/kotsadm/blob/cdb2add1296632b69e347afebd67af965f3bc8eb/web/src/components/apps/Dashboard.jsx#L94).
The design will make the API return these as an object containing protocol, port, path.
The UI will calculate the link.


## Detailed Design

A KOTS application spec currently contains:

```yaml
  ports:
    - serviceName: "sentry"
      servicePort: 9000
      localPort: 9000
      applicationUrl: "https://sentry"
```

This link is currently sent down as `https://localhost:9000`.
The link field will be updated to be an object, and in the above example, it will contain:

```json
{
    "protocol": "https",
    "port": 9000,
    "path": "/"
}
```

All fields will be required in the JSON object.

The React front end will build the link by determining the hostname or IP address that is currently being used to access the Admin Console.
It will build a string using `${link.protocol}://${browser.hostname}:${link.port}/${link.path}`

If the port is `80`, it will not be included in the link.
If the path is `/`, there will not be a duplicate `/` on the link.

## Alternatives Considered

No alternatives considered.

## Security Considerations

There are no known security implications here.
This is building a link, not port forwarding or otherwise making a service accessible from outside the cluster.
