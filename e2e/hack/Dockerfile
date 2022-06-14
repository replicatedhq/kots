FROM ubuntu:latest

ENV DEBIAN_FRONTEND=noninteractive

RUN apt-get update \
    && apt-get install -y --no-install-recommends \
        ca-certificates curl sudo \
    && rm -rf /var/lib/apt/lists/*

env PATH=$PATH:/usr/local/bin/node_modules/.bin/

ADD hack/deps.sh /usr/local/bin/deps.sh

RUN /usr/local/bin/deps.sh

CMD [ "/bin/bash" ]
