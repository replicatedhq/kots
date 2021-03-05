#!bin/bash

set -e

export FS_MINIO_CONFIG_PATH=/fs/.minio.sys/config
export FS_MINIO_KEYS_SHA_FILE=/fs/.kots/minio-keys-sha.txt

if [ ! -d "$FS_MINIO_CONFIG_PATH" ]; then
  echo '{"hasMinioConfig": false}'
  exit 0
fi

if [ ! -f "$FS_MINIO_KEYS_SHA_FILE" ]; then
  echo '{"hasMinioConfig": true}'
  exit 0
fi

export FS_MINIO_KEYS_SHA=$(cat $FS_MINIO_KEYS_SHA_FILE)
echo '{"hasMinioConfig": true, "minioKeysSHA":"'"$FS_MINIO_KEYS_SHA"'"}'
