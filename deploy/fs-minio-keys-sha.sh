#!bin/bash

set -e

export FS_KOTS_DIR=/fs/.kots
export FS_MINIO_KEYS_SHA_FILE=$FS_KOTS_DIR/minio-keys-sha.txt

if [ "$#" -eq 0 ]; then
    echo '{"success": false, "errorMsg": "Keys SHA argument missing"}'
    exit 0
fi

if [ ! -d "$FS_KOTS_DIR" ]; then
  mkdir -p -m 777 "$FS_KOTS_DIR"
fi

echo "$1" > "$FS_MINIO_KEYS_SHA_FILE"
echo '{"success": true}'
