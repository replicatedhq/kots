#!/bin/bash

if [ -z ${GIT_TAG} ]; then
    echo "This script must run from GithubActions with GIT_TAG env variable set"
    exit 1
fi

if [ -z ${GITHUB_RUN_ID} ]; then
    echo "This script must run from GithubActions with GITHUB_RUN_ID env variable set"
    exit 1
fi

export KGRID_RUN_ID=`date +%s`-${GITHUB_RUN_ID}

echo ${REPLICATEDCOM_GITHUB_PRIVATE_KEY} | base64 -d > ~/github_private_key
chmod 600 ~/github_private_key
export GIT_SSH_COMMAND='ssh -i ~/github_private_key'
git config --global user.email ${GIT_COMMIT_COMMITTER_EMAIL}
git config --global user.name ${GIT_COMMIT_COMMITTER_NAME}

rm -rf ${GITOPS_REPO}
git clone --single-branch -b ${GITOPS_BRANCH} git@github.com:${GITOPS_OWNER}/${GITOPS_REPO}
cd ${GITOPS_REPO}

cat <<EOT > kgrid/version.yaml
apiVersion: kgrid.replicated.com/v1alpha1
kind: Version
metadata:
  name: version
  namespace: kgrid-system
  labels:
    runId: ${KGRID_RUN_ID}
spec:
  kots:
    latest: "${GIT_TAG}"
EOT

git add .
git commit --allow-empty -m "${PR_URL}"
git push origin ${GITOPS_BRANCH}

echo "::set-output name=kgrid-run-id::$KGRID_RUN_ID"