ARG TAG=10.19-alpine
FROM postgres:$TAG

ENV POSTGRES_USER=kotsadm
ENV POSTGRES_PASSWORD=password
ENV POSTGRES_DB=kotsadm

## Insert fixtures
COPY ./fixtures.sql /docker-entrypoint-initdb.d/

