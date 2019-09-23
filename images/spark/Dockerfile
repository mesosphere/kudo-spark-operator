ARG SPARK_HOME=/opt/spark

FROM alpine:3.10.2 AS spark-dist

ARG SPARK_DIST_URL="http://archive.apache.org/dist/spark/spark-2.4.4/spark-2.4.4-bin-hadoop2.7.tgz"
ARG SPARK_HOME

RUN apk add --no-cache gnupg curl
RUN curl -O ${SPARK_DIST_URL} -O ${SPARK_DIST_URL}.sha512
RUN gpg --print-md sha512 spark-2.4.4-bin-hadoop2.7.tgz | diff - spark-2.4.4-bin-hadoop2.7.tgz.sha512
RUN mkdir -p ${SPARK_HOME} \
    && tar xf spark-2.4.4-bin-hadoop2.7.tgz -C ${SPARK_HOME} --strip-components=1 \
    && rm spark-2.4.4-bin-hadoop2.7.tgz*

FROM openjdk:8-alpine

ARG SPARK_HOME
ENV SPARK_HOME ${SPARK_HOME}

RUN set -ex && \
    apk upgrade --no-cache && \
    ln -s /lib /lib64 && \
    apk add --no-cache bash tini libc6-compat linux-pam nss && \
    mkdir -p /opt/spark && \
    mkdir -p /opt/spark/work-dir && \
    touch /opt/spark/RELEASE && \
    rm /bin/sh && \
    ln -sv /bin/bash /bin/sh && \
    echo "auth required pam_wheel.so use_uid" >> /etc/pam.d/su && \
    chgrp root /etc/passwd && chmod ug+rw /etc/passwd

WORKDIR ${SPARK_HOME}

COPY --from=spark-dist ${SPARK_HOME}/jars jars
COPY --from=spark-dist ${SPARK_HOME}/bin bin
COPY --from=spark-dist ${SPARK_HOME}/sbin sbin
COPY --from=spark-dist ${SPARK_HOME}/data data
COPY --from=spark-dist ${SPARK_HOME}/examples examples
COPY --from=spark-dist ${SPARK_HOME}/kubernetes/dockerfiles/spark/entrypoint.sh /opt/
COPY --from=spark-dist ${SPARK_HOME}/kubernetes/tests tests

# https://github.com/GoogleCloudPlatform/spark-on-k8s-operator/issues/591
RUN rm jars/kubernetes-client-4.1.2.jar
ADD https://repo1.maven.org/maven2/io/fabric8/kubernetes-client/4.4.2/kubernetes-client-4.4.2.jar jars

RUN mkdir -p /prometheus \
    && wget -O /prometheus/jmx_prometheus_javaagent-0.11.0.jar http://central.maven.org/maven2/io/prometheus/jmx/jmx_prometheus_javaagent/0.11.0/jmx_prometheus_javaagent-0.11.0.jar \
    && chmod 777 /prometheus/jmx_prometheus_javaagent-0.11.0.jar

WORKDIR ${SPARK_HOME}/work-dir

ENTRYPOINT [ "/opt/entrypoint.sh" ]
