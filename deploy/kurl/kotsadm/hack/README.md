# kURL Add-On Development Environment

1. Create a new test server

1. SSH into your test server and install dependencies.

   ```shell
   sudo apt-get update && sudo apt-get install openssl rsync # yum install openssl rsync
   mkdir -p kurl
   curl -L https://kurl.sh/dist/kubernetes-1.24.6.tar.gz | tar -xzv -C kurl -f -
   curl -L https://k8s.kurl.sh/dist/containerd-1.6.8.tar.gz | tar -xzv -C kurl -f -
   ```

1. On your dev server, run the make command to build kURL and KOTS dependencies and sync them to your test server.
   This command will watch files in the generated kurl directory for changes.
   If you need to make a change to the add-on, you can run `make generate` in another terminal session.

   ```shell
   REMOTES=34.72.15.201 make
   ```

1. Now you can test your changes by installing kURL and KOTS on your test server.

   ```shell
   cat install.sh | sudo bash
   ```
