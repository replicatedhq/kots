FROM debian:stretch-slim

RUN apt-get update && apt-get install -y --no-install-recommends \
    curl ca-certificates git \
  && rm -rf /var/lib/apt/lists/*

# Setup user
RUN useradd -c 'kotsadm user' -m -d /home/kotsadm -s /bin/bash -u 1001 kotsadm
USER kotsadm
ENV HOME /home/kotsadm

COPY ./bin/kurl_proxy /kurl_proxy
COPY ./assets /assets
USER root
RUN chmod a+x /kurl_proxy
RUN chmod a+w /assets/*
USER kotsadm
WORKDIR /

EXPOSE 8800
# ARG version=unknown
# ENV VERSION=${version}
CMD ["/kurl_proxy"]
