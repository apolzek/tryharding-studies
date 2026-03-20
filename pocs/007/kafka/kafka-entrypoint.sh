#!/bin/bash

KAFKA_ADVERTISED_LISTENERS="PLAINTEXT://$(hostname):29092"
export KAFKA_ADVERTISED_LISTENERS
exec /etc/confluent/docker/run
