#!/bin/bash
set -e

export CURRENT_USER=`id -u -n`

make -C web deps build-kotsadm
make kotsadm build-ttl.sh
make -C api no-yarn deps build build-ttl.sh

GREEN='\033[0;32m'
NC='\033[0m' # No Color

printf "\n\n\n"
printf "Run command:        ${GREEN}kubectl edit deployment kotsadm-api${NC}\n"
printf "Replace image with: ${GREEN}ttl.sh/${CURRENT_USER}/kotsadm-api:12h${NC}\n"
printf "\n"
printf "Run command:        ${GREEN}kubectl edit deployment kotsadm${NC}\n"
printf "Replace image with: ${GREEN}ttl.sh/${CURRENT_USER}/kotsadm:12h${NC}\n"
printf "\n"
printf "These images are good for 12 hours\n"
