#!/bin/bash

#
# Install the SchemaHero database driver plugins that KOTS needs to run
# schemahero in-process (pkg/persistence.UpdateDBSchema).
#
# As of schemahero v0.23+ the database drivers (postgres, rqlite, ...) are no
# longer compiled into the schemahero library: they are hashicorp/go-plugin
# binaries that schemahero launches as subprocesses and discovers on disk. In
# the shipped images these binaries come from apk packages installed into
# /var/lib/schemahero/plugins at build time (see deploy/apko.yaml.tmpl and
# migrations/deploy/apko.yaml.tmpl), so nothing is fetched at runtime.
#
# CI runs the migration integration test (integration/database) directly with
# `go test`, outside of those images, so it needs the plugin binaries provisioned
# on the runner. This script builds them from the schemahero source at the exact
# version pinned in go.mod, guaranteeing the plugin RPC protocol matches the
# schemahero library KOTS links against.
#
# Usage:
#   scripts/install-schemahero-plugins.sh [DEST_DIR]
#
# DEST_DIR defaults to /var/lib/schemahero/plugins (schemahero's default
# discovery path). Point SCHEMAHERO_PLUGIN_PATH at DEST_DIR when it is not the
# default, e.g.:
#   scripts/install-schemahero-plugins.sh "$RUNNER_TEMP/schemahero-plugins"
#   SCHEMAHERO_PLUGIN_PATH="$RUNNER_TEMP/schemahero-plugins" make ci-test
#
set -euo pipefail

DEST_DIR="${1:-/var/lib/schemahero/plugins}"

# Drivers KOTS exercises in-process: postgres (legacy postgres->rqlite migration)
# and rqlite (schema apply). Extend this list if new in-process drivers are used.
DRIVERS=(postgres rqlite)

# Pin to the exact schemahero version KOTS links against so the plugin's
# go-plugin protocol matches the library. Derived from go.mod, so it stays in
# sync automatically when the dependency is bumped.
SCHEMAHERO_VERSION="$(go list -m -f '{{.Version}}' github.com/schemahero/schemahero)"

echo "Installing schemahero plugins ${DRIVERS[*]} at ${SCHEMAHERO_VERSION} into ${DEST_DIR}"

SRC_DIR="$(mktemp -d)"
trap 'rm -rf "${SRC_DIR}"' EXIT

git clone --depth 1 --branch "${SCHEMAHERO_VERSION}" \
	https://github.com/schemahero/schemahero.git "${SRC_DIR}"

mkdir -p "${DEST_DIR}"
for driver in "${DRIVERS[@]}"; do
	echo "Building schemahero-${driver}..."
	# Each plugin is its own nested Go module under plugins/<driver>.
	( cd "${SRC_DIR}/plugins/${driver}" && go build -o "${DEST_DIR}/schemahero-${driver}" . )
done

echo "Installed:"
ls -l "${DEST_DIR}"
