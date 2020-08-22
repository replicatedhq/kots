#!/bin/bash
set -e

make build build-ttl.sh

GREEN='\033[0;32m'
NC='\033[0m' # No Color

printf "\n\n\n"
printf "Run command:        ${GREEN}kubectl edit deployment kotsadm-operator${NC}\n"
printf "Replace image with: ${GREEN}ttl.sh/${CURRENT_USER}/kotsadm-operator:12h${NC}\n"
printf "\n"
printf "This image is good for 12 hours\n"
