FROM golang:1.17 as builder
WORKDIR /action
COPY . /action

ENV GO111MODULE=auto
ENV CGO_ENABLED=0
RUN go build -ldflags="-w -s" -v -o copy-assets .

FROM alpine:latest
COPY --from=builder /action/copy-assets /copy-assets
ENTRYPOINT ["/copy-assets"]
