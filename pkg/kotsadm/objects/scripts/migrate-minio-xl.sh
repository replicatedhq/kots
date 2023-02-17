#!/bin/bash
set -e

if [ -z $S3_ENDPOINT_OLD ] ||
   [ -z $S3_ENDPOINT_NEW ] ||
   [ -z $S3_ACCESS_KEY_ID ] ||
   [ -z $S3_SECRET_ACCESS_KEY ] ||
   [ -z $S3_BUCKET_NAME ];
then
  echo 'object store configuration not found'
  exit 1
fi

# REFERENCE: https://min.io/docs/minio/linux/operations/install-deploy-manage/migrate-fs-gateway.html

# alias old and new minio instances
echo 'aliasing old and new minio instances'
mc alias set old $S3_ENDPOINT_OLD $S3_ACCESS_KEY_ID $S3_SECRET_ACCESS_KEY
mc alias set new $S3_ENDPOINT_NEW $S3_ACCESS_KEY_ID $S3_SECRET_ACCESS_KEY

# export and import config
echo 'exporting and importing config'

set +e
mc config export old > config.txt
if [ $? -ne 0 ]; then
    echo 'failed to export config, skipping...'
else
    mc config import new < config.txt
    if [ $? -ne 0 ]; then
      echo 'failed to import config, skipping...'
    fi
fi
set -e


# restart new minio instance
echo 'restarting new minio instance'
mc admin service restart new

# export and import cluster metadata
echo 'exporting and importing cluster metadata'
set +e
mc admin cluster bucket export old
if [ $? -ne 0 ]; then
    # old instances may throw error: "This 'admin' API is not supported by server in 'mode-server-fs'"
    # ref: https://github.com/minio/mc/issues/4409 and https://github.com/minio/console/issues/863
    echo "exporting minio bucket metadata failed, skipping"
    # make sure the bucket exists
    mc mb new/$S3_BUCKET_NAME
else
    mc admin cluster bucket import new cluster-metadata.zip
    if [ $? -ne 0 ]; then
        echo "importing minio bucket metadata failed, skipping"
        # make sure the bucket exists
        mc mb new/$S3_BUCKET_NAME
    fi
fi
set -e

# export and import iam info
echo 'exporting and importing iam info'
set +e
mc admin cluster iam export old
if [ $? -ne 0 ]; then
    # old instances may throw error: "This 'admin' API is not supported by server in 'mode-server-fs'"
    # ref: https://github.com/minio/mc/issues/4409 and https://github.com/minio/console/issues/863
    echo "exporting minio iam info failed, skipping"
else
    mc admin cluster iam import new alias-iam-info.zip
    if [ $? -ne 0 ]; then
        echo "importing minio iam info failed, skipping"
    fi
fi
set -e

# mirror old bucket content to new bucket
echo 'mirroring old bucket content to new bucket'
mc mirror --preserve old/$S3_BUCKET_NAME new/$S3_BUCKET_NAME

# # stop old minio instance
# echo 'stopping old minio instance'
# mc admin service stop old

# # restart new minio instance
# echo 'stopping new minio instance'
# mc admin service stop new

echo 'migration ran successfully!'
