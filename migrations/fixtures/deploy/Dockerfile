ARG POSTGRES_14_TAG=14.5-alpine
FROM postgres:$POSTGRES_14_TAG

ENV POSTGRES_USER=kotsadm
ENV POSTGRES_PASSWORD=password
ENV POSTGRES_DB=kotsadm

## Insert fixtures
COPY ./fixtures.sql /docker-entrypoint-initdb.d/
