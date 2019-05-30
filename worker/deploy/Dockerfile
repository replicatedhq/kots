FROM debian:stretch-slim
WORKDIR /src
COPY ./bin/ship-cluster-worker /ship-cluster-worker
RUN apt-get update && apt-get install -y --no-install-recommends \
    curl ca-certificates \
  && rm -rf /var/lib/apt/lists/*

EXPOSE 3000
ARG version=unknown
ENV VERSION=${version}
ENTRYPOINT ["/ship-cluster-worker"]
