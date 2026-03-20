```
docker compose exec clickhouse clickhouse-client --query "SHOW TABLES"
 docker-compose exec clickhouse clickhouse-client --query "SELECT COUNT(*) FROM otel_traces"
```

```
curl -X POST http://localhost:4318/v1/traces \
  -H "Content-Type: application/json" \
  -d '{
    "resourceSpans": [{
      "resource": {
        "attributes": [{
          "key": "service.name",
          "value": {"stringValue": "test-service"}
        }]
      },
      "scopeSpans": [{
        "spans": [{
          "traceId": "1234567890abcdef1234567890abcdef",
          "spanId": "1234567890abcdef",
          "name": "test-span",
          "startTimeUnixNano": "1640995200000000000",
          "endTimeUnixNano": "1640995201000000000"
        }]
      }]
    }]
  }'
```


```
docker volume rm $(docker volume ls -q)
```