FROM postgres:10.7

ENV POSTGRES_USER=shipcloud
ENV POSTGRES_PASSWORD=password
ENV POSTGRES_DB=shipcloud

## Insert fixtures
COPY ./fixtures.sql /docker-entrypoint-initdb.d/

