ARG SPARK_IMAGE=mesosphere/spark:spark-2.4.3-hadoop-2.9-k8s

FROM golang:1.12.5-alpine as builder
LABEL stage=spark-operator-builder
ARG DEP_VERSION="0.5.3"
RUN apk add --no-cache bash git
ADD https://github.com/golang/dep/releases/download/v${DEP_VERSION}/dep-linux-amd64 /usr/bin/dep
RUN chmod +x /usr/bin/dep

WORKDIR ${GOPATH}/src/github.com/GoogleCloudPlatform/spark-on-k8s-operator
COPY Gopkg.toml Gopkg.lock ./
RUN dep ensure -vendor-only
COPY . ./
RUN go generate && CGO_ENABLED=0 GOOS=linux go build -o /usr/bin/spark-operator

FROM ${SPARK_IMAGE}
COPY --from=builder /usr/bin/spark-operator /usr/bin/
RUN apk add --no-cache openssl curl tini
COPY hack/gencerts.sh /usr/bin/

COPY entrypoint.sh /usr/bin/
ENTRYPOINT ["/usr/bin/entrypoint.sh"]
