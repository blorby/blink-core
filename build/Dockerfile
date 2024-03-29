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
COPY config.yaml plugin.yaml python/requirements.txt ./
COPY python python/
COPY nodejs nodejs/

ENV DEBIAN_FRONTEND="noninteractive"

RUN apt-get update && \
    apt-get install -y \
    ca-certificates \
    curl \
    software-properties-common \
    apt-transport-https \
    lsb-release gnupg && \
    update-ca-certificates && \
    apt-get install -y jq jp unzip git && \
    mkdir /opt/blink && \
    mv /usr/bin/git /opt/blink

# Downloading python3 & pip
RUN apt-get update && \
    apt-get install -y python3 python3-apt python3-dev python3-distutils python3-venv && \
    ln /usr/bin/python3 /usr/bin/python && \
    curl https://bootstrap.pypa.io/get-pip.py | python && \
    pip install -r requirements.txt

# Downloading nodejs & npm
RUN apt-get update && \
    apt-get install -y nodejs && \
    apt-get install -y npm

# Downloading aws cli
RUN curl "https://awscli.amazonaws.com/awscli-exe-linux-x86_64-2.2.32.zip" -o awscliv2.zip && \
    unzip awscliv2.zip && \
    ./aws/install && \
    rm -rf ./aws awscliv2.zip && \
    mv /usr/local/bin/aws /opt/blink

# Downloading eksctl
RUN curl --silent --location "https://github.com/weaveworks/eksctl/releases/latest/download/eksctl_$(uname -s)_amd64.tar.gz" | tar xz -C /tmp && \
    mv /tmp/eksctl /opt/blink

# Downloading kubectl
RUN curl "https://amazon-eks.s3.us-west-2.amazonaws.com/1.20.4/2021-04-12/bin/linux/amd64/kubectl" -o "/opt/blink/kubectl" && \
    chmod +x /opt/blink/kubectl


# Download google cloud cli
RUN echo "deb [signed-by=/usr/share/keyrings/cloud.google.gpg] http://packages.cloud.google.com/apt cloud-sdk main" | \
    tee -a /etc/apt/sources.list.d/google-cloud-sdk.list && \
    curl https://packages.cloud.google.com/apt/doc/apt-key.gpg | \
    apt-key --keyring /usr/share/keyrings/cloud.google.gpg  add - && \
    apt-get update -y && \
    apt-get install -y google-cloud-sdk && \
    rm /usr/bin/gcloud && \
    ln -s /usr/lib/google-cloud-sdk/bin/gcloud /opt/blink/gcloud

# Downloading azure cli
RUN curl -sL https://packages.microsoft.com/keys/microsoft.asc | \
    gpg --dearmor | \
    tee /etc/apt/trusted.gpg.d/microsoft.gpg > /dev/null && \
    echo "deb [arch=amd64] http://packages.microsoft.com/repos/azure-cli/ $(lsb_release -cs) main" | \
    tee /etc/apt/sources.list.d/azure-cli.list && \
    apt-get update -y && \
    apt-get install -y azure-cli && \
    mv /usr/bin/az /opt/blink/

## Install sudo
RUN apt-get install sudo

## Install Vault and TerraForm cli.
RUN curl -fsSL https://apt.releases.hashicorp.com/gpg | apt-key add -
RUN apt-get update && \
    apt-add-repository "deb [arch=amd64] https://apt.releases.hashicorp.com $(lsb_release -cs) main" && \
    apt-get install vault && \
    apt-get install terraform && \
    rm -rf /var/lib/apt/lists/* && \
    setcap cap_ipc_lock= /usr/bin/vault && \
    mv /usr/bin/vault /opt/blink/ && \
    mv /usr/bin/terraform /opt/blink/

## Install groff to support aws cli.
RUN apt-get update && \
    apt-get install groff-base -y

# keep this at the end to make better use of caching
COPY --from=builder /go/bin/blink-core .
COPY actions actions/

# copy sudo cli commands
COPY bash /usr/local/bin
RUN chmod +x /usr/local/bin/*

RUN groupadd core

# Expose the gRPC port.
EXPOSE 1337
RUN chmod a+x blink-core
ENTRYPOINT ./blink-core
