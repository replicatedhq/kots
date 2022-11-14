FROM node:slim as builder

WORKDIR /action

COPY package.json package-lock.json ./
RUN npm install

COPY *.js ./
RUN npm run build

FROM node:alpine

COPY --from=builder /action/dist/index.js /index.js

CMD ["node", "index.js"]