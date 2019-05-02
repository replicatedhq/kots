#!/bin/sh

kustomize build overlays/ship | kubectl apply -f -