FROM node:8.11

ADD . /src
WORKDIR /src
RUN yarn install --silent

CMD ["make", "deps", "build", "test-and-publish"]
