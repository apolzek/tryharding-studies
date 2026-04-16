## PostgreSQL Replication

### Objectives

To demonstrate asynchronous streaming replication between a PostgreSQL primary and a hot standby replica using Docker Compose. The replica serves as a read-only copy of the primary, providing read scaling and a warm copy of the data for disaster recovery. Note that transparent failover is not part of this setup: the replica accepts only read queries (writes return `cannot execute INSERT in a read-only transaction`), and promoting it to primary requires a manual step or an external tool such as Patroni, repmgr, or pg_auto_failover, typically combined with a connection proxy (HAProxy, PgBouncer).

### Prerequisites

- docker
- docker compose
- postgresql-client(*psql*)

### Reproducing

Up docker compose postgre services
```
cd content/004
docker compose up
```

Install postgresql-client
```
sudo apt install postgresql-client
```
> Debian based distros

#### Replication data

Run psql command to insert data on *postgres_primary*
```sh
psql postgres://user:password@localhost:5432/postgres -xc \
  "CREATE SCHEMA IF NOT EXISTS test_schema;
   CREATE TABLE IF NOT EXISTS test_schema.test_table (
       id SERIAL PRIMARY KEY,
       name VARCHAR(100),
       age INT
   );
   INSERT INTO test_schema.test_table (name, age) 
   VALUES 
   ('João', 30),
   ('Maria', 25),
   ('Pedro', 35),
   ('Ana', 28),
   ('Carlos', 40),
   ('Fernanda', 22),
   ('Lucas', 33),
   ('Beatriz', 29),
   ('Rafael', 31),
   ('Larissa', 27),
   ('Gabriel', 26),
   ('Juliana', 32),
   ('Fernando', 38),
   ('Clara', 24),
   ('Ricardo', 36),
   ('Patrícia', 30),
   ('Daniel', 34),
   ('Camila', 23),
   ('Eduardo', 39),
   ('Júlia', 32),
   ('Sérgio', 29),
   ('Roberta', 26),
   ('Tiago', 33),
   ('Renata', 28),
   ('Vinícius', 40),
   ('Larissa', 25),
   ('Mário', 35),
   ('Joana', 37),
   ('Igor', 30),
   ('Tatiane', 31),
   ('Vitor', 27),
   ('Fernanda', 24),
   ('André', 33),
   ('Mariana', 29),
   ('Natália', 28),
   ('Gustavo', 39),
   ('Isabela', 36),
   ('Robson', 32),
   ('Heloísa', 34),
   ('Amanda', 23),
   ('Maurício', 38),
   ('Simone', 26),
   ('Eduarda', 32),
   ('Juliano', 30),
   ('Marcos', 25),
   ('Rogério', 37),
   ('Camila', 40),
   ('Paulo', 30),
   ('Marcia', 28),
   ('Fernando', 33),
   ('Letícia', 27),
   ('Cláudio', 34),
   ('Sônia', 32),
   ('José', 31),
   ('Vera', 29),
   ('Felipe', 35),
   ('Carla', 30),
   ('Giovana', 38),
   ('Flávia', 24),
   ('Adriana', 39),
   ('Eduardo', 36),
   ('Célia', 32),
   ('Patrícia', 26),
   ('Marcio', 33),
   ('Thiago', 34),
   ('Aline', 30),
   ('Tiago', 37),
   ('Ricardo', 25),
   ('Sabrina', 28),
   ('Ricardo', 35),
   ('Gabriela', 32),
   ('Alessandro', 30),
   ('Rui', 29),
   ('Carolina', 31),
   ('Danilo', 40),
   ('Cássia', 36),
   ('Priscila', 34),
   ('Ricardo', 28),
   ('Natália', 30),
   ('Wagner', 33),
   ('Luiza', 32),
   ('Luciano', 29),
   ('Milena', 37),
   ('Paula', 28),
   ('Fábio', 32),
   ('Jorge', 25),
   ('Cristina', 31),
   ('Igor', 33),
   ('Bárbara', 29),
   ('Cecília', 26),
   ('Renato', 34),
   ('Sônia', 37),
   ('Roberta', 32),
   ('Felipe', 30),
   ('Aline', 28),
   ('Gustavo', 25),
   ('Sérgio', 34),
   ('Jéssica', 33),
   ('Márcia', 40),
   ('Larissa', 39),
   ('Ricardo', 30),
   ('Célia', 32),
   ('Júlia', 25),
   ('Tatiane', 28),
   ('Vítor', 37),
   ('Fábio', 30),
   ('Rogério', 31),
   ('Luciane', 40),
   ('Renato', 29),
   ('Kleber', 26),
   ('Eliane', 35),
   ('Rafaela', 34),
   ('Jorge', 28),
   ('Vera', 32),
   ('Rodrigo', 30),
   ('Thiago', 31),
   ('Marlene', 39),
   ('Douglas', 38),
   ('Mariana', 37);
   
   SELECT * FROM test_schema.test_table;"
```

