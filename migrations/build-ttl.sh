#!/bin/bash
set -e

CURRENT_USER=${GITHUB_USER:-$(id -u -n)}
IMAGE=ttl.sh/${CURRENT_USER}/kotsadm-migrations:24h

docker build -f deploy/Dockerfile -t ${IMAGE} .
docker push ${IMAGE}

GREEN='\033[0;32m'
NC='\033[0m' # No Color

printf "\n\n\n"
printf "Run command:        ${GREEN}kubectl edit deployment kotsadm${NC}\n"
printf "Replace image with: ${GREEN}${IMAGE}${NC}\n"
printf "\n"
printf "This image is good for 24 hours\n"
