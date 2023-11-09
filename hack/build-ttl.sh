#!/bin/bash
set -e

export CURRENT_USER=`id -u -n`

make build-ttl.sh

GREEN='\033[0;32m'
NC='\033[0m' # No Color

printf "\n\n\n"
printf "Run command:        ${GREEN}kubectl edit deployment kotsadm${NC}\n"
printf "Replace image with: ${GREEN}ttl.sh/${CURRENT_USER}/kotsadm:24h${NC}\n"
printf "\n"
printf "These images are good for 24 hours\n"
