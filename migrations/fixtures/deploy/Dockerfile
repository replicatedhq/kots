FROM centos/postgresql-10-centos7

ENV POSTGRESQL_USER=kotsadm
ENV POSTGRESQL_PASSWORD=password
ENV POSTGRESQL_DATABASE=kotsadm

## Insert fixtures
COPY ./fixtures.sql /docker-entrypoint-initdb.d/

