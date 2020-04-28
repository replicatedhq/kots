Adding Capability for BYO Object Store
================

This document outlines the current architecture of how Object Stores are used by [kots](https://github.com/replicatedhq/kots) and [kotsadm](https://github.com/replicatedhq/kotsadm), and a proposal for modifying how it is configured to allow end users of `kotsadm` to bring their own object store rather than using an embedded minio instance.

### Goals

- For `kots install` and `kots pull`, allow an end user to bring their own object store rather than using an embedded minio instance that requires the cluster to include a storage class.
- Move closer to unifying the YAML used across `kots install` and the [kURL addon](https://kurl.sh/docs/add-ons/kotsadm) (or, at the very least, avoid creating any more differences between the two methods)

### Architecture Background

An Object store is required by kots to store each version of the deployable application. For each version deployed, a tar bundle is pushed to the object store which contains the `upstream`, `base`, and `overlays` directories from the application. This includes any end-user specified `Config`, `Application`, and/or `License` state stored in `upstream/userdata`, as well as any custom last-mile [kustomize](https://kustomize.io) changes made in `overlays/`.


When a new upstream version is detected by kotsadm, either via polling or via an airgap bundle upload, the new upstream version will be templated out, with the user-supplied config and overlays from the previous version being applied to the new version from upstream. Once the upstream has been inflated to `upstream`, `base`, and `overlays`, it is bundled and pushed to a new key in the object store.

Versions stored in the object store are used

1. To deploy a specific version of the application after the upstream has been processed
1. To roll back to previous versions of the application (if [allowRollback](https://kots.io/reference/v1beta1/application/#allowrollback) is specified).



Current Architecture 
---------------

In every case, five environment variables are configured for `kotsadm` and `kotsadm-api`:

- `S3_ACCESS_KEY_ID`
- `S3_SECRET_ACCESS_KEY`
- `S3_ENDPOINT`
- `S3_BUCKET_NAME`
- `S3_BUCKET_ENDPOINT` 

`S3_BUCKET_ENDPOINT` appears to be a boolean, configured whether to use a custom endpoint. Seems to always be `true`, I believe this controls whether to override the default S3 endpoint in our S3 client.



### With [kots install](https://kots.io/kots-cli/install/)

On `kots install`, a secret `kotsadm-minio` is created with the following fields:

- `accesskey`
- `secretkey`

These are generated UIDs, as of kots 1.14.2, they are UUIDs:

```go
		_, err := clientset.CoreV1().Secrets(namespace).Create(s3Secret(namespace, uuid.New().String(), uuid.New().String()))
```

`kotsadm` and `kotsadm-api` Deployments get their environment variables as follows:


- `S3_ACCESS_KEY_ID` from the `kotsadm-minio` secret
- `S3_SECRET_ACCESS_KEY` from the `kotsadm-minio` secret
- `S3_ENDPOINT` hardcoded to `http://kotsadm-minio:9000`
- `S3_BUCKET_NAME` hardcoded to `kotsadm`
- `S3_BUCKET_ENDPOINT` hardcoded to the string `true`


### With [kots pull](https://kots.io/kots-cli/pull/)

As with `kots install`, `kotsadm` and `kotsadm-api` Deployments get their environment variables as follows:

- `S3_ACCESS_KEY_ID` from the `kotsadm-minio` secret
- `S3_SECRET_ACCESS_KEY` from the `kotsadm-minio` secret
- `S3_ENDPOINT` hardcoded to `http://kotsadm-minio:9000`
- `S3_BUCKET_NAME` hardcoded to `kotsadm`
- `S3_BUCKET_ENDPOINT` hardcoded to the string `true`


a `kotsadmtypes.DeployOptions` is built from an instance of `upstream.UpstreamSettings` in `generateNewAdminConsoleFiles`, and the files are written to disk.

### With [kURL](https://kurl.sh/docs/add-ons/kotsadm)

In kURL, we create a secret `kotsadm-s3`, with the following fields:

- `access-key-id` -- generated
- `secret-access-key` -- generated
- `endpoint` -- hardcoded to `http://rook-ceph-rgw-rook-ceph-store.rook-ceph`

`kotsadm` and `kotsadm-api` deployments pull this in via:

- `S3_ACCESS_KEY_ID` from the `kotsadm-s3` secret
- `S3_SECRET_ACCESS_KEY` from the `kotsadm-s3` secret
- `S3_ENDPOINT` from the `kotsadm-s3` secret
- `S3_BUCKET_NAME` hardcoded to `kotsadm`
- `S3_BUCKET_ENDPOINT` hardcoded to the string `true`

Proposed New Changes
---------------------

This covers a proposal for `kots install`. It includes migrating the `kotsadm-minio` secret to `kotsadm-s3`, which probably is trickier in the `kots pull` case, but I'd like to table that for now, avoiding `kots pull` considerations. If it is clear this will cause a lot of friction, we can consider leaving the secret called `kotsadm-minio` and kick that can down the road.

### API Changes


- Add a flag to the kots CLI --object-store=external options are `minio,external`, flags whether to use an external object store. Used so as not to overload access-key-id and friends with toggling this external object store functionality on/off. I'm on the fence about requiring this to be passed, maybe we could simplify and remove it. Defaults to `minio`
- Add a flag to the kots CLI --object-store-external-access-key-id
- Add a flag to the kots CLI --object-store-external-secret-access-key
- Add a flag to the kots CLI --object-store-external-bucket-name
- Add a flag to the kots CLI --object-store-external-endpoint  but can be empty to use Amazon S3

**Question:** should the object store types `minio` and `external` should be constants in the `types` package? Or a separate configuration package? https://github.com/replicatedhq/kots/blob/master/pkg/kotsadm/types/constants.go

### Internal changes

#### Validating configuration


If `--object-store=minio`, validation should fail if any of the following is set:

- `--object-store-external-access-key-id`
- `--object-store-external-secret-access-key`
- `--object-store-external-bucket-name`
- `--object-store-external-endpoint`


If `--object-store=external`, validation should fail if any of the following is unset:

- `--object-store-external-access-key-id`
- `--object-store-external-secret-access-key`
- `--object-store-external-bucket-name`


Once validated, these options should all be added to the `kotsadmtypes.DeployOptions` object passed into `kotsadm.Deploy()`.

#### Migrating the kotsadm-minio secret

When creating/ensuring the object store secret, rename the secret `kotsadm-minio` to `kotsadm-s3`. We should first check for a `kotsadm-minio` secret and migrate it to `kotsadm-s3`. Field names stay the same. In the odd case that something previously failed mid-migration, and both `kotsadm-minio` and `kotsadm-s3` exist, use the values from `kotsadm-s3` and delete `kotsadm-minio`. 

When ensuring kotsadm and kotsadm-api, patch the `valueFrom` `secretKeyRef` invocations in the environment to pull from the new kotsadm-s3 secret. This can be done transparently by updating the implementation of `kotsadmDeployment` and `kotsadmApiDeployment` functions that build the k8s objects.

#### Mapping user-supplied values when creating the secret

Based on `object-store=external`, pipe in the values of `object-store-external-access-key-id` and `object-store-external-secret-access-key` to the `kotsadm-s3` secret values of `accesskey` and `secretkey`.

Add a new field `bucket-name` to match the field name in the secret created by kURL. If `--object-store=minio`, set this to the current hardcoded value: `kotsadm`. 

Add a new field `endpoint` to match the field name in the secret created by kURL. If `--object-store=minio`, set this to the current hardcoded value: `http://kotsadm-minio:9000`. 


Add a new field for `bucket-endpoint`, if `--object-store=external`, either:

1. If `--object-store-external-endpoint` is empty, set `endpoint` and `bucket-endpoint` to an empty string `""`
1. If `--object-store-external-endpoint` is not empty, set `endpoint` to the value provided and set `bucket-endpoint` to the string `"true"`



#### Updating deployed objects

`accesskey` and `secretkey` should continue to be accessed in the same way by both `kotsadm` and `kotsadm-api` deployments.

`kotsadm` and `kotsadm-api` should be modified to pull the `S3_ENDPOINT` field from the `kotsadm-s3` secret rather than hardcoding it to the embedded `kotsadm-minio` service. 

`kotsadm` and `kotsadm-api` should be modified to pull the `S3_BUCKET_NAME` field from the `kotsadm-s3` secret rather than hardcoding it to the default `kotsadm` bucket.

`kotsadm` and `kotsadm-api` should be modified to pull the `S3_BUCKET_NAME` field from the `kotsadm-s3` secret rather than hardcoding it to the default `kotsadm` bucket.

