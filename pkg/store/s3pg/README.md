# s3pgstore

This backing store uses S3 for application archives and support bundles.
In addition, this store uses postgres for storage of all metadata and cache.
There are some scenarios where this store uses the local Kubernetes cluster for storing some sensitive information (gitops, etc).
