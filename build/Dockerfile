FROM golang:1.16.3 AS base

ENV GOPRIVATE=github.com/blinkops

WORKDIR /go/src/github.com/blinkops/blink-core

COPY go.mod go.sum ./
RUN go mod download
COPY .. .

FROM base AS builder
# Install the package
RUN go install -v ./...

#  ubuntu 20.04
FROM ubuntu:focal-20210416 AS plugin

WORKDIR /blink-core
COPY --from=builder /go/bin/blink-core .
COPY config.yaml plugin.yaml python/requirements.txt ./
COPY python python/
COPY actions actions/

RUN apt-get update && \
    apt-get install -y ca-certificates && \
    update-ca-certificates && \
    apt-get install -y python3 python3-apt python3-dev python3-distutils curl && \
    ln /usr/bin/python3 /usr/bin/python && \
    curl https://bootstrap.pypa.io/get-pip.py | python && \
    pip install -r requirements.txt

# Expose the gRPC port.
EXPOSE 1337

RUN chmod a+x blink-core

ENTRYPOINT ./blink-core
