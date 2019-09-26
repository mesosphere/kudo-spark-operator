FROM golang:1.13.0@sha256:de697ce5ae02f3d9a57b0603fbb648efadfa212727e702ad3a807b43eba7f6d6

RUN curl -LO https://storage.googleapis.com/kubernetes-release/release/v1.16.0/bin/linux/amd64/kubectl && \
    chmod +x ./kubectl && \
    mv ./kubectl /usr/local/bin/kubectl && \
    curl -LO https://get.helm.sh/helm-v2.14.3-linux-amd64.tar.gz -LO https://get.helm.sh/helm-v2.14.3-linux-amd64.tar.gz.sha256 && \
    sha256sum helm-v2.14.3-linux-amd64.tar.gz | awk '{print $1}' | diff - helm-v2.14.3-linux-amd64.tar.gz.sha256 && \
    tar -zxf helm-v2.14.3-linux-amd64.tar.gz && \
    mv linux-amd64/helm /usr/local/bin/helm && \
    rm helm-v2.14.3-linux-amd64.tar.gz*

