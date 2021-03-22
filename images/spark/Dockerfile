FROM ubuntu:18.04

ARG DEBIAN_FRONTEND=noninteractive
ARG TINI_VERSION=v0.18.0
ARG SPARK_REPO="apache/spark"
ARG SPARK_TAG="v3.1.1"
ARG SCALA_VERSION="2.12"
ARG HADOOP_VERSION="2.9.2"

ARG SPARK_BUILD_ARGS="\
    -Pkubernetes \
    -Phadoop-cloud \
    -Pscala-${SCALA_VERSION} \
    -Dhadoop.version=${HADOOP_VERSION} \
    -Pnetlib-lgpl \
    -Psparkr \
    -Phive \
    -Phive-thriftserver \
    -DskipTests \
    -Dmaven.source.skip=true \
    -Dmaven.site.skip=true \
    -Dmaven.javadoc.skip=true"

ENV SPARK_HOME /opt/spark
ENV PYTHONPATH ${SPARK_HOME}/python/lib/pyspark.zip:${SPARK_HOME}/python/lib/py4j-*.zip
ENV R_HOME /usr/lib/R
ENV JAVA_HOME /usr/lib/jvm/java-8-openjdk-amd64
ENV PATH $JAVA_HOME/bin:$PATH

RUN set -ex && \
    apt-get update && \
    ln -s /lib /lib64 && \
    apt-get install --no-install-recommends -y bash libc6 libpam-modules krb5-user libnss3 git openssl curl openjdk-8-jdk \
        r-base r-base-dev python python-pip python3 python3-pip ngrep && \
    curl -L https://github.com/krallin/tini/releases/download/${TINI_VERSION}/tini -o /usr/bin/tini && \
    chmod +x /usr/bin/tini && \
    ln -sv /usr/bin/tini /sbin/tini && \
    mkdir -p ${SPARK_HOME}/work-dir && \
    chmod ugo+rw ${SPARK_HOME}/work-dir && \
    touch ${SPARK_HOME}/RELEASE && \
    rm /bin/sh && \
    ln -sv /bin/bash /bin/sh && \
    echo "auth required pam_wheel.so use_uid" >> /etc/pam.d/su && \
    chgrp root /etc/passwd && chmod ug+rw /etc/passwd && \
    rm -r /usr/lib/python*/ensurepip && \
    pip install --upgrade pip setuptools && \
    rm -rf /var/cache/apt/* /root/.cache

RUN git clone https://github.com/${SPARK_REPO}.git && \
    cd spark && \
    git fetch --tags && \
    git checkout ${SPARK_TAG} && \
    ./dev/make-distribution.sh ${SPARK_BUILD_ARGS} && \
    mv /spark/dist/* ${SPARK_HOME} && \
    rm -rf /spark /root/.m2/repository/*

WORKDIR ${SPARK_HOME}

# Setup for the Prometheus JMX exporter.
RUN mkdir -p /etc/metrics/conf
# Add the Prometheus JMX exporter Java agent jar for exposing metrics sent to the JmxSink to Prometheus.
ADD https://repo1.maven.org/maven2/io/prometheus/jmx/jmx_prometheus_javaagent/0.11.0/jmx_prometheus_javaagent-0.11.0.jar /prometheus/
COPY conf/metrics.properties /etc/metrics/conf
COPY conf/prometheus.yaml /etc/metrics/conf
COPY conf/entrypoint.sh /opt

WORKDIR ${SPARK_HOME}/work-dir

ENTRYPOINT [ "/opt/entrypoint.sh" ]
