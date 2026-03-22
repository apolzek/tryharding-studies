

<table>
  <thead>
    <tr>
      <th style="width:40px">id</th>
      <th>title</th>
      <th>tags</th>
    </tr>
  </thead>
  <tbody>
    <tr>
      <td style="text-align:center"><a href="content/009">009</a></td>
      <td>Quick Guide to Chart Types and When to Use Them</td>
      <td><code>grafana</code> <code>visualization</code> <code>charts</code></td>
    </tr>
    <tr>
      <td style="text-align:center"><a href="content/008">008</a></td>
      <td>Understanding the high cardinality problem in Prometheus</td>
      <td><code>prometheus</code> <code>cardinality</code> <code>metrics</code> <code>observability</code></td>
    </tr>
    <tr>
      <td style="text-align:center"><a href="content/007">007</a></td>
      <td>Processing observability data at scale with Apache Flink</td>
      <td><code>flink</code> <code>observability</code> <code>streaming</code> <code>scalability</code></td>
    </tr>
    <tr>
      <td style="text-align:center"><a href="content/006">006</a></td>
      <td>Building a custom Opentelemetry Collector with custom processor</td>
      <td><code>opentelemetry</code> <code>collector</code> <code>golang</code></td>
    </tr>
    <tr>
      <td style="text-align:center"><a href="content/005">005</a></td>
      <td>Basic log segregation with OpenTelemetry using routing connector</td>
      <td><code>opentelemetry</code> <code>logs</code> <code>routing</code></td>
    </tr>
    <tr>
      <td style="text-align:center"><a href="content/004">004</a></td>
      <td>PostgreSQL Data Replication</td>
      <td><code>postgresql</code> <code>replication</code> <code>database</code></td>
    </tr>
    <tr>
      <td style="text-align:center"><a href="content/003">003</a></td>
      <td>How to monitor PostgreSQL running in container</td>
      <td><code>postgresql</code> <code>monitoring</code> <code>docker</code> <code>observability</code></td>
    </tr>
    <tr>
      <td style="text-align:center"><a href="content/002">002</a></td>
      <td>Building a modern observability platform using Prometheus, Grafana, Tempo and OpenTelemetry</td>
      <td><code>observability</code> <code>prometheus</code> <code>grafana</code> <code>opentelemetry</code></td>
    </tr>
    <tr>
      <td style="text-align:center"><a href="content/001">001</a></td>
      <td>Exploring alternatives for load balancing and reverse proxy</td>
      <td><code>loadbalancing</code> <code>nginx</code> <code>haproxy</code> <code>reverseproxy</code></td>
    </tr>
  </tbody>
</table>

<!--
| [005](content/005) | 2025-01-01      | PostgreSQL observability #docker-compose                                    |
| [006](content/006) | 2025-01-01      | Logging with Loki, Fluent Bit and Grafana                                   |
| [007](content/007) | 2025-01-01      | Parsing YAML in Golang                                                      |
| [008](content/008) | 2025-01-01      | Container metrics #docker-compose                                           |
| [009](content/009) | 2025-01-01      | Creating efficient alerts                                                   |
| [010](content/010) | 2025-01-01      | Setting up a local cluster for kubernetes testing #kind                     | -->

<!-- find . -type f -size +10M | grep -v ".git" | sed 's|^\./||' >> .gitignore -->
<!-- echo "Running: $(docker compose ps --services --filter status=running | wc -l) out of $(docker compose ps --services | wc -l)" -->
