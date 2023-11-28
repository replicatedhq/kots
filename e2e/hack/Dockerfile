FROM ubuntu:latest

ENV DEBIAN_FRONTEND=noninteractive

RUN apt-get update \
    && apt-get install -y --no-install-recommends \
        ca-certificates curl gnupg git sudo \
    && rm -rf /var/lib/apt/lists/*

ENV INSTALL_DIR=/usr/local/bin
ENV KREW_ROOT=$INSTALL_DIR/krew
ENV PATH=$INSTALL_DIR/node_modules/.bin:$KREW_ROOT/bin:$PATH

COPY deps.sh /usr/local/bin/deps.sh

RUN /usr/local/bin/deps.sh

CMD [ "/bin/bash" ]