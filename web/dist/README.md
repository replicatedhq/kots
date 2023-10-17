The dist directory is used in the go build to embed the compiled web resources into the kotsadm binary.  Because 
web isn't always built (testing, okteto, etc), this README.md will allow compiling of the go binary without first 
building web. 