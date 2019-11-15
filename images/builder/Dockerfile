FROM golang:1.13.0@sha256:de697ce5ae02f3d9a57b0603fbb648efadfa212727e702ad3a807b43eba7f6d6

ARG KUDO_DOWNLOAD_URL=https://github.com/kudobuilder/kudo/releases/download/v0.8.0/kubectl-kudo_0.8.0_linux_x86_64
ARG KUBECTL_DOWNLOAD_URL=https://storage.googleapis.com/kubernetes-release/release/v1.16.0/bin/linux/amd64/kubectl

RUN curl -LO ${KUBECTL_DOWNLOAD_URL} && \
    chmod +x ./kubectl && \
    mv ./kubectl /usr/local/bin/kubectl && \
    curl -L ${KUDO_DOWNLOAD_URL} -o /go/bin/kubectl-kudo && \
    chmod +x /go/bin/kubectl-kudo && \
    go get github.com/2tvenom/go-test-teamcity
