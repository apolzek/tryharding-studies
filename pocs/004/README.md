## PostgreSQL Data Replication 

### Objectives

To demonstrate data replication between PostgreSQL primary and replica using Docker Compose, ensuring that if one database instance goes down, the other can be used transparently by the application with minimal impact or downtime

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

#### Calculate *replication_delay*

Create Schema and Table on Primary
```
psql postgres://user:password@localhost:5432/postgres -c "
CREATE SCHEMA IF NOT EXISTS test_schema;
CREATE TABLE IF NOT EXISTS test_schema.xd_table (
  id SERIAL PRIMARY KEY,
  name VARCHAR(100),
  age INT,
  created_at timestamptz DEFAULT now()
);
"
```

Insert Test Record on Primary
```
psql postgres://user:password@localhost:5432/postgres -c "
INSERT INTO test_schema.xd_table (name, age, created_at) VALUES ('Teste Lag', 99, now());
"
```

Check Replication Delay on Secondary
```
psql postgres://user:password@localhost:5433/postgres -c "
SELECT
  name,
  age,
  created_at,
  now() - created_at AS replication_delay
FROM test_schema.xd_table
WHERE name = 'Teste Lag';
"
```

### Results

Based on the tests performed, it was confirmed that replication between the primary and secondary servers is working correctly. The data inserted into the primary database was successfully reflected on the replica server. The *replication_delay* field showed that the average time for changes to propagate to the replica ranged between 4 and 6 seconds, which is considered good performance for a local Docker-based setup. These results validate the effectiveness of the asynchronous replication implemented in this environment.

### References

```
🔗 https://medium.com/@eremeykin/how-to-setup-single-primary-postgresql-replication-with-docker-compose-98c48f233bbf
```