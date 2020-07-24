FROM debian:stretch-slim

RUN apt-get update && apt-get install -y --no-install-recommends \
    curl ca-certificates git \
  && rm -rf /var/lib/apt/lists/*

# Setup user
RUN useradd -c 'kotsadm user' -m -d /home/kotsadm -s /bin/bash -u 1001 kotsadm
USER kotsadm
ENV HOME /home/kotsadm

COPY --chown=kotsadm:kotsadm ./bin/kurl_proxy /kurl_proxy
COPY --chown=kotsadm:kotsadm ./assets /assets

WORKDIR /

EXPOSE 8800
# ARG version=unknown
# ENV VERSION=${version}
CMD ["/kurl_proxy"]
