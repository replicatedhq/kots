#!/bin/bash 

# 
# Check to see if the spdx-sbom-generator is installed. If it is not, download it, check to make sure that the md5sum is correct, and extract it. 
# 
if ! command -v spdx-sbom-generator &> /dev/null 
then 
	mkdir -p sbom
	curl -L https://github.com/spdx/spdx-sbom-generator/releases/download/v0.0.13/spdx-sbom-generator-v0.0.13-linux-amd64.tar.gz -o ./sbom/spdx-sbom-generator.tar.gz
	curl -L https://github.com/spdx/spdx-sbom-generator/releases/download/v0.0.13/spdx-sbom-generator-v0.0.13-linux-amd64.tar.gz.md5 -o ./sbom/spdx-sbom-generator.tar.gz.md5
	md5sum ./sbom/spdx-sbom-generator.tar.gz | cut --bytes=1-32 > ./sbom/checksum

	if ! cmp ./sbom/checksum ./sbom/spdx-sbom-generator.tar.gz.md5
	then 
		echo "spdx-sbom-generator.tar.gz md5 sum does not match"
		exit 1
	fi 

	tar -xzvf ./sbom/spdx-sbom-generator.tar.gz -C sbom
fi
