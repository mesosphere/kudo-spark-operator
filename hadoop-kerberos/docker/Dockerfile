FROM centos:7

RUN yum -y install \
    krb5-server \
    krb5-workstation \
    java-1.8.0-openjdk-headless \
    apache-commons-daemon-jsvc \
    net-tools \
    telnet \
    telnet-server \ 
    which

ADD https://www.apache.org/dist/hadoop/core/hadoop-2.9.2/hadoop-2.9.2.tar.gz /

RUN useradd -u 1000 hdfs && \
    ln -s hadoop-2.9.2 hadoop && \
    chown -R -L hdfs:hdfs /hadoop

COPY conf/hadoop /hadoop/etc/hadoop/
COPY conf/krb5.conf /etc/
COPY bin/*  /usr/local/bin/

ENV JAVA_HOME=/usr/lib/jvm/jre-1.8.0-openjdk
ENV PATH=/hadoop/bin:$PATH
ENV HADOOP_CONF_DIR=/hadoop/etc/hadoop
