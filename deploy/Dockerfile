FROM debian:stretch-slim

RUN apt-get update && apt-get install -y --no-install-recommends \
    curl ca-certificates git \
  && rm -rf /var/lib/apt/lists/*

# Setup user
RUN useradd -c 'kotsadm user' -m -d /home/kotsadm -s /bin/bash -u 1001 kotsadm
USER kotsadm
ENV HOME /home/kotsadm

COPY ./bin/kotsadm /kotsadm
COPY ./web/dist /web/dist
USER root
RUN chmod a+x /kotsadm
RUN chmod a+w /web/dist/*
USER kotsadm
WORKDIR /

EXPOSE 3000
# ARG version=unknown
# ENV VERSION=${version}
ENTRYPOINT ["/kotsadm"]
CMD ["api"]
