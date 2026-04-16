# Use a imagem base do PostgreSQL
FROM postgres:17

# Instale as dependências necessárias
RUN apt-get update && apt-get install -y \
    postgresql-17-postgis-3 \
    && rm -rf /var/lib/apt/lists/*

COPY init.sql /docker-entrypoint-initdb.d/
RUN chmod 755 /docker-entrypoint-initdb.d/init.sql
