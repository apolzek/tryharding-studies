FROM rust:1.85-slim-bookworm AS build
WORKDIR /app
RUN apt-get update && apt-get install -y --no-install-recommends \
    pkg-config protobuf-compiler ca-certificates && rm -rf /var/lib/apt/lists/*

COPY services/product/Cargo.toml services/product/Cargo.toml
COPY services/product/build.rs services/product/build.rs
COPY services/product/src services/product/src
COPY proto proto

WORKDIR /app/services/product
RUN cargo build --release

FROM debian:bookworm-slim
RUN apt-get update && apt-get install -y --no-install-recommends ca-certificates && rm -rf /var/lib/apt/lists/*
COPY --from=build /app/services/product/target/release/product /usr/local/bin/product
EXPOSE 8004 50051
ENTRYPOINT ["/usr/local/bin/product"]
