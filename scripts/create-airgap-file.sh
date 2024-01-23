#!/bin/bash

no_minio=$1

if [ "$no_minio" == "true" ]; then
	cat > "${BUNDLE_DIR}"/airgap.yaml <<EOF
apiVersion: kots.io/v1beta1
kind: Airgap
spec:
  format: docker
  savedImages:
  - kotsadm/kotsadm:${GIT_TAG//\'/}
  - kotsadm/kotsadm-migrations:${GIT_TAG//\'/}
  - kotsadm/dex:${DEX_TAG//\'/}
  - kotsadm/rqlite:${RQLITE_TAG//\'/}
  - replicated/local-volume-provider:${LVP_TAG//\'/}
EOF
else
	cat > "${BUNDLE_DIR}"/airgap.yaml <<EOF
apiVersion: kots.io/v1beta1
kind: Airgap
spec:
  format: docker
  savedImages:
  - kotsadm/kotsadm:${GIT_TAG//\'/}
  - kotsadm/kotsadm-migrations:${GIT_TAG//\'/}
  - kotsadm/dex:${DEX_TAG//\'/}
  - kotsadm/minio:${MINIO_TAG//\'/}
  - kotsadm/rqlite:${RQLITE_TAG//\'/}
  - replicated/local-volume-provider:${LVP_TAG//\'/}
EOF
fi
