## How to monitor PostgreSQL running in container

### Objectives

Create an observability ecosystem for PostgreSQL using containers. Ensure visibility into the database and perform load testing. Perhaps a good start would be to be able to answer questions like:

- Which queries are taking the longest to execute ? average time per query (p95/p99 also important)
- What is the usage of server resources (CPU, memory, IOPS, disk) ?
- Are there any locks or blocking operations in the database ?
- Which indexes are missing or could be optimized to improve performance ?
- How is the database connection pool behaving (timeouts, saturation, etc.) ?
- Are there any unusual spikes in read/write activity or error rates ?

### Services and ports

| Service             | Port/Endpoint                                      |
| ------------------- | -------------------------------------------------- |
| PostgreSQL          | postgresql://localhost:5432/app_db?sslmode=disable |
| pgAdmin             | [http://localhost:8080](http://localhost:8080)     |
| PostgreSQL Exporter | 9187                                               |
| Prometheus          | [http://localhost:9090](http://localhost:9090)     |
| Grafana             | [http://localhost:3000](http://localhost:3000)     |
| Loki                | 3100                                               |
| Promtail            | N/A                                                |
| stress_elephant     | 8888                                               |

### Prerequisites

- make
- docker
- docker compose

### Reproducing

Start
```
make start
```

Stop
```
make stop
```

Login database using psql
```sh
docker exec -it $(docker ps | grep postgres_db | awk '{print $1}') bash
psql -h localhost -U rinha -d app_db
```

```bash
docker exec -it postgres_db psql -U rinha -d app_db -c "CREATE EXTENSION postgis;"
docker exec -it postgres_db psql -U rinha -d app_db -c "CREATE EXTENSION pg_stat_statements;"
docker exec -it postgres_db psql -U rinha -d app_db -c "SELECT * FROM pg_extension;"
```

Check extensions
```sql
SELECT * FROM pg_available_extensions;
SELECT * FROM pg_available_extensions WHERE name = 'postgis';
```

Execute demanding queries
```bash
docker exec -it postgres_db psql -U rinha -d app_db -c "
WITH RECURSIVE fibonacci(n, a, b) AS (
    SELECT 1, 0::bigint, 1::bigint
    UNION ALL
    SELECT n + 1, b, a + b
    FROM fibonacci
    WHERE n < 45
)
SELECT n, b as fib_number FROM fibonacci;
"
```

```bash
docker exec -it postgres_db psql -U rinha -d app_db -c "
SELECT 
    a.n * b.n * c.n as produto,
    md5(a.n::text || b.n::text || c.n::text) as hash
FROM 
    generate_series(1, 1000) a(n),
    generate_series(1, 500) b(n),
    generate_series(1, 100) c(n)
WHERE 
    a.n + b.n + c.n = 1000
LIMIT 100;
"
```

```bash
docker exec -it postgres_db psql -U rinha -d app_db -c "
SELECT 
    n,
    regexp_replace(
        repeat('abcdefghijklmnopqrstuvwxyz' || n::text, 100),
        '([a-z]{3})([0-9]+)',
        '\1_\2_processed',
        'g'
    ) as processed_string
FROM generate_series(1, 10000) n
WHERE n % 13 = 0
LIMIT 20;
"
```

### Results

```bash
docker exec -it postgres_db psql -U rinha -d app_db -c "
SELECT 
    substring(query, 1, 80) as query_preview,
    calls,
    round(total_exec_time::numeric, 2) as total_time_ms,
    round(mean_exec_time::numeric, 2) as avg_time_ms,
    round(max_exec_time::numeric, 2) as max_time_ms
FROM pg_stat_statements 
ORDER BY total_exec_time DESC 
LIMIT 5;
"
```

*output*:
```
                                  query_preview                                   | calls | total_time_ms | avg_time_ms | max_time_ms 
----------------------------------------------------------------------------------+-------+---------------+-------------+-------------
 SELECT                                                                          +|     2 |       2127.41 |     1063.71 |     1078.23
     a.n * b.n * c.n as produto,                                                 +|       |               |             | 
     md5(a.n::text || b.n::text || c.n::t                                         |       |               |             | 
 SELECT                                                                          +|     1 |        204.45 |      204.45 |      204.45
     n,                                                                          +|       |               |             | 
     power(n, $1) + sqrt(n) + log(n) as calc1,                                   +|       |               |             | 
     sin(n) * cos(n)                                                              |       |               |             | 
 SELECT pg_database_size($1)                                                      |    92 |         62.58 |        0.68 |        0.86
 SELECT name, setting, COALESCE(unit, $1), short_desc, vartype FROM pg_settings W |    23 |         12.03 |        0.52 |        0.63
 SELECT                                                                          +|    23 |          3.23 |        0.14 |        0.19
                   pg_database.datname as datname,                               +|       |               |             | 
                   tmp.mode as mode,                                             +|       |               |             | 
                   COALESCE(c                                                     |       |               |             | 
```

By completing this lab, we successfully established a containerized observability stack that provided deep visibility into the behavior of a PostgreSQL database. Through Prometheus and its integration with PostgreSQL Exporter, we were able to collect real-time metrics such as query throughput, connection stats, and slow query patterns. This setup not only made it easier to identify performance bottlenecks but also demonstrated how Prometheus works behind the scenes to scrape and store time-series data. Ultimately, the lab reinforced the importance of monitoring as a proactive practice for maintaining healthy, performant databases in containerized environments.

![image](./.image/image-1.png) 
![image](./.image/image-2.png) 
![image](./.image/image-4.png) 
![image](./.image/image-5.png) 

### References

```
🔗 https://medium.com/@shaileshkumarmishra/find-slow-queries-in-postgresql-42dddafc8a0e
🔗 https://mxulises.medium.com/simple-prometheus-setup-on-docker-compose-f702d5f98579
```
