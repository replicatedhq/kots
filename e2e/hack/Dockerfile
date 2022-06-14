FROM ubuntu:latest

ENV DEBIAN_FRONTEND=noninteractive

RUN apt-get update \
    && apt-get install -y --no-install-recommends \
        ca-certificates curl git sudo \
    && rm -rf /var/lib/apt/lists/*

env INSTALL_DIR=/usr/local/bin
env KREW_ROOT=$INSTALL_DIR/krew
env PATH=$INSTALL_DIR/node_modules/.bin:$KREW_ROOT/bin:$PATH

ADD hack/deps.sh /usr/local/bin/deps.sh

RUN /usr/local/bin/deps.sh

CMD [ "/bin/bash" ]
