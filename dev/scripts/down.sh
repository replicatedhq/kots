#!/bin/bash

set -e

component=$1

# Check if a deployment name was provided
if [ -z "$component" ]; then
	echo "Error: No deployment name provided."
	exit 1
fi

# Check if already down
if [ ! -f "./dev/patches/$component-down.yaml.tmp" ]; then
  echo "Error: already down, run 'make $component-up' first."
  exit 1
fi

echo "Reverting deployment..."

kubectl replace -f ./dev/patches/$component-down.yaml.tmp --force
rm ./dev/patches/$component-down.yaml.tmp
