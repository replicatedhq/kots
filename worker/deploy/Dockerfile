FROM debian:stretch-slim
WORKDIR /src
COPY ./bin/kotsadm-worker /kotsadm-worker
RUN apt-get update && apt-get install -y --no-install-recommends \
    curl ca-certificates git \
  && rm -rf /var/lib/apt/lists/*

EXPOSE 3000
ARG version=unknown
ENV VERSION=${version}
ENTRYPOINT ["/kotsadm-worker"]
