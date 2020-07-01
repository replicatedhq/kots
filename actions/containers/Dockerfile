FROM replicated/gitops-builder:buildkite-go14-node10

# builder sets user to builder, which doesn't work in github actions
USER root

COPY entrypoint.sh /entrypoint.sh
ENTRYPOINT ["/entrypoint.sh"]