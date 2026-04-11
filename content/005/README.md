# Basic log segregation with OpenTelemetry

### Objectives

The goal of this PoC is to implement a granular log routing mechanism in the OpenTelemetry Collector using a routing connector. Incoming logs are ingested from a single file source, parsed, and then dynamically redirected to different storage destinations based on the storage_class attribute (e.g., hot, cold, or other categories). This approach enables fine-grained retention policies per log category, while maintaining flexibility to include or remove attributes before sharing the logs with downstream systems.

### Prerequisites

- docker
- docker compose

### Reproducing

Clean files
```sh
> logs.json && > logs-cold.json && > logs-hot.json
```

Create and start containers
```sh
docker compose up
```

Check 
```sh
jq -s '[.[] | .resourceLogs[]?.scopeLogs[]?.logRecords[]?] | length' logs-hot.json
jq -s '[.[] | .resourceLogs[]?.scopeLogs[]?.logRecords[]?] | length' logs-cold.json
jq -s '[.[] | .resourceLogs[]?.scopeLogs[]?.logRecords[]?] | length' logs.json
```
expected result:
```
3
1
4
```
> using sample_short.log in docker-compose config

### Results

When using the OpenTelemetry Collector, you can combine receivers, processors, exporters, and connectors to build efficient data pipelines and handle data in a straightforward and flexible way. In this example, we use simple local files as the data source, but there is no limitation to exporting logs to real backends like Loki, Splunk, or streaming services such as Kafka. The main challenge is handling large volumes of data. As data and processing rules increase, performance becomes the key factor to consider. Nevertheless, this solution proves to be extremely versatile and effective, a true Swiss Army knife for data collection and routing.

### References

```
🔗 https://github.com/open-telemetry/opentelemetry-collector-contrib
🔗 https://github.com/open-telemetry/opentelemetry-collector-contrib/tree/main/connector/routingconnector
```