# Contributing to Kots

In this doc we'll go over the steps required to contribute to kots and setup a dev environment.

## Install Go

```shell
brew install go
```

## Fork and then Clone this Repo

```shell
git clone https://github.com/<gh_username>/kots.git
```

## Build Kots

```shell
$ cd kots

$ make kots
```

## Test

Move the built binary
```shell
sudo cp bin/kots /usr/local/bin/kubectl-kots
```

Check the version to verify
```shell
kubectl kots version
```
