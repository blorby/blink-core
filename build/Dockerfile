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
    apt-get install -y python3 python3-apt python3-dev python3-distutils python3-venv curl jq jp unzip git && \
    ln /usr/bin/python3 /usr/bin/python && \
    curl https://bootstrap.pypa.io/get-pip.py | python && \
    pip install -r requirements.txt && \
    curl "https://s3.amazonaws.com/aws-cli/awscli-bundle-1.16.312.zip" -o "awscli-bundle.zip" && \
    unzip awscli-bundle.zip && \
    ./awscli-bundle/install -i /usr/aws -b /usr/bin/aws && \
    chmod +x /usr/bin/aws && \
    rm -rf ./awscli-bundle* && \
    curl "https://amazon-eks.s3.us-west-2.amazonaws.com/1.20.4/2021-04-12/bin/linux/amd64/kubectl" -o "/usr/bin/kubectl" && \
    chmod +x /usr/bin/kubectl && \
    mkdir /repos


# Expose the gRPC port.
EXPOSE 1337

RUN chmod a+x blink-core

ENTRYPOINT ./blink-core
