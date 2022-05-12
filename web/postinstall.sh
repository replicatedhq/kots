#!/bin/bash


if [[ -n "${PUBLIC_ASSET_PATH}" ]]; then
    mkdir -p ${PUBLIC_ASSET_PATH}/vs && cp -R node_modules/monaco-editor/min/vs/ ${PUBLIC_ASSET_PATH}/vs
else 
    mkdir -p ./public/vs && cp -R node_modules/monaco-editor/min/vs/ ./public/vs
fi