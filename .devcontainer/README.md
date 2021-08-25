# Replicated KOTS Codespace Container

Most of the code here is borrowed from this [Microsoft repo of base images](https://github.com/microsoft/vscode-dev-containers), except for replicated specific things.

## Notes
* k3d *DOES NOT* work with DinD. You have to use the docker with docker install instead.
* If you try to do a skaffold dev in the onCreate.sh, it will take a LONG time to start and I've seen weird behavior here. Need to find a better soln to get the images on the cluster.
* Might be faster to install kubectl plugins on the `$PATH` in the `Dockerfile` instead of downloading them `onCreate.sh`.
