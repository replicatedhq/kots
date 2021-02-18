
The Dockerfiles, skaffold.yaml, Makefile, and kustomize/ subdirectory in this directory are for developing locally with skaffold.
The kotsadm addon has the deployment yaml that will be installed on a kurl cluster.

The proxy requires a secret with a cert and key to start.
The kotsadm addon from the [kURL](https://github.com/replicatedhq/kurl) project will generate this secret when installed.
That secret will have the annotation `acceptAnonymousUploads` which allows anybody to upload a new cert at /tls.
After the first upload that flag will be turned off and the cert/key in the secret will be replaced with the uploaded pair.
Navigating to /tls after that will show a warning rather than an upload form.
Manually add the flag back to the secret to re-enable uploads.

## Developing

Run `make up` to start the proxy on port 8800 while watching for changes to `assets/` and `main.go`.

`assets/` is a static directory that also has some html templates. From the insecure page link to files with `/assets/styles.css` and from the `/tls` page with `/tls/assets/styles.css`.

Delete the secret to return to a state where uploads are accepted.
