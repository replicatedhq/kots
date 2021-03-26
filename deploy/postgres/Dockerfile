FROM postgres:10.16

RUN apt-get update && apt-get upgrade -y && apt-get install -y --no-install-recommends \
    curl ca-certificates git \
  && rm -rf /var/lib/apt/lists/*
