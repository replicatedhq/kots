ARG SCHEMAHERO_TAG

FROM schemahero/schemahero:${SCHEMAHERO_TAG} AS base

FROM debian:buster
WORKDIR /

COPY --from=base /schemahero /schemahero

RUN apt-get update \
  && apt-get install -y --no-install-recommends \
    ca-certificates \
  && apt-get install -y --only-upgrade --no-install-recommends \
    passwd login \
  && rm -rf /var/lib/apt/lists/*

RUN useradd -c 'schemahero user' -m -d /home/schemahero -s /bin/bash -u 1001 schemahero
USER schemahero
ENV HOME /home/schemahero

COPY --chown=schemahero:schemahero ./tables ./tables

CMD ["apply"]

ENTRYPOINT [ "/schemahero" ]
