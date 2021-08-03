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
COPY nodejs nodejs/

RUN apt-get update && \
    apt-get install -y --no-install-recommends ca-certificates curl apt-transport-https lsb-release gnupg && \
    update-ca-certificates && \
    apt-get install -y jq jp unzip \
        # Downloading python3 & pip
        python3 python3-apt python3-dev python3-distutils python3-venv \
        # Downloading nodejs
        nodejs && \
    ln /usr/bin/python3 /usr/bin/python && \
    curl https://bootstrap.pypa.io/get-pip.py | python && \
    pip install -r requirements.txt && \
    apt-get clean && rm -rf /var/lib/apt/lists/*

    # Install aws cli
RUN curl "https://s3.amazonaws.com/aws-cli/awscli-bundle-1.16.312.zip" -o "awscli-bundle.zip" && \
    unzip awscli-bundle.zip && \
    ./awscli-bundle/install -i /usr/aws -b /usr/bin/aws && \
    chmod +x /usr/bin/aws && \
    rm -rf ./awscli-bundle*

    # Install kubectl
# RUN curl "https://amazon-eks.s3.us-west-2.amazonaws.com/1.20.4/2021-04-12/bin/linux/amd64/kubectl" -o "/usr/bin/kubectl" && \
#     chmod +x /usr/bin/kubectl
RUN curl -fsSLo /usr/share/keyrings/kubernetes-archive-keyring.gpg https://packages.cloud.google.com/apt/doc/apt-key.gpg && \
    echo "deb [signed-by=/usr/share/keyrings/kubernetes-archive-keyring.gpg] https://apt.kubernetes.io/ kubernetes-xenial main" | tee /etc/apt/sources.list.d/kubernetes.list && \
    apt-get update && apt-get install -y --no-install-recommends kubectl && \
    apt-get clean && rm -rf /var/lib/apt/lists/*

    # Install azure cli
RUN curl -sL https://packages.microsoft.com/keys/microsoft.asc | gpg --dearmor | tee /etc/apt/trusted.gpg.d/microsoft.gpg > /dev/null && \
    echo "deb [arch=amd64] http://packages.microsoft.com/repos/azure-cli/ $(lsb_release -cs) main" | \
    tee /etc/apt/sources.list.d/azure-cli.list && \
    apt-get update && \
    apt-get install -y --no-install-recommends azure-cli && \
    apt-get clean && rm -rf /var/lib/apt/lists/*

    # Install google cloud cli
RUN echo "deb [signed-by=/usr/share/keyrings/cloud.google.gpg] http://packages.cloud.google.com/apt cloud-sdk main" | \
    tee -a /etc/apt/sources.list.d/google-cloud-sdk.list && \
    curl https://packages.cloud.google.com/apt/doc/apt-key.gpg | \
    apt-key --keyring /usr/share/keyrings/cloud.google.gpg  add - && \
    apt-get update -y && apt-get install -y --no-install-recommends google-cloud-sdk && \
    apt-get clean && rm -rf /var/lib/apt/lists/*


# Expose the gRPC port.
EXPOSE 1337

RUN chmod a+x blink-core

ENTRYPOINT ./blink-core
