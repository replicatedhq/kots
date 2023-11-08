FROM debian:bookworm-slim

ENV DEBIAN_FRONTEND=noninteractive

RUN apt-get update && apt-get install -y --no-install-recommends \
    curl ca-certificates git \
  && apt-get clean \
  && apt-get autoremove -y \
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
