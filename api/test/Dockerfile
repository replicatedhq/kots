FROM node:10

ADD . /src
WORKDIR /src
RUN npm install --silent

CMD ["make", "deps", "build", "test-and-publish"]
