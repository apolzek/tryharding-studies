# Use the official Flink image as a base image
FROM flink:1.20.1

# Set the working directory inside the container
WORKDIR /opt/flink

ENV PYTHONPATH=${PYTHONPATH}:/opt/flink:/opt/flink/core:/opt/flink/jobs

# Install curl (for downloading JARs) and pip (for installing PyFlink)
RUN apt-get update && apt-get install -y curl python3-pip

# Install PyFlink using pip
RUN pip3 install apache-flink pandas joblib scikit-learn==1.2.2 numpy==1.26.4 httpx

RUN ln -s /usr/bin/python3 /usr/bin/python


# Download the necessary Flink connector JARs using curl into /opt/flink/lib
RUN curl -L https://repo1.maven.org/maven2/org/apache/flink/flink-connector-filesystem_2.12/1.11.6/flink-connector-filesystem_2.12-1.11.6.jar -o /opt/flink/lib/flink-connector-filesystem_2.12-1.11.6.jar \
    && curl -L https://repo1.maven.org/maven2/org/apache/flink/flink-connector-jdbc/3.2.0-1.19/flink-connector-jdbc-3.2.0-1.19.jar -o /opt/flink/lib/flink-connector-jdbc-3.2.0-1.19.jar \
    && mkdir -p /opt/flink/plugins/s3-fs-hadoop && curl -L https://repo1.maven.org/maven2/org/apache/flink/flink-s3-fs-hadoop/1.20.1/flink-s3-fs-hadoop-1.20.1.jar -o /opt/flink/plugins/s3-fs-hadoop/flink-s3-fs-hadoop-1.20.1.jar \
    && curl -L https://repo.maven.apache.org/maven2/org/apache/flink/flink-sql-connector-kafka/3.3.0-1.20/flink-sql-connector-kafka-3.3.0-1.20.jar -o /opt/flink/lib/flink-sql-connector-kafka-3.3.0-1.20.jar \
    && curl -L https://repo1.maven.org/maven2/org/apache/flink/flink-avro/1.20.1/flink-avro-1.20.1.jar -o /opt/flink/lib/flink-avro-1.20.1.jar \
    && curl -L https://repo1.maven.org/maven2/org/apache/avro/avro/1.12.0/avro-1.12.0.jar -o /opt/flink/lib/avro-1.12.0.jar \
    && curl -L https://repo1.maven.org/maven2/org/postgresql/postgresql/42.7.4/postgresql-42.7.4.jar -o /opt/flink/lib/postgresql-42.7.4.jar \
    && curl -L https://repo1.maven.org/maven2/org/apache/hadoop/hadoop-common/3.4.1/hadoop-common-3.4.1.jar -o /opt/flink/lib/hadoop-common-3.4.1.jar \
    && curl -L https://repo1.maven.org/maven2/org/apache/hadoop/hadoop-auth/3.4.1/hadoop-auth-3.4.1.jar -o /opt/flink/lib/hadoop-auth-3.4.1.jar \
    && curl -L https://repo1.maven.org/maven2/org/apache/hadoop/hadoop-hdfs/3.4.1/hadoop-hdfs-3.4.1.jar -o /opt/flink/lib/hadoop-hdfs-3.4.1.jar \
    && curl -L https://repo1.maven.org/maven2/org/apache/hadoop/hadoop-aws/3.4.0/hadoop-aws-3.4.0.jar -o /opt/flink/lib/hadoop-aws-3.4.0.jar \
    && curl -L https://repo1.maven.org/maven2/com/amazonaws/aws-java-sdk-bundle/1.12.782/aws-java-sdk-bundle-1.12.782.jar -o /opt/flink/lib/aws-java-sdk-bundle-1.12.782.jar \
    && curl -L https://repo1.maven.org/maven2/software/amazon/awssdk/bundle/2.31.19/bundle-2.31.19.jar -o /opt/flink/lib/bundle-2.31.19.jar

## COPY shell script into image
COPY auto_submit_jobs.sh /opt/flink/auto_submit_jobs.sh
RUN chmod +x /opt/flink/auto_submit_jobs.sh

# Expose necessary ports
EXPOSE 8081 6121 6122 6123 6124 6125

# Set the entrypoint to run Flink with the JobManager and TaskManager as required
ENTRYPOINT ["/docker-entrypoint.sh"]


