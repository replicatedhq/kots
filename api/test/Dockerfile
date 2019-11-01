FROM node:10

ADD . /src
WORKDIR /src
RUN yarn install --silent

CMD ["make", "deps", "build", "test-and-publish"]
