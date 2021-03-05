#!bin/bash

set -e

export FS_MINIO_CONFIG_PATH=/fs/.minio.sys/config
rm -rf "$FS_MINIO_CONFIG_PATH"
echo '{"success": true}'
