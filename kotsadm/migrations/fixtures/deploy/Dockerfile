FROM postgres:10.7

ENV POSTGRES_USER=kotsadm
ENV POSTGRES_PASSWORD=password
ENV POSTGRES_DB=kotsadm

## Insert fixtures
COPY ./fixtures.sql /docker-entrypoint-initdb.d/

