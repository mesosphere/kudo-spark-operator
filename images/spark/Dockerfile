FROM openjdk:8-alpine

ARG SPARK_HOME=/opt/spark
ARG SPARK_DIST=/dist
ENV SPARK_HOME ${SPARK_HOME}
ARG SPARK_DIST_NAME="spark-2.4.4-bin-hadoop2.7"
ARG SPARK_DIST_URL="http://archive.apache.org/dist/spark/spark-2.4.4/${SPARK_DIST_NAME}.tgz"

RUN set -ex && \
    apk upgrade --no-cache && \
    ln -s /lib /lib64 && \
    apk add --no-cache bash tini libc6-compat linux-pam nss gnupg && \
    mkdir -p ${SPARK_HOME}/work-dir && \
    touch ${SPARK_HOME}/RELEASE && \
    rm /bin/sh && \
    ln -sv /bin/bash /bin/sh && \
    echo "auth required pam_wheel.so use_uid" >> /etc/pam.d/su && \
    chgrp root /etc/passwd && chmod ug+rw /etc/passwd && \
    rm -rf /var/cache/apk/*

RUN mkdir -p ${SPARK_DIST} && \
    cd ${SPARK_DIST} && \
    wget ${SPARK_DIST_URL} && wget ${SPARK_DIST_URL}.sha512 && \
    gpg --print-md sha512 ${SPARK_DIST_NAME}.tgz | diff - ${SPARK_DIST_NAME}.tgz.sha512 && \
    tar xf ${SPARK_DIST_NAME}.tgz -C ${SPARK_DIST} --strip-components=1 && \
    mv jars bin sbin data examples ${SPARK_HOME} && \
    mv kubernetes/dockerfiles/spark/entrypoint.sh /opt && \
    mv kubernetes/tests ${SPARK_HOME} && \
    rm -rf ${SPARK_DIST}

WORKDIR ${SPARK_HOME}

# https://github.com/GoogleCloudPlatform/spark-on-k8s-operator/issues/591
RUN rm jars/kubernetes-client-4.1.2.jar
ADD https://repo1.maven.org/maven2/io/fabric8/kubernetes-client/4.4.2/kubernetes-client-4.4.2.jar jars

# Setup for the Prometheus JMX exporter.
RUN mkdir -p /etc/metrics/conf
# Add the Prometheus JMX exporter Java agent jar for exposing metrics sent to the JmxSink to Prometheus.
ADD https://repo1.maven.org/maven2/io/prometheus/jmx/jmx_prometheus_javaagent/0.11.0/jmx_prometheus_javaagent-0.11.0.jar /prometheus/
# test change
COPY conf/metrics.properties /etc/metrics/conf
COPY conf/prometheus.yaml /etc/metrics/conf

WORKDIR ${SPARK_HOME}/work-dir

ENTRYPOINT [ "/opt/entrypoint.sh" ]
