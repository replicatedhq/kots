# kotskinds

This directory contains the definitions and clientsets for the kots.io kinds. These aren't CRDs and controllers, but are implemented as normal Kubernetes objects. This allows us to use the client-go and other functionality to parse and ensure conformance.

## Building

To build these, simply run `make` in this directory. This will execute the kubernetes controller-gen and client-gen code to generate a typed client and the necessary deepcopy methods for these types.



