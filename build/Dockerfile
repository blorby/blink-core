FROM golang AS base

ENV GOPRIVATE=github.com/blinkops

WORKDIR /go/src/github.com/blinkops/blink-core-plugin

COPY go.mod go.sum ./
RUN go mod download
COPY .. .

FROM base AS builder
# Install the package
RUN go install -v ./...

FROM ubuntu AS plugin

WORKDIR /blink-core-plugin
COPY --from=builder /go/bin/blink-core-plugin .
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

RUN chmod a+x blink-core-plugin

ENTRYPOINT ./blink-core-plugin
