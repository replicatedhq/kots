#!/bin/bash

set -e

function is_not_in_path() {
  if ! which "$1" > /dev/null; then
    echo "$1 is not installed"
    return 0
  fi
  return 1
}

function check_gomplate() {
  if is_not_in_path gomplate; then
    echo "gomplate is not installed. Installing it now."
    go install github.com/hairyhenderson/gomplate/v4/cmd/gomplate@latest
  fi
}

check_gomplate
