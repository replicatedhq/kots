
# kotsadm: Advance To Go

TODO write the why and motivations for this
- Vendor KOTS in and remove FFI
- Vendor Troubleshoot in
- Remove Ship code
- Reduce complexity of the project
...


## Proposal

We should treat the kotsadm repo as a go project, with a separate “web” directory containing the static react site.

The end result will be 1 container running gin or gorilla as a web server. The static .js (and index.html) will be served from this container also, so there’s no need for a separate nginx container to be serving the web front end.

This will take more than one release cycle of kotsadm to complete. We should be able to deploy updates to kotsadm that function in the “split mode” where some requests are made to the new Go REST API while others continue to be made to the Typescript GQL API. During this period, we should retain the same number of containers running in production deployments.

Forever, the kotsadm-web container will be served in webpack-dev-server in the dev environment. During the transition, the api will be proxied through the gorilla mux server to validate this is working. So the typescript api will not have a nodeport, but the new go server will take the nodeport that the typescript api currently has.

## #Release 1.10.0
The first release will replace nginx with a new container image named “kotsadm”. We currently have “kotsadm-web” and “kotsadm-api”. This new image will be a gorilla mux server in go that serves the static files and all of the go rest handlers that we build.

The first release will have login implemented and removed from the typescript gql api.

This release will continue to ship the kotsadm-api container, but it will be proxied from the gorilla “kotsadm” container instead of the nginx container that’s currently proxying it. Gorilla will need to know the endpoints of the rest to proxy and proxy /graphql also.

No more web container. Still have the same number of container images.

### Release 1.11.0
Move /controllers from typescript into go.

### Release 1.12.0-...  (as many as it takes):
Move GQL resolvers into Rest endpoints.

### Release 1.xx.0 (next)
Remove operator and make that part of kotsadm directly as a new command. When running in a single namespace, there’s no need for operator. Kotsadm can be it’s own operator. But the operator command in the project will allow it to connect to a kotsadm api server to run separately.

## Rest API

Current GraphQL schema is defined: https://github.com/replicatedhq/kots/kotsadm/blob/61e89966eefb76fd02d276d21314389997105818/api/src/schema/query.ts and https://github.com/replicatedhq/kots/kotsadm/blob/61e89966eefb76fd02d276d21314389997105818/api/src/schema/mutation.ts. There are also numerous Rest APIs defined in https://github.com/replicatedhq/kots/kotsadm/tree/61e89966eefb76fd02d276d21314389997105818/api/src/controllers. Some of these are deprecated from Ship, and not needed.

New Rest API (Go version) will be:

| Path | Method | Description |
|------|--------|-------------|
| /v1/login | POST | log in
| /v1/logout | POST | log out
| /healthz | GET | healthz request
| /v1/metadata | GET | return app metadata
| /v1/apps | GET | list all apps
| /v1/app/:appId | GET | get a single app by id
| /v1/app/:appId/history | GET | get the version history for an app
| /v1/app/:appId/sequence/:sequence/files | GET | get the files for an app sequence
| /v1/app/:appId/troubleshoot | GET | list the support bundles for an app
| /v1/app/:appId/sequence/:sequence/preflight | POST | accept failed or warned preflights

(Note, you need to get apps using /v1/apps, find the appId from the list there (match on slug) and make requests using the appId)

## App
The app type will be a lot larger now, and will contain a lot of the info needed to render more of the dashboard, without needing smaller queries.

```
{
  id: "abc",
  name: "app name",
  slug: "app-slug",
  config: {
    ...
  },
  registry: {
    ...
  },
  license: {
    ...
  },
  prometheus: {
    ...
  },
  currentVersion: {
   // a single app version, see below
  },
}
```

## App Version

```
{
    updateCursor: "1",
    sequence: 1,
    versionLabel: "1.0.0",
    status: "deployed",
}
```

## Websockets
Instead of polling, we will implement a websocket ticketing and auth system....  TODO define.

## Deprecated features that will be removed

- GitHub Login
- Username and password login
- Support for Ship Applications
- Multi-tenant mode
- User features
- User info
- Helm charts (there's some specific helm stuff in here)
- Support for multiple downstreams in an app
- "Clusters" will become "Namespaces".
