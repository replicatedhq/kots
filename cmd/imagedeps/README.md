# Image Deps

This is a utility designed to take a list of images as input, and to generate FQN's of each input image including the 
latest tag and output this information into an environment file, and a Go source file containing constant declarations. It
is designed to be run as a script, for example:
```shell
go run github.com/replicatedhq/kots/cmd/imagedeps 
2021/09/15 10:33:46 started tagged image file generator
2021/09/15 10:33:48 successfully generated constant file "pkg/image/constants.go"
2021/09/15 10:33:48 successfully generated dot env file ".image.env"
```
If successful it will generate two files, a file of constant declarations *pkg/image/constants.go* and *.image.env*.  The 
Go file contains constant declarations of image references with the latest version tags.  The .env file contains environment
variables defining the latest tags for images. 

## Input 
Latest tags will be found for images that are defined in a text file *cmd/imagedeps/image-spec*. Each line contains space delimited
information about an image and an optional filter. If the filter is present, only tags that match will be included.  This 
is useful to restrict release tags to a major version, or to filter out garbage tags. 

| Name | Image URI | Matcher Regexp (Optional) |
|------|--------------------|----------|
| Name of the image for example **minio** | Untagged image reference **kotsadm/minio**| An optional regular expression, only matching tags will be included.  |

### Sample image-spec
```text
minio kotsadm/minio
rqlite kotsadm/rqlite
dex kotsadm/dex
```
The preceding image spec will produce the following environment and Go files.
```shell
MINIO_TAG='0.20231101.183725'
RQLITE_TAG='7.21.4'
DEX_TAG='2.37.0'
```
```go
package image

const (
	Minio = "kotsadm/minio:0.20231101.183725"
	Rqlite = "kotsadm/rqlite:7.21.4"
	Dex = "kotsadm/dex:2.37.0"
)
```

## GitHub 
Some of the image tags are resolved by looking at the GitHub release history of their associated projects.  This involves 
interacting with the GitHub API.  The program uses an optional environment variable `GITHUB_AUTH_TOKEN` which is a GitHub API token 
with **public_repo** scope for the purpose of avoiding rate limiting.  The program will work without `GITHUB_AUTH_TOKEN`
but it is not recommended. 
