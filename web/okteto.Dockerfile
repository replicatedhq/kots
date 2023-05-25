# syntax=docker/dockerfile:1.3
FROM node:18-bullseye-slim as dev
EXPOSE 8080 9229

WORKDIR /src

RUN apt-get update \
  && apt-get install -y --no-install-recommends make \
  && rm -rf /var/lib/apt/lists/*

COPY ./package.json ./yarn.lock Makefile ./
RUN --mount=type=cache,target=/root/.yarn YARN_CACHE_FOLDER=/root/.yarn make deps

COPY . .

FROM dev as builder
ARG OKTETO_NAMESPACE
RUN --mount=type=cache,target=./node_modules/.cache/webpack make build-local

FROM nginx:1.21.4-alpine as nginx
COPY --from=builder /src/dist /usr/share/nginx/html
COPY deploy/nginx.conf /etc/nginx/conf.d/default.conf
EXPOSE 8080