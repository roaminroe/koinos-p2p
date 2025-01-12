FROM golang:1.18-alpine as builder

ADD . /koinos-p2p
WORKDIR /koinos-p2p

RUN apk update && \
    apk add \
        gcc \
        musl-dev \
        linux-headers

RUN go get ./... && \
    go build -o koinos_p2p cmd/koinos-p2p/main.go

FROM alpine:latest
COPY --from=builder /koinos-p2p/koinos_p2p /usr/local/bin
ENTRYPOINT [ "/usr/local/bin/koinos_p2p" ]
