ARG SCHEMAHERO_TAG

FROM schemahero/schemahero:${SCHEMAHERO_TAG} AS base

FROM debian:buster
WORKDIR /

COPY --from=base /schemahero /schemahero

ENV DEBIAN_FRONTEND=noninteractive

# gzip and liblzma5 are installed to patch cves
RUN apt-get update \
  && apt-get install -y --no-install-recommends \
    ca-certificates \
  && apt-get install -y --only-upgrade --no-install-recommends \
    passwd login gzip liblzma5 \
  && apt-get clean \
  && apt-get autoremove -y \
  && rm -rf /var/lib/apt/lists/*

RUN useradd -c 'schemahero user' -m -d /home/schemahero -s /bin/bash -u 1001 schemahero
USER schemahero
ENV HOME /home/schemahero

COPY --chown=schemahero:schemahero ./tables ./tables

ENTRYPOINT ["/schemahero"]
CMD ["apply"]