Checking if data was entered
```
psql postgres://user:password@localhost:5432/postgres -xc \
  "SELECT schema_name
   FROM information_schema.schemata
   WHERE schema_name = 'test_schema';
   
   SELECT table_name
   FROM information_schema.tables
   WHERE table_schema = 'test_schema'
     AND table_name = 'test_table';
   
   SELECT * FROM test_schema.test_table;"
```

Now coming to whether the data was replicated in the *postgres_replica* database
```
psql postgres://user:password@localhost:5433/postgres -c "SELECT * FROM test_schema.test_table"
```

#### Measure replication lag

The correct way to measure replication lag is to ask PostgreSQL itself, not to subtract `now()` from a client-side `created_at`. A client-side diff grows with the time you wait between running the INSERT and the SELECT, so it reflects your typing speed, not the replication pipeline.

From the replica, using the timestamp of the last transaction replayed from WAL:
```
psql postgres://user:password@localhost:5433/postgres -c "
SELECT
  pg_is_in_recovery() AS is_replica,
  pg_last_xact_replay_timestamp() AS last_replay,
  EXTRACT(EPOCH FROM (now() - pg_last_xact_replay_timestamp())) AS lag_seconds;
"
```

Note: `pg_last_xact_replay_timestamp()` returns the time of the last transaction replayed from WAL, so if nothing is being written on the primary the value stays frozen and `lag_seconds` grows. To get a meaningful reading, run an INSERT on the primary and then this query on the replica.

From the primary, using `pg_stat_replication` for per-stage lag and LSN byte distance:
```
psql postgres://user:password@localhost:5432/postgres -c "
SELECT
  client_addr,
  state,
  sync_state,
  write_lag,
  flush_lag,
  replay_lag,
  pg_wal_lsn_diff(pg_current_wal_lsn(), replay_lsn) AS lag_bytes
FROM pg_stat_replication;
"
```

### Results

Replication between primary and replica works as expected: rows inserted on the primary become visible on the replica, and writes attempted against the replica are rejected with `cannot execute INSERT in a read-only transaction`, which is the correct behavior for a hot standby.

Measured on this local Docker setup:

- `pg_stat_replication.replay_lag` on the primary: sub-millisecond (around `00:00:00.0008`), with `lag_bytes = 0` right after a write.
- `now() - pg_last_xact_replay_timestamp()` on the replica immediately after a write: around 50 ms, which mostly reflects the round-trip of issuing the query, not a real backlog.

The previously reported "4 to 6 seconds" figure was an artifact of measuring `now() - created_at` from a client session: that value grows with the wall-clock time between the INSERT and the follow-up SELECT and does not represent replication delay.

### References

```
🔗 https://medium.com/@eremeykin/how-to-setup-single-primary-postgresql-replication-with-docker-compose-98c48f233bbf
```