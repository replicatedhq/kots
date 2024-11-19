#!/bin/bash

set -e

function is_not_in_path() {
  if ! command -v "$1" > /dev/null; then
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

function check_jq() {
  if is_not_in_path jq; then
    echo "jq is not installed. Attempting to install it using brew."
    if is_not_in_path brew; then
      echo "brew is not installed. Please install jq manually."
      return
    fi
    brew install jq
  fi
}

check_gomplate
check_jq
