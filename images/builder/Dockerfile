FROM python:3.8.2-slim-buster

ARG KUDO_VERSION=0.18.2
ARG KUBECTL_VERSION=1.17.8
ARG KUTTL_VERSION=0.9.0
ARG KUDO_DOWNLOAD_URL=https://github.com/kudobuilder/kudo/releases/download/v${KUDO_VERSION}/kubectl-kudo_${KUDO_VERSION}_linux_x86_64
ARG KUBECTL_DOWNLOAD_URL=https://storage.googleapis.com/kubernetes-release/release/v${KUBECTL_VERSION}/bin/linux/amd64/kubectl
ARG KUTTL_DOWNLOAD_URL=https://github.com/kudobuilder/kuttl/releases/download/v${KUTTL_VERSION}/kubectl-kuttl_${KUTTL_VERSION}_linux_x86_64
ARG JAVA_URL=https://downloads.mesosphere.com/java/openjdk-8u212b03-hotspot-linux-x64.tar.gz
ENV JAVA_HOME=/usr/local/java
ENV PATH=$PATH:${JAVA_HOME}/bin

RUN apt-get update && apt-get install -y curl jq && \
    pip install awscli && \
    curl -L ${KUBECTL_DOWNLOAD_URL} -o /usr/local/bin/kubectl && \
    curl -L ${KUDO_DOWNLOAD_URL} -o /usr/local/bin/kubectl-kudo && \
    curl -L ${KUTTL_DOWNLOAD_URL} -o /usr/local/bin/kubectl-kuttl && \
    for tool in kubectl*; do chmod +x /usr/local/bin/${tool}; done && \
    mkdir -p /usr/local/java && \
    curl -L -O ${JAVA_URL} && tar -xf openjdk-8u212b03-hotspot-linux-x64.tar.gz -C /usr/local/java --strip-components=1
