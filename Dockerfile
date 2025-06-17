FROM ubuntu:22.04

ENV DEBIAN_FRONTEND=noninteractive

RUN apt-get update && \
    apt-get install -y \
    build-essential \
    git \
    curl \
    wget \
    nano \
    vim

RUN wget https://go.dev/dl/go1.22.0.linux-amd64.tar.gz && \
    tar -C /usr/local -xzf go1.22.0.linux-amd64.tar.gz && \
    rm go1.22.0.linux-amd64.tar.gz

ENV PATH="/usr/local/go/bin:${PATH}"

WORKDIR /app

CMD ["/bin/bash"]
