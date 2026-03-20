# Use a imagem base do PostgreSQL
FROM postgres:latest

# Instale as dependências necessárias
RUN apt-get update && apt-get install -y \
    postgis \
    && rm -rf /var/lib/apt/lists/*

COPY init.sql /docker-entrypoint-initdb.d/
RUN chmod 755 /docker-entrypoint-initdb.d/init.sql
