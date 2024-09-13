ARG SCHEMAHERO_TAG

FROM schemahero/schemahero:${SCHEMAHERO_TAG} AS base

FROM debian:bookworm
WORKDIR /

COPY --from=base /schemahero /schemahero

ENV DEBIAN_FRONTEND=noninteractive

RUN apt-get update \
  && apt-get install -y --no-install-recommends \
    ca-certificates \
  && apt-get install -y --only-upgrade --no-install-recommends \
    passwd login \
  && apt-get clean \
  && apt-get autoremove -y \
  && rm -rf /var/lib/apt/lists/*

RUN useradd -c 'schemahero user' -m -d /home/schemahero -s /bin/bash -u 1001 schemahero
USER schemahero
ENV HOME /home/schemahero

COPY --chown=schemahero:schemahero ./tables ./tables

ENTRYPOINT ["/schemahero"]
CMD ["apply"]
