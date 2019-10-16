FROM golang:1.13.0@sha256:de697ce5ae02f3d9a57b0603fbb648efadfa212727e702ad3a807b43eba7f6d6

RUN curl -LO https://storage.googleapis.com/kubernetes-release/release/v1.16.0/bin/linux/amd64/kubectl && \
    chmod +x ./kubectl && \
    mv ./kubectl /usr/local/bin/kubectl && \
    curl -L https://github.com/kudobuilder/kudo/releases/download/v0.7.3/kubectl-kudo_0.7.3_linux_x86_64 -o /go/bin/kubectl-kudo && \
    chmod +x /go/bin/kubectl-kudo
