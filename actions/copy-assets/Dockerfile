FROM golang:1.16 as builder
WORKDIR /action
COPY . /action

ENV GO111MODULE=auto
ENV CGO_ENABLED=0
RUN go build -ldflags="-w -s" -v -o copy-assets .

FROM gcr.io/distroless/static
COPY --from=builder /action/copy-assets /copy-assets
ENTRYPOINT ["/copy-assets"]