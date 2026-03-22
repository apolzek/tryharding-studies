#!/bin/bash

KAFKA_ADVERTISED_LISTENERS="PLAINTEXT://kafka:29092"
export KAFKA_ADVERTISED_LISTENERS
exec /etc/confluent/docker/run
