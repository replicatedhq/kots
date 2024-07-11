The dist directory is used in the go build to embed the compiled web resources into the kots & kotsadm binaries. Because 
web isn't always built (testing, dev, etc), this README.md will allow compiling of the go binary without first 
building web. 
