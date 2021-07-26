FROM schemahero/schemahero:0.12.4

USER root
RUN apt-get update && apt-get install -y --only-upgrade --no-install-recommends \
    passwd login \
  && rm -rf /var/lib/apt/lists/*
USER schemahero

ADD --chown=schemahero:schemahero ./tables ./tables

CMD ["apply"]
